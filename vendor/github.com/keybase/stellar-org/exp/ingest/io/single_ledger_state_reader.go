package io

import (
	"context"
	"encoding/base64"
	"io"
	"sync"
	"time"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// readResult is the result of reading a bucket value
type readResult struct {
	entryChange xdr.LedgerEntryChange
	e           error
}

// SingleLedgerStateReader is a streaming implementation that reads ledger entries
// from buckets for a given HistoryArchiveState (single ledger/checkpoint).
// SingleLedgerStateReader hides internal structure of buckets from the user so
// entries returned by `Read()` are exactly the ledger entries present at the given
// ledger.
type SingleLedgerStateReader struct {
	ctx        context.Context
	has        *historyarchive.HistoryArchiveState
	archive    historyarchive.ArchiveInterface
	tempStore  tempSet
	sequence   uint32
	readChan   chan readResult
	streamOnce sync.Once
	closeOnce  sync.Once
	done       chan bool
	// how many times should we retry when there are errors in
	// the xdr stream returned by GetXdrStreamForHash()
	maxStreamRetries int

	// This should be set to true in tests only
	disableBucketListHashValidation bool
	sleep                           func(time.Duration)
}

// Ensure SingleLedgerStateReader implements ChangeReader
var _ ChangeReader = &SingleLedgerStateReader{}

// tempSet is an interface that must be implemented by stores that
// hold temporary set of objects for state reader. The implementation
// does not need to be thread-safe.
type tempSet interface {
	Open() error
	// Preload batch-loads keys into internal cache (if a store has any) to
	// improve execution time by removing many round-trips.
	Preload(keys []string) error
	// Add adds key to the store.
	Add(key string) error
	// Exist returns value true if the value is found in the store.
	// If the value has not been set, it should return false.
	Exist(key string) (bool, error)
	Close() error
}

const (
	msrBufferSize = 50000

	// preloadedEntries defines a number of bucket entries to preload from a
	// bucket in a single run. This is done to allow preloading keys from
	// temp set.
	preloadedEntries = 20000

	sleepDuration = time.Second
)

// MakeSingleLedgerStateReader is a factory method for SingleLedgerStateReader.
// `maxStreamRetries` determines how many times the reader will retry when encountering
// errors while streaming xdr bucket entries from the history archive.
// Set `maxStreamRetries` to 0 if there should be no retry attempts
func MakeSingleLedgerStateReader(
	ctx context.Context,
	archive historyarchive.ArchiveInterface,
	sequence uint32,
	maxStreamRetries int,
) (*SingleLedgerStateReader, error) {
	has, err := archive.GetCheckpointHAS(sequence)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get checkpoint HAS at ledger sequence %d", sequence)
	}

	tempStore := &memoryTempSet{}
	err = tempStore.Open()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get open temp store")
	}

	return &SingleLedgerStateReader{
		ctx:              ctx,
		has:              &has,
		archive:          archive,
		tempStore:        tempStore,
		sequence:         sequence,
		readChan:         make(chan readResult, msrBufferSize),
		streamOnce:       sync.Once{},
		closeOnce:        sync.Once{},
		done:             make(chan bool),
		maxStreamRetries: maxStreamRetries,
		sleep:            time.Sleep,
	}, nil
}

func (msr *SingleLedgerStateReader) bucketExists(hash historyarchive.Hash) (bool, error) {
	duration := sleepDuration
	var exists bool
	var err error
	for attempts := 0; ; attempts++ {
		exists, err = msr.archive.BucketExists(hash)
		if err == nil {
			return exists, nil
		}
		if attempts >= msr.maxStreamRetries {
			break
		}
		msr.sleep(duration)
		duration *= 2
	}
	return exists, err
}

