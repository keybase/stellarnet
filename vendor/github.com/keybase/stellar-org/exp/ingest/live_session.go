package ingest

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
)

var _ Session = &LiveSession{}

const defaultCoreCursorName = "EXPINGESTLIVESESSION"

func (s *LiveSession) Run() error {
	s.standardSession.shutdown = make(chan bool)

	err := s.validate()
	if err != nil {
		return errors.Wrap(err, "Validation error")
	}

	s.setRunningState(true)
	defer s.setRunningState(false)

	historyAdapter := adapters.MakeHistoryArchiveAdapter(s.Archive)
	currentLedger, err := historyAdapter.GetLatestLedgerSequence()
	if err != nil {
		return errors.Wrap(err, "Error getting the latest ledger sequence")
	}

	// Update cursor
	err = s.updateCursor(currentLedger)
	if err != nil {
		return errors.Wrap(err, "Error setting cursor")
	}

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	// Validate bucket list hash
	err = s.validateBucketList(currentLedger, historyAdapter, ledgerAdapter)
	if err != nil {
		return errors.Wrap(err, "Error validating bucket list hash")
	}

	err = s.initState(historyAdapter, currentLedger)
	if err != nil {
		return errors.Wrap(err, "initState error")
	}

	s.latestSuccessfullyProcessedLedger = currentLedger

	// Exit early if Shutdown() was called.
	select {
	case <-s.standardSession.shutdown:
		return nil
	default:
		// Continue
	}

	// `currentLedger` is incremented because applied state is AFTER the
	// current value of `currentLedger`
	currentLedger++

	return s.resume(currentLedger, ledgerAdapter)
}

// GetArchive returns the archive configured for the current session
func (s *LiveSession) GetArchive() historyarchive.ArchiveInterface {
	return s.Archive
}

func (s *LiveSession) updateCursor(ledgerSequence uint32) error {
	if s.StellarCoreClient == nil {
		return nil
	}

	cursor := defaultCoreCursorName
	if s.StellarCoreCursor != "" {
		cursor = s.StellarCoreCursor
	}

	err := s.StellarCoreClient.SetCursor(context.Background(), cursor, int32(ledgerSequence))
	if err != nil {
		return errors.Wrap(err, "Setting stellar-core cursor failed")
	}

	return nil
}

// Resume resumes the session from `ledgerSequence`.
//
// WARNING: it's likely that developers will use `GetLatestSuccessfullyProcessedLedger()`
// to get the latest successfuly processed ledger after `Resume` returns error.
// It's critical to understand that `GetLatestSuccessfullyProcessedLedger()` will
// return `(0, false)` when no ledgers have been successfully processed, ex.
// error while trying to process a ledger after application restart.
// You should always check if the second returned value is equal `false` before
// overwriting your local variable.
func (s *LiveSession) Resume(ledgerSequence uint32) error {
	s.standardSession.shutdown = make(chan bool)

	ledgerAdapter := &adapters.LedgerBackendAdapter{
		Backend: s.LedgerBackend,
	}

	return s.resume(ledgerSequence, ledgerAdapter)
}

// validateBucketList validates if the bucket list hash in history archive
// matches the one in corresponding ledger header in stellar-core backend.
// This gives you full security if data in stellar-core backend can be trusted
// (ex. you run it in your infrastructure).
// The hashes of actual buckets of this HAS file are checked using
// historyarchive.XdrStream.SetExpectedHash (this is done in
// SingleLedgerStateReader).
func (s *LiveSession) validateBucketList(
	ledgerSequence uint32,
	historyAdapter *adapters.HistoryArchiveAdapter,
	ledgerAdapter *adapters.LedgerBackendAdapter,
) error {
	historyBucketListHash, err := historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	ledgerReader, err := ledgerAdapter.GetLedger(ledgerSequence)
	if err != nil {
		if err == io.ErrNotFound {
			return fmt.Errorf(
				"Cannot validate bucket hash list. Checkpoint ledger (%d) must exist in Stellar-Core database.",
				ledgerSequence,
			)
		} else {
			return errors.Wrap(err, "Error getting ledger")
		}
	}

	ledgerHeader := ledgerReader.GetHeader()
	ledgerBucketHashList := ledgerHeader.Header.BucketListHash

	if !bytes.Equal(historyBucketListHash[:], ledgerBucketHashList[:]) {
		return fmt.Errorf(
			"Bucket list hash of history archive and ledger header does not match: %#x %#x",
			historyBucketListHash,
			ledgerBucketHashList,
		)
	}

	return nil
}

