package io

import (
	"context"
	"io"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/xdr"
)

// ChangeReader provides convenient, streaming access to a sequence of Changes
type ChangeReader interface {
	// Read should return the next `Change` in the leader. If there are no more
	// changes left it should return an `io.EOF` error.
	Read() (Change, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some changes available so reader can stop
	// streaming them.
	Close() error
}

// LedgerChangeReader is a ChangeReader which returns Changes from Stellar Core
// for a single ledger
type LedgerChangeReader struct {
	dbReader               DBLedgerReader
	streamedFeeChanges     bool
	streamedMetaChanges    bool
	streamedUpgradeChanges bool
	pending                []Change
	pendingIndex           int
}

// Ensure LedgerChangeReader implements ChangeReader
var _ ChangeReader = (*LedgerChangeReader)(nil)

// NewLedgerChangeReader constructs a new LedgerChangeReader instance bound to the given ledger.
// Note that the returned LedgerChangeReader is not thread safe and should not be shared
// by multiple goroutines.
func NewLedgerChangeReader(
	ctx context.Context, sequence uint32, backend ledgerbackend.LedgerBackend,
) (*LedgerChangeReader, error) {
	reader, err := NewDBLedgerReader(ctx, sequence, backend)
	if err != nil {
		return nil, err
	}

	return &LedgerChangeReader{dbReader: *reader}, nil
}

// GetHeader returns the ledger header for the reader
func (r *LedgerChangeReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return r.dbReader.GetHeader()
}

func (r *LedgerChangeReader) getNextFeeChange() (Change, error) {
	if r.streamedFeeChanges {
		return Change{}, io.EOF
	}

	// Remember that it's possible that transaction can remove a preauth
	// tx signer even when it's a failed transaction so we need to check
	// failed transactions too.
	for {
		transaction, err := r.dbReader.Read()
		if err != nil {
			if err == io.EOF {
				r.dbReader.rewind()
				r.streamedFeeChanges = true
				return Change{}, io.EOF
			} else {
				return Change{}, err
			}
		}

		changes := transaction.GetFeeChanges()
		if len(changes) >= 1 {
			r.pending = append(r.pending, changes[1:]...)
			return changes[0], nil
		}
	}
}

func (r *LedgerChangeReader) getNextMetaChange() (Change, error) {
	if r.streamedMetaChanges {
		return Change{}, io.EOF
	}

	for {
		transaction, err := r.dbReader.Read()
		if err != nil {
			if err == io.EOF {
				r.streamedMetaChanges = true
				return Change{}, io.EOF
			} else {
				return Change{}, err
			}
		}

		changes, err := transaction.GetChanges()
		if err != nil {
			return Change{}, err
		}
		if len(changes) >= 1 {
			r.pending = append(r.pending, changes[1:]...)
			return changes[0], nil
		}
	}
}

func (r *LedgerChangeReader) getNextUpgradeChange() (Change, error) {
	if r.streamedUpgradeChanges {
		return Change{}, io.EOF
	}

	change, err := r.dbReader.readUpgradeChange()
	if err != nil {
		if err == io.EOF {
			r.streamedUpgradeChanges = true
			return Change{}, io.EOF
		} else {
			return Change{}, err
		}
	}

	return change, nil
}

// Read returns the next change in the stream.
// If there are no changes remaining io.EOF is returned
// as an error.
func (r *LedgerChangeReader) Read() (Change, error) {
	if err := r.dbReader.ctx.Err(); err != nil {
		return Change{}, err
	}

	if r.pendingIndex < len(r.pending) {
		next := r.pending[r.pendingIndex]
		r.pendingIndex++
		if r.pendingIndex == len(r.pending) {
			r.pendingIndex = 0
			r.pending = r.pending[:0]
		}
		return next, nil
	}

	change, err := r.getNextFeeChange()
	if err == nil || err != io.EOF {
		return change, err
	}

	change, err = r.getNextMetaChange()
	if err == nil || err != io.EOF {
		return change, err
	}

	return r.getNextUpgradeChange()
}

func (r *LedgerChangeReader) Close() error {
	r.pending = nil
	r.streamedFeeChanges = true
	r.streamedMetaChanges = true
	r.streamedUpgradeChanges = true
	return r.dbReader.Close()
}