// streamBuckets is internal method that streams buckets from the given HAS.
//
// Buckets should be processed from oldest to newest, `snap` and then `curr` at
// each level. The correct value of ledger entry is the latest seen
// `INITENTRY`/`LIVEENTRY` except the case when there's a `DEADENTRY` later
// which removes the entry.
//
// We can implement trivial algorithm (processing from oldest to newest buckets)
// but it requires to keep map of all entries in memory and stream what's left
// when all buckets are processed.
//
// However, we can modify this algorithm to work from newest to oldest ledgers:
//
//   1. For each `INITENTRY`/`LIVEENTRY` we check if we've seen the key before
//      (stored in `tempStore`). If the key hasn't been seen, we write that bucket
//      entry to the stream and add it to the `tempStore` (we don't mark `INITENTRY`,
//      see the inline comment or CAP-20).
//   2. For each `DEADENTRY` we keep track of removed bucket entries in
//      `tempStore` map.
//
// In such algorithm we just need to store a set of keys that require much less space.
// The memory requirements will be lowered when CAP-0020 is live and older buckets are
// rewritten. Then, we will only need to keep track of `DEADENTRY`.
func (msr *SingleLedgerStateReader) streamBuckets() {
	defer func() {
		err := msr.tempStore.Close()
		if err != nil {
			msr.readChan <- msr.error(errors.New("Error closing tempStore"))
		}

		msr.closeOnce.Do(msr.close)
		close(msr.readChan)
	}()

	var buckets []historyarchive.Hash
	for i := 0; i < len(msr.has.CurrentBuckets); i++ {
		b := msr.has.CurrentBuckets[i]
		for _, hashString := range []string{b.Curr, b.Snap} {
			hash, err := historyarchive.DecodeHash(hashString)
			if err != nil {
				msr.readChan <- msr.error(errors.Wrap(err, "Error decoding bucket hash"))
				return
			}

			if hash.IsZero() {
				continue
			}

			buckets = append(buckets, hash)
		}
	}

	for i, hash := range buckets {
		exists, err := msr.bucketExists(hash)
		if err != nil {
			msr.readChan <- msr.error(
				errors.Wrapf(err, "error checking if bucket exists: %s", hash),
			)
			return
		}

		if !exists {
			msr.readChan <- msr.error(
				errors.Errorf("bucket hash does not exist: %s", hash),
			)
			return
		}

		oldestBucket := i == len(buckets)-1
		if shouldContinue := msr.streamBucketContents(hash, oldestBucket); !shouldContinue {
			break
		}
	}
}

// readBucketEntry will attempt to read a bucket entry from `stream`.
// If any errors are encountered while reading from `stream`, readBucketEntry will
// retry the operation using a new *historyarchive.XdrStream.
// The total number of retries will not exceed `maxStreamRetries`.
func (msr *SingleLedgerStateReader) readBucketEntry(stream *historyarchive.XdrStream, hash historyarchive.Hash) (
	xdr.BucketEntry,
	error,
) {
	var entry xdr.BucketEntry
	var err error
	currentPosition := stream.BytesRead()

	for attempts := 0; ; attempts++ {
		if msr.ctx.Err() != nil {
			err = msr.ctx.Err()
			break
		}
		if err == nil {
			err = stream.ReadOne(&entry)
			if err == nil || err == io.EOF {
				break
			}
		}
		if attempts >= msr.maxStreamRetries {
			break
		}

		stream.Close()

		var retryStream *historyarchive.XdrStream
		retryStream, err = msr.newXDRStream(hash)
		if err != nil {
			err = errors.Wrap(err, "Error creating new xdr stream")
			continue
		}

		*stream = *retryStream

		_, err = stream.Discard(currentPosition)
		if err != nil {
			err = errors.Wrap(err, "Error discarding from xdr stream")
			continue
		}
	}

	return entry, err
}

func (msr *SingleLedgerStateReader) newXDRStream(hash historyarchive.Hash) (
	*historyarchive.XdrStream,
	error,
) {
	rdr, e := msr.archive.GetXdrStreamForHash(hash)
	if e == nil && !msr.disableBucketListHashValidation {
		// Calling SetExpectedHash will enable validation of the stream hash. If hashes
		// don't match, rdr.Close() will return an error.
		rdr.SetExpectedHash(hash)
	}

	return rdr, e
}

