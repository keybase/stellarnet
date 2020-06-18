package adapters

import (
	"context"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/xdr"
)

// HistoryArchiveAdapter is an adapter for the historyarchive package to read from history archives
type HistoryArchiveAdapter struct {
	archive historyarchive.ArchiveInterface
}

type HistoryArchiveAdapterInterface interface {
	GetLatestLedgerSequence() (uint32, error)
	BucketListHash(sequence uint32) (xdr.Hash, error)
	GetState(
		ctx context.Context, sequence uint32, maxStreamRetries int,
	) (io.ChangeReader, error)
}

// MakeHistoryArchiveAdapter is a factory method to make a HistoryArchiveAdapter
func MakeHistoryArchiveAdapter(archive historyarchive.ArchiveInterface) HistoryArchiveAdapterInterface {
	return &HistoryArchiveAdapter{archive: archive}
}

// GetLatestLedgerSequence returns the latest ledger sequence or an error
func (haa *HistoryArchiveAdapter) GetLatestLedgerSequence() (uint32, error) {
	has, err := haa.archive.GetRootHAS()
	if err != nil {
		return 0, errors.Wrap(err, "could not get root HAS")
	}

	return has.CurrentLedger, nil
}

// BucketListHash returns the bucket list hash to compare with hash in the
// ledger header fetched from Stellar-Core.
func (haa *HistoryArchiveAdapter) BucketListHash(sequence uint32) (xdr.Hash, error) {
	exists, err := haa.archive.CategoryCheckpointExists("history", sequence)
	if err != nil {
		return xdr.Hash{}, errors.Wrap(err, "error checking if category checkpoint exists")
	}
	if !exists {
		return xdr.Hash{}, errors.Errorf("history checkpoint does not exist for ledger %d", sequence)
	}

	has, err := haa.archive.GetCheckpointHAS(sequence)
	if err != nil {
		return xdr.Hash{}, errors.Wrapf(err, "unable to get checkpoint HAS at ledger sequence %d", sequence)
	}

	return has.BucketListHash()
}

// GetState returns a reader with the state of the ledger at the provided sequence number
// `maxStreamRetries` determines how many times the reader will retry when encountering
// errors while streaming xdr bucket entries from the history archive.
// Set `maxStreamRetries` to 0 if there should be no retry attempts
func (haa *HistoryArchiveAdapter) GetState(
	ctx context.Context, sequence uint32, maxStreamRetries int,
) (io.ChangeReader, error) {
	exists, err := haa.archive.CategoryCheckpointExists("history", sequence)
	if err != nil {
		return nil, errors.Wrap(err, "error checking if category checkpoint exists")
	}
	if !exists {
		return nil, errors.Errorf("history checkpoint does not exist for ledger %d", sequence)
	}

	sr, e := io.MakeSingleLedgerStateReader(ctx, haa.archive, sequence, maxStreamRetries)
	if e != nil {
		return nil, errors.Wrap(e, "could not make memory state reader")
	}

	return sr, nil
}
