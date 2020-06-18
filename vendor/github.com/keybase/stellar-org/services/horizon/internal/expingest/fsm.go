package expingest

import (
	"fmt"
	"time"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	logpkg "github.com/stellar/go/support/log"
)

var (
	defaultSleep = time.Second
	// ErrReingestRangeConflict indicates that the reingest range overlaps with
	// horizon's most recently ingested ledger
	ErrReingestRangeConflict = errors.New("reingest range overlaps with horizon ingestion")
)

type stateMachineNode interface {
	run(*System) (transition, error)
	String() string
}

type transition struct {
	node          stateMachineNode
	sleepDuration time.Duration
}

func stop() transition {
	return transition{node: stopState{}, sleepDuration: 0}
}

func start() transition {
	return transition{node: startState{}, sleepDuration: defaultSleep}
}

func rebuild(checkpointLedger uint32) transition {
	return transition{
		node: buildState{
			checkpointLedger: checkpointLedger,
		},
		sleepDuration: defaultSleep,
	}
}

func resume(latestSuccessfullyProcessedLedger uint32) transition {
	return transition{
		node: resumeState{
			latestSuccessfullyProcessedLedger: latestSuccessfullyProcessedLedger,
		},
		sleepDuration: defaultSleep,
	}
}

func resumeImmediately(latestSuccessfullyProcessedLedger uint32) transition {
	return transition{
		node: resumeState{
			latestSuccessfullyProcessedLedger: latestSuccessfullyProcessedLedger,
		},
		sleepDuration: 0,
	}
}

func retryResume(r resumeState) transition {
	return transition{
		node:          r,
		sleepDuration: defaultSleep,
	}
}

func historyRange(fromLedger, toLedger uint32) transition {
	return transition{
		node: historyRangeState{
			fromLedger: fromLedger,
			toLedger:   toLedger,
		},
		sleepDuration: defaultSleep,
	}
}

func waitForCheckPoint() transition {
	return transition{
		node:          waitForCheckpointState{},
		sleepDuration: 0,
	}
}

type stopState struct{}

func (stopState) String() string {
	return "stop"
}

func (stopState) run(s *System) (transition, error) {
	return stop(), errors.New("Cannot run terminal state")
}

type startState struct{}

func (startState) String() string {
	return "start"
}

