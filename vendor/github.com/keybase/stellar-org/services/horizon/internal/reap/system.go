package reap

import (
	"time"

	"github.com/stellar/go/services/horizon/internal/errors"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/log"
)

// DeleteUnretainedHistory removes all data associated with unretained ledgers.
func (r *System) DeleteUnretainedHistory() error {
	// RetentionCount of 0 indicates "keep all history"
	if r.RetentionCount == 0 {
		return nil
	}

	var (
		latest      = ledger.CurrentState()
		targetElder = (latest.HistoryLatest - int32(r.RetentionCount)) + 1
	)

	if targetElder < latest.HistoryElder {
		return nil
	}

	err := r.clearBefore(targetElder)
	if err != nil {
		return err
	}

	log.
		WithField("new_elder", targetElder).
		Info("reaper succeeded")

	return nil
}

// Tick triggers the reaper system to update itself, deleted unretained history
// if it is the appropriate time.
func (r *System) Tick() {
	if time.Now().Before(r.nextRun) {
		return
	}

	r.runOnce()
	r.nextRun = time.Now().Add(1 * time.Hour)
}

func (r *System) runOnce() {
	defer func() {
		if rec := recover(); rec != nil {
			err := errors.FromPanic(rec)
			log.Errorf("reaper panicked: %s", err)
			errors.ReportToSentry(err, nil)
		}
	}()

	err := r.DeleteUnretainedHistory()
	if err != nil {
		log.Errorf("reaper failed: %s", err)
	}
}

func (r *System) clearBefore(seq int32) error {
	log.WithField("new_elder", seq).Info("reaper: clearing")

	start, end, err := toid.LedgerRangeInclusive(1, seq-1)
	if err != nil {
		return err
	}

	err = r.HistoryQ.DeleteRangeAll(start, end)
	if err != nil {
		return err
	}

	return nil
}
