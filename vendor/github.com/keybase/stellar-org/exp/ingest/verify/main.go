// Package verify provides helpers used for verifying if the ingested data is
// correct.
package verify

import (
	"bytes"
	"encoding/base64"
	stdio "io"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TransformLedgerEntryFunction is a function that transforms ledger entry
// into a form that should be compared to checkpoint state. It can be also used
// to decide if the given entry should be ignored during verification.
// Sometimes the application needs only specific type entries or specific fields
// for a given entry type. Use this function to create a common form of an entry
// that will be used for equality check.
type TransformLedgerEntryFunction func(xdr.LedgerEntry) (ignore bool, newEntry xdr.LedgerEntry)

// StateError are errors indicating invalid state. Type is used to differentiate
// between network, i/o, marshaling, bad usage etc. errors and actual state errors.
// You can use type assertion or type switch to check for type.
type StateError struct {
	error
}

// NewStateError creates a new StateError.
func NewStateError(err error) StateError {
	return StateError{err}
}

// StateVerifier verifies if ledger entries provided by Add method are the same
// as in the checkpoint ledger entries provided by SingleLedgerStateReader.
// The algorithm works in the following way:
//   0. Develop TransformFunction. It should remove all fields and objects not
//      stored in your app. For example, if you only store accounts, all other
//      ledger entry types should be ignored (return ignore = true).
//   1. In a loop, get entries from history archive by calling GetEntries()
//      and Write() your version of entries found in the batch (in any order).
//   2. When GetEntries() return no more entries, call Verify with a number of
//      entries in your storage (to find if some extra entires exist in your
//      storage).
// Functions will return StateError type if state is found to be incorrect.
// It's user responsibility to call `StateReader.Close()` when reading is done.
// Check Horizon for an example how to use this tool.
type StateVerifier struct {
	StateReader io.StateReader
	// TransformFunction transforms (or ignores) ledger entries streamed from
	// checkpoint buckets to match the form added by `Write`. Read
	// TransformLedgerEntryFunction godoc for more information.
	TransformFunction TransformLedgerEntryFunction

	readEntries int
	readingDone bool

	currentEntries map[string]xdr.LedgerEntry
}

// GetLedgerKeys returns up to `count` ledger keys from history buckets
// storing actual entries in cache to compare in Write.
func (v *StateVerifier) GetLedgerKeys(count int) ([]xdr.LedgerKey, error) {
	err := v.checkUnreadEntries()
	if err != nil {
		return nil, err
	}

	keys := make([]xdr.LedgerKey, 0, count)
	v.currentEntries = make(map[string]xdr.LedgerEntry)

	for count > 0 {
		entryChange, err := v.StateReader.Read()
		if err != nil {
			if err == stdio.EOF {
				v.readingDone = true
				return keys, nil
			}
			return keys, err
		}

		entry := entryChange.MustState()

		if v.TransformFunction != nil {
			ignore, _ := v.TransformFunction(entry)
			if ignore {
				continue
			}
		}

		ledgerKey := entry.LedgerKey()
		key, err := xdr.MarshalBase64(ledgerKey)
		if err != nil {
			return keys, errors.Wrap(err, "Error marshaling ledgerKey")
		}

		keys = append(keys, ledgerKey)
		v.currentEntries[key] = entry

		count--
		v.readEntries++
	}

	return keys, nil
}

// Write compares the entry with entries in the latest batch of entries fetched
// using `GetEntries`. Entries don't need to follow the order in entries returned
// by `GetEntries`.
// Any `StateError` returned by this method indicates invalid state!
func (v *StateVerifier) Write(entry xdr.LedgerEntry) error {
	actualEntry := entry
	actualEntryMarshaled, err := actualEntry.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "Error marshaling actualEntry")
	}

	key, err := xdr.MarshalBase64(actualEntry.LedgerKey())
	if err != nil {
		return errors.Wrap(err, "Error marshaling ledgerKey")
	}

	expectedEntry, exist := v.currentEntries[key]
	if !exist {
		return StateError{errors.Errorf(
			"Cannot find entry in currentEntries map: %s (key = %s)",
			base64.StdEncoding.EncodeToString(actualEntryMarshaled),
			key,
		)}
	}
	delete(v.currentEntries, key)

	preTransformExpectedEntry := expectedEntry
	preTransformExpectedEntryMarshaled, err := preTransformExpectedEntry.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "Error marshaling preTransformExpectedEntry")
	}

	if v.TransformFunction != nil {
		var ignore bool
		ignore, expectedEntry = v.TransformFunction(expectedEntry)
		// Extra check: if entry was ignored in GetEntries, it shouldn't be
		// ignored here.
		if ignore {
			return errors.Errorf(
				"Entry ignored in GetEntries but not ignored in Write: %s. Possibly TransformFunction is buggy.",
				base64.StdEncoding.EncodeToString(preTransformExpectedEntryMarshaled),
			)
		}
	}

	expectedEntryMarshaled, err := expectedEntry.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "Error marshaling expectedEntry")
	}

	if !bytes.Equal(actualEntryMarshaled, expectedEntryMarshaled) {
		return StateError{errors.Errorf(
			"Entry does not match the fetched entry. Expected: %s (pretransform = %s), actual: %s",
			base64.StdEncoding.EncodeToString(expectedEntryMarshaled),
			base64.StdEncoding.EncodeToString(preTransformExpectedEntryMarshaled),
			base64.StdEncoding.EncodeToString(actualEntryMarshaled),
		)}
	}

	return nil
}

// Verify should be run after all GetEntries/Write calls. If there were no errors
// so far it means that all entries present in history buckets matches the entries
// in application storage. However, it's still possible that state is invalid when:
//   * Not all entries have been read from history buckets (ex. due to a bug).
//   * Some entries were not compared using Write.
//   * There are some extra entries in application storage not present in history
//     buckets.
// Any `StateError` returned by this method indicates invalid state!
func (v *StateVerifier) Verify(countAll int) error {
	err := v.checkUnreadEntries()
	if err != nil {
		return err
	}

	if !v.readingDone {
		return errors.New("There are unread entries in state reader. Process all entries before calling Verify.")
	}

	if v.readEntries != countAll {
		return StateError{errors.Errorf(
			"Number of entries read using GetEntries (%d) does not match number of entries in your storage (%d).",
			v.readEntries,
			countAll,
		)}
	}

	return nil
}

func (v *StateVerifier) checkUnreadEntries() error {
	if len(v.currentEntries) > 0 {
		var entry xdr.LedgerEntry
		for _, e := range v.currentEntries {
			entry = e
			break
		}

		// Ignore error as StateError below is more important
		entryString, _ := xdr.MarshalBase64(entry)
		return StateError{errors.Errorf(
			"Entries (%d) not found locally, example: %s",
			len(v.currentEntries),
			entryString,
		)}
	}

	return nil
}