func (startState) run(s *System) (transition, error) {
	if err := s.historyQ.Begin(); err != nil {
		return start(), errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()

	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return start(), errors.Wrap(err, getLastIngestedErrMsg)
	}

	ingestVersion, err := s.historyQ.GetExpIngestVersion()
	if err != nil {
		return start(), errors.Wrap(err, getExpIngestVersionErrMsg)
	}

	if ingestVersion > CurrentVersion {
		log.WithFields(logpkg.F{
			"ingestVersion":  ingestVersion,
			"currentVersion": CurrentVersion,
		}).Info("ingestion version in db is greater than current version, going to terminate")
		return stop(), nil
	}

	lastHistoryLedger, err := s.historyQ.GetLatestLedger()
	if err != nil {
		return start(), errors.Wrap(err, "Error getting last history ledger sequence")
	}

	if ingestVersion != CurrentVersion || lastIngestedLedger == 0 {
		// This block is either starting from empty state or ingestion
		// version upgrade.
		// This will always run on a single instance due to the fact that
		// `LastLedgerExpIngest` value is blocked for update and will always
		// be updated when leading instance finishes processing state.
		// In case of errors it will start `Init` from the beginning.
		var lastCheckpoint uint32
		lastCheckpoint, err = s.historyAdapter.GetLatestLedgerSequence()
		if err != nil {
			return start(), errors.Wrap(err, "Error getting last checkpoint")
		}

		if lastHistoryLedger != 0 {
			// There are ledgers in history_ledgers table. This means that the
			// old or new ingest system was running prior the upgrade. In both
			// cases we need to:
			// * Wait for the checkpoint ledger if the latest history ledger is
			//   greater that the latest checkpoint ledger.
			// * Catchup history data if the latest history ledger is less than
			//   the latest checkpoint ledger.
			// * Build state from the last checkpoint if the latest history ledger
			//   is equal to the latest checkpoint ledger.
			switch {
			case lastHistoryLedger > lastCheckpoint:
				return waitForCheckPoint(), nil
			case lastHistoryLedger < lastCheckpoint:
				return historyRange(lastHistoryLedger+1, lastCheckpoint), nil
			default: // lastHistoryLedger == lastCheckpoint
				// Build state but make sure it's using `lastCheckpoint`. It's possible
				// that the new checkpoint will be created during state transition.
				return rebuild(lastCheckpoint), nil
			}
		}

		return rebuild(lastCheckpoint), nil
	}

	switch {
	case lastHistoryLedger > lastIngestedLedger:
		// Expingest was running at some point the past but was turned off.
		// Now it's on by default but the latest history ledger is greater
		// than the latest expingest ledger. We reset the exp ledger sequence
		// so init state will rebuild the state correctly.
		err = s.historyQ.UpdateLastLedgerExpIngest(0)
		if err != nil {
			return start(), errors.Wrap(err, updateLastLedgerExpIngestErrMsg)
		}
		err = s.historyQ.Commit()
		if err != nil {
			return start(), errors.Wrap(err, commitErrMsg)
		}
		return start(), nil
	// lastHistoryLedger != 0 check is here to check the case when one node ingested
	// the state (so latest exp ingest is > 0) but no history has been ingested yet.
	// In such case we execute default case and resume from the last ingested
	// ledger.
	case lastHistoryLedger != 0 && lastHistoryLedger < lastIngestedLedger:
		// Expingest was running at some point the past but was turned off.
		// Now it's on by default but the latest history ledger is less
		// than the latest expingest ledger. We catchup history.
		return historyRange(lastHistoryLedger+1, lastIngestedLedger), nil
	default: // lastHistoryLedger == lastIngestedLedger
		// The other node already ingested a state (just now or in the past)
		// so we need to get offers from a DB, then resume session normally.
		// State pipeline is NOT processed.
		log.WithField("last_ledger", lastIngestedLedger).
			Info("Resuming ingestion system from last processed ledger...")

		return resume(lastIngestedLedger), nil
	}
}

type buildState struct {
	checkpointLedger uint32
}

func (b buildState) String() string {
	return fmt.Sprintf("buildFromCheckpoint(checkpointLedger=%d)", b.checkpointLedger)
}

func (b buildState) run(s *System) (transition, error) {
	if b.checkpointLedger == 0 {
		return start(), errors.New("unexpected checkpointLedger value")
	}

	if err := s.historyQ.Begin(); err != nil {
		return start(), errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()

	// We need to get this value `FOR UPDATE` so all other instances
	// are blocked.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return start(), errors.Wrap(err, getLastIngestedErrMsg)
	}

	ingestVersion, err := s.historyQ.GetExpIngestVersion()
	if err != nil {
		return start(), errors.Wrap(err, getExpIngestVersionErrMsg)
	}

	// Double check if we should proceed with state ingestion. It's possible that
	// another ingesting instance will be redirected to this state from `init`
	// but it's first to complete the task.
	if ingestVersion == CurrentVersion && lastIngestedLedger > 0 {
		log.Info("Another instance completed `buildState`. Skipping...")
		return start(), nil
	}

	if err = s.updateCursor(b.checkpointLedger - 1); err != nil {
		// Don't return updateCursor error.
		log.WithError(err).Warn("error updating stellar-core cursor")
	}

	log.Info("Starting ingestion system from empty state...")

	// Clear last_ingested_ledger in key value store
	err = s.historyQ.UpdateLastLedgerExpIngest(0)
	if err != nil {
		return start(), errors.Wrap(err, updateLastLedgerExpIngestErrMsg)
	}

	// Clear invalid state in key value store. It's possible that upgraded
	// ingestion is fixing it.
	err = s.historyQ.UpdateExpStateInvalid(false)
	if err != nil {
		return start(), errors.Wrap(err, updateExpStateInvalidErrMsg)
	}

	// State tables should be empty.
	err = s.historyQ.TruncateExpingestStateTables()
	if err != nil {
		return start(), errors.Wrap(err, "Error clearing ingest tables")
	}

	log.WithFields(logpkg.F{
		"ledger": b.checkpointLedger,
	}).Info("Processing state")
	startTime := time.Now()

	stats, err := s.runner.RunHistoryArchiveIngestion(b.checkpointLedger)
	if err != nil {
		return start(), errors.Wrap(err, "Error ingesting history archive")
	}

	if err = s.historyQ.UpdateExpIngestVersion(CurrentVersion); err != nil {
		return start(), errors.Wrap(err, "Error updating expingest version")
	}

	if err = s.completeIngestion(b.checkpointLedger); err != nil {
		return start(), err
	}

	log.
		WithFields(stats.Map()).
		WithFields(logpkg.F{
			"ledger":   b.checkpointLedger,
			"duration": time.Since(startTime).Seconds(),
		}).
		Info("Processed state")

	// If successful, continue from the next ledger
	return resume(b.checkpointLedger), nil
}