func (s *LiveSession) resume(ledgerSequence uint32, ledgerAdapter *adapters.LedgerBackendAdapter) error {
	for {
		ledgerReader, err := ledgerAdapter.GetLedger(ledgerSequence)
		if err != nil {
			if err == io.ErrNotFound {
				// Ensure that there are no gaps. This is "just in case". There shouldn't
				// be any gaps if CURSOR in core is updated and core version is v11.2.0+.
				var latestLedger uint32
				latestLedger, err = ledgerAdapter.GetLatestLedgerSequence()
				if err != nil {
					return err
				}

				if latestLedger > ledgerSequence {
					return errors.Errorf("Gap detected (ledger %d does not exist but %d is latest)", ledgerSequence, latestLedger)
				}

				select {
				case <-s.standardSession.shutdown:
					s.LedgerPipeline.Shutdown()
					return nil
				case <-time.After(time.Second):
					// TODO make the idle time smaller
				}

				continue
			}

			return errors.Wrap(err, "Error getting ledger")
		}

		if s.LedgerReporter != nil {
			s.LedgerReporter.OnNewLedger(ledgerSequence)
			ledgerReader = reporterLedgerReader{ledgerReader, s.LedgerReporter}
		}

		errChan := s.LedgerPipeline.Process(ledgerReader)
		select {
		case err2 := <-errChan:
			if err2 != nil {
				if s.LedgerReporter != nil {
					s.LedgerReporter.OnEndLedger(err2, false)
				}
				return errors.Wrap(err2, "Ledger pipeline errored")
			}
		case <-s.standardSession.shutdown:
			if s.LedgerReporter != nil {
				s.LedgerReporter.OnEndLedger(nil, true)
			}
			s.LedgerPipeline.Shutdown()
			return nil
		}

		if s.LedgerReporter != nil {
			s.LedgerReporter.OnEndLedger(nil, false)
		}
		s.latestSuccessfullyProcessedLedger = ledgerSequence

		// Update cursor - this needs to be done after `latestSuccessfullyProcessedLedger`
		// is updated as the cursor update shouldn't affect this value.
		err = s.updateCursor(ledgerSequence)
		if err != nil {
			return errors.Wrap(err, "Error setting cursor")
		}

		ledgerSequence++
	}

	return nil
}

// GetLatestSuccessfullyProcessedLedger returns the last SUCCESSFULLY processed
// ledger. Returns (0, false) if no ledgers have been successfully processed yet
// to prevent situations where `GetLatestSuccessfullyProcessedLedger()` value is
// not properly checked in a loop resulting in ingesting ledger 0+1=1.
// Please check `Resume` godoc to understand possible implications.
func (s *LiveSession) GetLatestSuccessfullyProcessedLedger() (ledgerSequence uint32, processed bool) {
	if s.latestSuccessfullyProcessedLedger == 0 {
		return 0, false
	}
	return s.latestSuccessfullyProcessedLedger, true
}

func (s *LiveSession) validate() error {
	switch {
	case s.Archive == nil:
		return errors.New("Archive not set")
	case s.LedgerBackend == nil:
		return errors.New("LedgerBackend not set")
	case s.StatePipeline == nil:
		return errors.New("State pipeline not set")
	case s.LedgerPipeline == nil:
		return errors.New("Ledger pipeline not set")
	}

	return nil
}

func (s *LiveSession) initState(historyAdapter *adapters.HistoryArchiveAdapter, sequence uint32) error {
	var tempSet io.TempSet = &io.MemoryTempSet{}
	if s.TempSet != nil {
		tempSet = s.TempSet
	}

	stateReader, err := historyAdapter.GetState(sequence, tempSet)
	if err != nil {
		return errors.Wrap(err, "Error getting state from history archive")
	}
	if s.StateReporter != nil {
		s.StateReporter.OnStartState(sequence)
		stateReader = reporterStateReader{stateReader, s.StateReporter}
	}

	errChan := s.StatePipeline.Process(stateReader)
	select {
	case err := <-errChan:
		if err != nil {
			if s.StateReporter != nil {
				s.StateReporter.OnEndState(err, false)
			}
			return errors.Wrap(err, "State pipeline errored")
		}
	case <-s.standardSession.shutdown:
		if s.StateReporter != nil {
			s.StateReporter.OnEndState(nil, true)
		}
		s.StatePipeline.Shutdown()
	}

	if s.StateReporter != nil {
		s.StateReporter.OnEndState(nil, false)
	}
	return nil
}
