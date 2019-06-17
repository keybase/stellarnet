package ingestadapters

import (
	"fmt"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/historyarchive"
)

const msrBufferSize = 50000

// HistoryArchiveAdapter is an adapter for the historyarchive package to read from history archives
type HistoryArchiveAdapter struct {
	archive *historyarchive.Archive
}

// MakeHistoryArchiveAdapter is a factory method to make a HistoryArchiveAdapter
func MakeHistoryArchiveAdapter(archive *historyarchive.Archive) *HistoryArchiveAdapter {
	return &HistoryArchiveAdapter{archive: archive}
}

// GetLatestLedgerSequence returns the latest ledger sequence or an error
func (haa *HistoryArchiveAdapter) GetLatestLedgerSequence() (uint32, error) {
	has, e := haa.archive.GetRootHAS()
	if e != nil {
		return 0, fmt.Errorf("could not get root HAS: %s", e)
	}

	return has.CurrentLedger, nil
}

// GetState returns a reader with the state of the ledger at the provided sequence number
func (haa *HistoryArchiveAdapter) GetState(sequence uint32) (io.StateReadCloser, error) {
	if !haa.archive.CategoryCheckpointExists("history", sequence) {
		return nil, fmt.Errorf("history checkpoint does not exist for ledger %d", sequence)
	}

	sr, e := io.MakeMemoryStateReader(haa.archive, sequence, msrBufferSize)
	if e != nil {
		return nil, fmt.Errorf("could not make memory state reader: %s", e)
	}

	return sr, nil
}

// GetLedger gets a ledger transaction result at the provided sequence number
func (haa *HistoryArchiveAdapter) GetLedger(sequence uint32) (io.ArchiveLedgerReader, error) {
	return nil, fmt.Errorf("not implemented yet")
}