type resumeState struct {
	latestSuccessfullyProcessedLedger uint32
}

func (r resumeState) String() string {
	return fmt.Sprintf("resume(latestSuccessfullyProcessedLedger=%d)", r.latestSuccessfullyProcessedLedger)
}

func (r resumeState) run(s *System) (transition, error) {
	if r.latestSuccessfullyProcessedLedger == 0 {
		return start(), errors.New("unexpected latestSuccessfullyProcessedLedger value")
	}

	if err := s.historyQ.Begin(); err != nil {
		return retryResume(r),
			errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()

	// This will get the value `FOR UPDATE`, blocking it for other nodes.
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		return retryResume(r), errors.Wrap(err, getLastIngestedErrMsg)
	}

	ingestLedger := r.latestSuccessfullyProcessedLedger + 1

	if ingestLedger > lastIngestedLedger+1 {
		return start(), errors.New("expected ingest ledger to be at most one greater " +
			"than last ingested ledger in db")
	} else if ingestLedger <= lastIngestedLedger {
		log.WithField("ingestLedger", ingestLedger).
			WithField("lastIngestedLedger", lastIngestedLedger).
			Info("bumping ingest ledger to next ledger after ingested ledger in db")
		ingestLedger = lastIngestedLedger + 1
	}

	ingestVersion, err := s.historyQ.GetExpIngestVersion()
	if err != nil {
		return retryResume(r), errors.Wrap(err, getExpIngestVersionErrMsg)
	}

	if ingestVersion != CurrentVersion {
		log.WithFields(logpkg.F{
			"ingestVersion":  ingestVersion,
			"currentVersion": CurrentVersion,
		}).Info("ingestion version in db is not current, going back to start state")
		return start(), nil
	}

	lastHistoryLedger, err := s.historyQ.GetLatestLedger()
	if err != nil {
		return retryResume(r), errors.Wrap(err, "could not get latest history ledger")
	}

	if lastHistoryLedger != 0 && lastHistoryLedger != lastIngestedLedger {
		log.WithFields(logpkg.F{
			"lastHistoryLedger":  lastHistoryLedger,
			"lastIngestedLedger": lastIngestedLedger,
		}).Info(
			"last history ledger does not match last ingested ledger, " +
				"going back to start state",
		)
		return start(), nil
	}

	// Check if ledger is closed
	latestLedgerCore, err := s.ledgerBackend.GetLatestLedgerSequence()
	if err != nil {
		return retryResume(r), errors.Wrap(err, "Error getting lastest ledger in stellar-core")
	}

	if latestLedgerCore < ingestLedger {
		log.WithFields(logpkg.F{
			"ingest_sequence": ingestLedger,
			"core_sequence":   latestLedgerCore,
		}).Info("Waiting for ledger to be available in stellar-core")
		// Go to the next state, machine will wait for 1s. before continuing.
		return retryResume(r), nil
	}

	startTime := time.Now()

	log.WithFields(logpkg.F{
		"sequence": ingestLedger,
		"state":    true,
		"ledger":   true,
		"commit":   true,
	}).Info("Processing ledger")

	changeStats, ledgerTransactionStats, err := s.runner.RunAllProcessorsOnLedger(ingestLedger)
	if err != nil {
		return retryResume(r), errors.Wrap(err, "Error running processors on ledger")
	}

	if err = s.completeIngestion(ingestLedger); err != nil {
		return retryResume(r), err
	}

	if err = s.updateCursor(ingestLedger); err != nil {
		// Don't return updateCursor error.
		log.WithError(err).Warn("error updating stellar-core cursor")
	}

	duration := time.Since(startTime)
	s.Metrics.LedgerIngestionTimer.Update(duration)
	log.
		WithFields(changeStats.Map()).
		WithFields(ledgerTransactionStats.Map()).
		WithFields(logpkg.F{
			"sequence": ingestLedger,
			"duration": duration.Seconds(),
			"state":    true,
			"ledger":   true,
			"commit":   true,
		}).
		Info("Processed ledger")

	s.maybeVerifyState(ingestLedger)

	return resumeImmediately(ingestLedger), nil
}

