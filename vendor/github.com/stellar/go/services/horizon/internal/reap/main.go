// Package reap contains the history reaping subsystem for horizon.  This system
// is designed to remove data from the history database such that it does not
// grow indefinitely.  The system can be configured with a number of ledgers to
// maintain at a minimum.
package reap

import (
	"context"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/support/db"
)

// System represents the history reaping subsystem of horizon.
type System struct {
	HistoryQ       *history.Q
	RetentionCount uint
	ledgerState    *ledger.State
	ctx            context.Context
	cancel         context.CancelFunc
}

// New initializes the reaper, causing it to begin polling the stellar-core
// database for now ledgers and ingesting data into the horizon database.
func New(retention uint, dbSession db.SessionInterface, ledgerState *ledger.State) *System {
	ctx, cancel := context.WithCancel(context.Background())

	r := &System{
		HistoryQ:       &history.Q{dbSession.Clone()},
		RetentionCount: retention,
		ledgerState:    ledgerState,
		ctx:            ctx,
		cancel:         cancel,
	}

	return r
}