// streamBucketContents pushes value onto the read channel, returning false when the channel needs to be closed otherwise true
func (msr *SingleLedgerStateReader) streamBucketContents(hash historyarchive.Hash, oldestBucket bool) bool {
	rdr, e := msr.newXDRStream(hash)
	if e != nil {
		msr.readChan <- msr.error(
			errors.Wrapf(e, "cannot get xdr stream for hash '%s'", hash.String()),
		)
		return false
	}

	defer func() {
		err := rdr.Close()
		if err != nil {
			msr.readChan <- msr.error(errors.Wrap(err, "Error closing xdr stream"))
			// Stop streaming from the rest of the files.
			msr.Close()
		}
	}()

	// bucketProtocolVersion is a protocol version read from METAENTRY or 0 when no METAENTRY.
	// No METAENTRY means that bucket originates from before protocol version 11.
	bucketProtocolVersion := uint32(0)

	n := -1
	var batch []xdr.BucketEntry
	lastBatch := false

LoopBucketEntry:
	for {
		// Preload entries for faster retrieve from temp store.
		if len(batch) == 0 {
			if lastBatch {
				return true
			}

			preloadKeys := []string{}

			for i := 0; i < preloadedEntries; i++ {
				var entry xdr.BucketEntry
				entry, e = msr.readBucketEntry(rdr, hash)
				if e != nil {
					if e == io.EOF {
						if len(batch) == 0 {
							// No entries loaded for this batch, nothing more to process
							return true
						}
						lastBatch = true
						break
					}
					msr.readChan <- msr.error(
						errors.Wrapf(e, "Error on XDR record %d of hash '%s'", n, hash.String()),
					)
					return false
				}

				batch = append(batch, entry)

				// Generate a key
				var key xdr.LedgerKey

				switch entry.Type {
				case xdr.BucketEntryTypeLiveentry, xdr.BucketEntryTypeInitentry:
					liveEntry := entry.MustLiveEntry()
					key = liveEntry.LedgerKey()
				case xdr.BucketEntryTypeDeadentry:
					key = entry.MustDeadEntry()
				default:
					// No ledger key associated with this entry, continue to the next one.
					continue
				}

				// We're using compressed keys here
				keyBytes, e := key.MarshalBinaryCompress()
				if e != nil {
					msr.readChan <- msr.error(
						errors.Wrapf(e, "Error marshaling XDR record %d of hash '%s'", n, hash.String()),
					)
					return false
				}

				h := base64.StdEncoding.EncodeToString(keyBytes)
				preloadKeys = append(preloadKeys, h)
			}

			err := msr.tempStore.Preload(preloadKeys)
			if err != nil {
				msr.readChan <- msr.error(errors.Wrap(err, "Error preloading keys"))
				return false
			}
		}

		var entry xdr.BucketEntry
		entry, batch = batch[0], batch[1:]

		n++

		var key xdr.LedgerKey

		switch entry.Type {
		case xdr.BucketEntryTypeMetaentry:
			if n != 0 {
				msr.readChan <- msr.error(
					errors.Errorf(
						"METAENTRY not the first entry (n=%d) in the bucket hash '%s'",
						n, hash.String(),
					),
				)
				return false
			}
			// We can't use MustMetaEntry() here. Check:
			// https://github.com/golang/go/issues/32560
			bucketProtocolVersion = uint32(entry.MetaEntry.LedgerVersion)
			continue LoopBucketEntry
		case xdr.BucketEntryTypeLiveentry, xdr.BucketEntryTypeInitentry:
			liveEntry := entry.MustLiveEntry()
			key = liveEntry.LedgerKey()
		case xdr.BucketEntryTypeDeadentry:
			key = entry.MustDeadEntry()
		default:
			msr.readChan <- msr.error(
				errors.Errorf("Unknown BucketEntryType=%d: %d@%s", entry.Type, n, hash.String()),
			)
			return false
		}

		// We're using compressed keys here
		keyBytes, e := key.MarshalBinaryCompress()
		if e != nil {
			msr.readChan <- msr.error(
				errors.Wrapf(
					e, "Error marshaling XDR record %d of hash '%s'", n, hash.String(),
				),
			)
			return false
		}

		h := base64.StdEncoding.EncodeToString(keyBytes)

		switch entry.Type {
		case xdr.BucketEntryTypeLiveentry, xdr.BucketEntryTypeInitentry:
			if entry.Type == xdr.BucketEntryTypeInitentry && bucketProtocolVersion < 11 {
				msr.readChan <- msr.error(
					errors.Errorf("Read INITENTRY from version <11 bucket: %d@%s", n, hash.String()),
				)
				return false
			}

			seen, err := msr.tempStore.Exist(h)
			if err != nil {
				msr.readChan <- msr.error(errors.Wrap(err, "Error reading from tempStore"))
				return false
			}

			if !seen {
				// Return LEDGER_ENTRY_STATE changes only now.
				liveEntry := entry.MustLiveEntry()
				entryChange := xdr.LedgerEntryChange{
					Type:  xdr.LedgerEntryChangeTypeLedgerEntryState,
					State: &liveEntry,
				}
				msr.readChan <- readResult{entryChange, nil}

				// We don't update `tempStore` for INITENTRY because CAP-20 says:
				// > a bucket entry marked INITENTRY implies that either no entry
				// > with the same ledger key exists in an older bucket, or else
				// > that the (chronologically) preceding entry with the same ledger
				// > key was DEADENTRY.
				if entry.Type == xdr.BucketEntryTypeLiveentry {
					// We skip adding entries from the last bucket to tempStore because:
					// 1. Ledger keys are unique within a single bucket.
					// 2. This is the last bucket we process so there's no need to track
					//    seen last entries in this bucket.
					if oldestBucket {
						continue
					}
					err := msr.tempStore.Add(h)
					if err != nil {
						msr.readChan <- msr.error(errors.Wrap(err, "Error updating to tempStore"))
						return false
					}
				}
			}
		case xdr.BucketEntryTypeDeadentry:
			err := msr.tempStore.Add(h)
			if err != nil {
				msr.readChan <- msr.error(errors.Wrap(err, "Error writing to tempStore"))
				return false
			}
		default:
			msr.readChan <- msr.error(
				errors.Errorf("Unexpected entry type %d: %d@%s", entry.Type, n, hash.String()),
			)
			return false
		}

		select {
		case <-msr.done:
			// Close() called: stop processing buckets.
			return false
		default:
			continue
		}
	}

	panic("Shouldn't happen")
}

// Read returns a new ledger entry change on each call, returning io.EOF when the stream ends.
func (msr *SingleLedgerStateReader) Read() (Change, error) {
	msr.streamOnce.Do(func() {
		go msr.streamBuckets()
	})

	// blocking call. anytime we consume from this channel, the background goroutine will stream in the next value
	result, ok := <-msr.readChan
	if !ok {
		// when channel is closed then return io.EOF
		return Change{}, io.EOF
	}

	if result.e != nil {
		return Change{}, errors.Wrap(result.e, "Error while reading from buckets")
	}
	return Change{
		Type: result.entryChange.EntryType(),
		Post: result.entryChange.State,
	}, nil
}

func (msr *SingleLedgerStateReader) error(err error) readResult {
	return readResult{xdr.LedgerEntryChange{}, err}
}

func (msr *SingleLedgerStateReader) close() {
	close(msr.done)
}

// Close should be called when reading is finished.
func (msr *SingleLedgerStateReader) Close() error {
	msr.closeOnce.Do(msr.close)
	return nil
}