type historyRangeState struct {
	fromLedger uint32
	toLedger   uint32
}

func (h historyRangeState) String() string {
	return fmt.Sprintf(
		"historyRange(fromLedger=%d, toLedger=%d)",
		h.fromLedger,
		h.toLedger,
	)
}

// historyRangeState is used when catching up history data
func (h historyRangeState) run(s *System) (transition, error) {
	if h.fromLedger == 0 || h.toLedger == 0 ||
		h.fromLedger > h.toLedger {
		return start(), errors.Errorf("invalid range: [%d, %d]", h.fromLedger, h.toLedger)
	}

	if err := s.historyQ.Begin(); err != nil {
		return start(), errors.Wrap(err, "Error starting a transaction")
	}
	defer s.historyQ.Rollback()

	// acquire distributed lock so no one else can perform ingestion operations.
	if _, err := s.historyQ.GetLastLedgerExpIngest(); err != nil {
		return start(), errors.Wrap(err, getLastIngestedErrMsg)
	}

	lastHistoryLedger, err := s.historyQ.GetLatestLedger()
	if err != nil {
		return start(), errors.Wrap(err, "could not get latest history ledger")
	}

	// We should be ingesting the ledger which occurs after
	// lastHistoryLedger. Otherwise, some other horizon node has
	// already completed the ingest history range operation and
	// we should go back to the init state
	if lastHistoryLedger != h.fromLedger-1 {
		return start(), nil
	}

	for cur := h.fromLedger; cur <= h.toLedger; cur++ {
		if err = runTransactionProcessorsOnLedger(s, cur); err != nil {
			return start(), err
		}
	}

	if err = s.historyQ.Commit(); err != nil {
		return start(), errors.Wrap(err, commitErrMsg)
	}

	return start(), nil
}

func runTransactionProcessorsOnLedger(s *System, ledger uint32) error {
	log.WithFields(logpkg.F{
		"sequence": ledger,
		"state":    false,
		"ledger":   true,
		"commit":   false,
	}).Info("Processing ledger")
	startTime := time.Now()

	ledgerTransactionStats, err := s.runner.RunTransactionProcessorsOnLedger(ledger)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error processing ledger sequence=%d", ledger))
	}

	log.
		WithFields(ledgerTransactionStats.Map()).
		WithFields(logpkg.F{
			"sequence": ledger,
			"duration": time.Since(startTime).Seconds(),
			"state":    false,
			"ledger":   true,
			"commit":   false,
		}).
		Info("Processed ledger")
	return nil
}

type reingestHistoryRangeState struct {
	fromLedger uint32
	toLedger   uint32
	force      bool
}

func (h reingestHistoryRangeState) String() string {
	return fmt.Sprintf(
		"reingestHistoryRange(fromLedger=%d, toLedger=%d, force=%t)",
		h.fromLedger,
		h.toLedger,
		h.force,
	)
}

