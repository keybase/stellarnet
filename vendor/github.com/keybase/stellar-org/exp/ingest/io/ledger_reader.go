package io

import (
	"context"
	"io"

	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerReader provides convenient, streaming access to the transactions within a ledger.
type LedgerReader interface {
	GetSequence() uint32
	GetHeader() xdr.LedgerHeaderHistoryEntry
	// Read should return the next transaction. If there are no more
	// transactions it should return `io.EOF` error.
	Read() (LedgerTransaction, error)
	// Close should be called when reading is finished. This is especially
	// helpful when there are still some transactions available so reader can stop
	// streaming them.
	Close() error
}

// DBLedgerReader is a database-backed implementation of the io.LedgerReader interface.
// Use NewDBLedgerReader to create a new instance.
type DBLedgerReader struct {
	ctx            context.Context
	sequence       uint32
	backend        ledgerbackend.LedgerBackend
	header         xdr.LedgerHeaderHistoryEntry
	transactions   []LedgerTransaction
	upgradeChanges []Change
	readIdx        int
	upgradeReadIdx int
}

// Ensure DBLedgerReader implements LedgerReader
var _ LedgerReader = (*DBLedgerReader)(nil)

// NewDBLedgerReader creates a new DBLedgerReader instance.
// Note that DBLedgerReader is not thread safe and should not be shared by multiple goroutines
func NewDBLedgerReader(
	ctx context.Context, sequence uint32, backend ledgerbackend.LedgerBackend,
) (*DBLedgerReader, error) {
	reader := &DBLedgerReader{
		ctx:      ctx,
		sequence: sequence,
		backend:  backend,
	}

	err := reader.init()
	if err != nil {
		return nil, err
	}

	return reader, nil
}

// GetSequence returns the sequence number of the ledger data stored by this object.
func (dblrc *DBLedgerReader) GetSequence() uint32 {
	return dblrc.sequence
}

// GetHeader returns the XDR Header data associated with the stored ledger.
func (dblrc *DBLedgerReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	return dblrc.header
}

// Read returns the next transaction in the ledger, ordered by tx number, each time it is called. When there
// are no more transactions to return, an EOF error is returned.
func (dblrc *DBLedgerReader) Read() (LedgerTransaction, error) {
	if err := dblrc.ctx.Err(); err != nil {
		return LedgerTransaction{}, err
	}

	if dblrc.readIdx < len(dblrc.transactions) {
		dblrc.readIdx++
		return dblrc.transactions[dblrc.readIdx-1], nil
	}
	return LedgerTransaction{}, io.EOF
}

// readUpgradeChange returns the next upgrade change in the ledger, each time it
// is called. When there are no more upgrades to return, an EOF error is returned.
func (dblrc *DBLedgerReader) readUpgradeChange() (Change, error) {
	if err := dblrc.ctx.Err(); err != nil {
		return Change{}, err
	}

	if dblrc.upgradeReadIdx < len(dblrc.upgradeChanges) {
		dblrc.upgradeReadIdx++
		return dblrc.upgradeChanges[dblrc.upgradeReadIdx-1], nil
	}
	return Change{}, io.EOF
}

// Rewind resets the reader back to the first transaction in the ledger
func (dblrc *DBLedgerReader) rewind() {
	dblrc.readIdx = 0
}

// Init pulls data from the backend to set this object up for use.
func (dblrc *DBLedgerReader) init() error {
	exists, ledgerCloseMeta, err := dblrc.backend.GetLedger(dblrc.sequence)

	if err != nil {
		return errors.Wrap(err, "error reading ledger from backend")
	}
	if !exists {
		return ErrNotFound
	}

	dblrc.header = ledgerCloseMeta.LedgerHeader

	dblrc.storeTransactions(ledgerCloseMeta)

	for _, upgradeChanges := range ledgerCloseMeta.UpgradesMeta {
		changes := getChangesFromLedgerEntryChanges(upgradeChanges)
		dblrc.upgradeChanges = append(dblrc.upgradeChanges, changes...)
	}

	return nil
}

// storeTransactions maps the close meta data into a slice of LedgerTransaction structs, to provide
// a per-transaction view of the data when Read() is called.
func (dblrc *DBLedgerReader) storeTransactions(lcm ledgerbackend.LedgerCloseMeta) {
	for i := range lcm.TransactionEnvelope {
		dblrc.transactions = append(dblrc.transactions, LedgerTransaction{
			Index:      uint32(i + 1), // Transactions start at '1'
			Envelope:   lcm.TransactionEnvelope[i],
			Result:     lcm.TransactionResult[i],
			Meta:       lcm.TransactionMeta[i],
			FeeChanges: lcm.TransactionFeeChanges[i],
		})
	}
}

func (dblrc *DBLedgerReader) Close() error {
	dblrc.transactions = nil
	dblrc.upgradeChanges = nil
	return nil
}