func (h reingestHistoryRangeState) ingestRange(s *System, fromLedger, toLedger uint32) error {
	if s.historyQ.GetTx() == nil {
		return errors.New("expected transaction to be present")
	}

	// Clear history data before ingesting - used in `reingest range` command.
	start, end, err := toid.LedgerRangeInclusive(
		int32(fromLedger),
		int32(toLedger),
	)
	if err != nil {
		return errors.Wrap(err, "Invalid range")
	}

	err = s.historyQ.DeleteRangeAll(start, end)
	if err != nil {
		return errors.Wrap(err, "error in DeleteRangeAll")
	}

	for cur := fromLedger; cur <= toLedger; cur++ {
		if err = runTransactionProcessorsOnLedger(s, cur); err != nil {
			return err
		}
	}

	return nil
}

// reingestHistoryRangeState is used as a command to reingest historical data
func (h reingestHistoryRangeState) run(s *System) (transition, error) {
	if h.fromLedger == 0 || h.toLedger == 0 ||
		h.fromLedger > h.toLedger {
		return stop(), errors.Errorf("invalid range: [%d, %d]", h.fromLedger, h.toLedger)
	}

	if h.force {
		if err := s.historyQ.Begin(); err != nil {
			return stop(), errors.Wrap(err, "Error starting a transaction")
		}
		defer s.historyQ.Rollback()

		// acquire distributed lock so no one else can perform ingestion operations.
		if _, err := s.historyQ.GetLastLedgerExpIngest(); err != nil {
			return stop(), errors.Wrap(err, getLastIngestedErrMsg)
		}

		if err := h.ingestRange(s, h.fromLedger, h.toLedger); err != nil {
			return stop(), err
		}

		if err := s.historyQ.Commit(); err != nil {
			return stop(), errors.Wrap(err, commitErrMsg)
		}
	} else {
		lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngestNonBlocking()
		if err != nil {
			return stop(), errors.Wrap(err, getLastIngestedErrMsg)
		}

		if lastIngestedLedger > 0 && h.toLedger >= lastIngestedLedger {
			return stop(), ErrReingestRangeConflict
		}

		for cur := h.fromLedger; cur <= h.toLedger; cur++ {
			err := func(ledger uint32) error {
				if err := s.historyQ.Begin(); err != nil {
					return errors.Wrap(err, "Error starting a transaction")
				}
				defer s.historyQ.Rollback()

				// ingest each ledger in a separate transaction to prevent deadlocks
				// when acquiring ShareLocks from multiple parallel reingest range processes
				if err := h.ingestRange(s, ledger, ledger); err != nil {
					return err
				}

				if err := s.historyQ.Commit(); err != nil {
					return errors.Wrap(err, commitErrMsg)
				}

				return nil
			}(cur)
			if err != nil {
				return stop(), err
			}
		}
	}

	return stop(), nil
}

type waitForCheckpointState struct{}

func (waitForCheckpointState) String() string {
	return "waitForCheckpoint"
}

func (waitForCheckpointState) run(*System) (transition, error) {
	log.Info("Waiting for the next checkpoint...")
	time.Sleep(10 * time.Second)
	return start(), nil
}

type verifyRangeState struct {
	fromLedger  uint32
	toLedger    uint32
	verifyState bool
}

func (v verifyRangeState) String() string {
	return fmt.Sprintf(
		"verifyRange(fromLedger=%d, toLedger=%d, verifyState=%t)",
		v.fromLedger,
		v.toLedger,
		v.verifyState,
	)
}

func (v verifyRangeState) run(s *System) (transition, error) {
	if v.fromLedger == 0 || v.toLedger == 0 ||
		v.fromLedger > v.toLedger {
		return stop(), errors.Errorf("invalid range: [%d, %d]", v.fromLedger, v.toLedger)
	}

	if err := s.historyQ.Begin(); err != nil {
		err = errors.Wrap(err, "Error starting a transaction")
		return stop(), err
	}
	defer s.historyQ.Rollback()

	// Simple check if DB clean
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		err = errors.Wrap(err, getLastIngestedErrMsg)
		return stop(), err
	}

	if lastIngestedLedger != 0 {
		err = errors.New("Database not empty")
		return stop(), err
	}

	log.WithFields(logpkg.F{
		"ledger": v.fromLedger,
	}).Info("Processing state")
	startTime := time.Now()

	stats, err := s.runner.RunHistoryArchiveIngestion(v.fromLedger)
	if err != nil {
		err = errors.Wrap(err, "Error ingesting history archive")
		return stop(), err
	}

	if err = s.completeIngestion(v.fromLedger); err != nil {
		return stop(), err
	}

	log.
		WithFields(stats.Map()).
		WithFields(logpkg.F{
			"ledger":   v.fromLedger,
			"duration": time.Since(startTime).Seconds(),
		}).
		Info("Processed state")

	for sequence := v.fromLedger + 1; sequence <= v.toLedger; sequence++ {
		log.WithFields(logpkg.F{
			"sequence": sequence,
			"state":    true,
			"ledger":   true,
			"commit":   true,
		}).Info("Processing ledger")
		startTime := time.Now()

		if err = s.historyQ.Begin(); err != nil {
			err = errors.Wrap(err, "Error starting a transaction")
			return stop(), err
		}

		var changeStats io.StatsChangeProcessorResults
		var ledgerTransactionStats io.StatsLedgerTransactionProcessorResults
		changeStats, ledgerTransactionStats, err = s.runner.RunAllProcessorsOnLedger(sequence)
		if err != nil {
			err = errors.Wrap(err, "Error running processors on ledger")
			return stop(), err
		}

		if err = s.completeIngestion(sequence); err != nil {
			return stop(), err
		}

		log.
			WithFields(changeStats.Map()).
			WithFields(ledgerTransactionStats.Map()).
			WithFields(logpkg.F{
				"sequence": sequence,
				"duration": time.Since(startTime).Seconds(),
				"state":    true,
				"ledger":   true,
				"commit":   true,
			}).
			Info("Processed ledger")
	}

	if v.verifyState {
		err = s.verifyState(false)
	}

	return stop(), err
}

type stressTestState struct{}

func (stressTestState) String() string {
	return "stressTest"
}

func (stressTestState) run(s *System) (transition, error) {
	if err := s.historyQ.Begin(); err != nil {
		err = errors.Wrap(err, "Error starting a transaction")
		return stop(), err
	}
	defer s.historyQ.Rollback()

	// Simple check if DB clean
	lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
	if err != nil {
		err = errors.Wrap(err, getLastIngestedErrMsg)
		return stop(), err
	}

	if lastIngestedLedger != 0 {
		err = errors.New("Database not empty")
		return stop(), err
	}

	curHeap, sysHeap := getMemStats()
	sequence := lastIngestedLedger + 1
	log.WithFields(logpkg.F{
		"currentHeapSizeMB": curHeap,
		"systemHeapSizeMB":  sysHeap,
		"sequence":          sequence,
		"state":             true,
		"ledger":            true,
		"commit":            true,
	}).Info("Processing ledger")
	startTime := time.Now()

	changeStats, ledgerTransactionStats, err := s.runner.RunAllProcessorsOnLedger(sequence)
	if err != nil {
		err = errors.Wrap(err, "Error running processors on ledger")
		return stop(), err
	}

	if err = s.completeIngestion(sequence); err != nil {
		return stop(), err
	}

	curHeap, sysHeap = getMemStats()
	log.
		WithFields(changeStats.Map()).
		WithFields(ledgerTransactionStats.Map()).
		WithFields(logpkg.F{
			"currentHeapSizeMB": curHeap,
			"systemHeapSizeMB":  sysHeap,
			"sequence":          sequence,
			"duration":          time.Since(startTime).Seconds(),
			"state":             true,
			"ledger":            true,
			"commit":            true,
		}).
		Info("Processed ledger")

	return stop(), nil
}

func (s *System) completeIngestion(ledger uint32) error {
	if ledger == 0 {
		return errors.New("ledger must be positive")
	}

	if err := s.historyQ.UpdateLastLedgerExpIngest(ledger); err != nil {
		err = errors.Wrap(err, updateLastLedgerExpIngestErrMsg)
		return err
	}

	if err := s.historyQ.Commit(); err != nil {
		return errors.Wrap(err, commitErrMsg)
	}

	return nil
}
