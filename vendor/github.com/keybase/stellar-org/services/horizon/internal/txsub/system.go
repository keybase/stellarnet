package txsub

import (
	"context"
	"fmt"
	"github.com/stellar/go/xdr"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/log"
)

// System represents a completely configured transaction submission system.
// Its methods tie together the various pieces used to reliably submit transactions
// to a stellar-core instance.
type System struct {
	initializer sync.Once

	tickMutex      sync.Mutex
	tickInProgress bool

	Pending           OpenSubmissionList
	Results           ResultProvider
	Sequences         SequenceProvider
	Submitter         Submitter
	SubmissionQueue   *sequence.Manager
	SubmissionTimeout time.Duration
	Log               *log.Entry

	Metrics struct {
		// SubmissionTimer exposes timing metrics about the rate and latency of
		// submissions to stellar-core
		SubmissionTimer metrics.Timer

		// BufferedSubmissionGauge tracks the count of submissions buffered
		// behind this system's SubmissionQueue
		BufferedSubmissionsGauge metrics.Gauge

		// OpenSubmissionsGauge tracks the count of "open" submissions (i.e.
		// submissions whose transactions haven't been confirmed successful or failed
		OpenSubmissionsGauge metrics.Gauge

		// FailedSubmissionsMeter tracks the rate of failed transactions that have
		// been submitted to this process
		FailedSubmissionsMeter metrics.Meter

		// SuccessfulSubmissionsMeter tracks the rate of successful transactions that
		// have been submitted to this process
		SuccessfulSubmissionsMeter metrics.Meter

		// V0TransactionsMeter tracks the rate of v0 transaction envelopes that
		// have been submitted to this process
		V0TransactionsMeter metrics.Meter

		// V1TransactionsMeter tracks the rate of v1 transaction envelopes that
		// have been submitted to this process
		V1TransactionsMeter metrics.Meter

		// FeeBumpTransactionsMeter tracks the rate of fee bump transaction envelopes that
		// have been submitted to this process
		FeeBumpTransactionsMeter metrics.Meter
	}
}

// Submit submits the provided base64 encoded transaction envelope to the
// network using this submission system.
func (sys *System) Submit(
	ctx context.Context,
	rawTx string,
	envelope xdr.TransactionEnvelope,
	hash string,
) (result <-chan Result) {
	sys.Init()
	response := make(chan Result, 1)
	result = response

	sys.Log.Ctx(ctx).WithFields(log.F{
		"hash":    hash,
		"tx_type": envelope.Type.String(),
		"tx":      rawTx,
	}).Info("Processing transaction")

	// check the configured result provider for an existing result
	r := sys.Results.ResultByHash(ctx, hash)

	if r.Err == nil {
		sys.Log.Ctx(ctx).WithField("hash", hash).Info("Found submission result in a DB")
		sys.finish(ctx, hash, response, r)
		return
	}

	if r.Err != ErrNoResults {
		sys.Log.Ctx(ctx).WithField("hash", hash).Info("Error getting submission result from a DB")
		sys.finish(ctx, hash, response, r)
		return
	}

	// From now: r.Err == ErrNoResults
	sourceAccount := envelope.SourceAccount()
	// The database doesn't (yet) store muxed accounts, so we query
	// the corresponding AccountId
	accid := sourceAccount.ToAccountId()
	sourceAddress := accid.Address()
	curSeq, err := sys.Sequences.GetSequenceNumbers([]string{sourceAddress})
	if err != nil {
		sys.finish(ctx, hash, response, Result{Err: err})
		return
	}

	// If account's sequence cannot be found, abort with tx_NO_ACCOUNT
	// error code
	if _, ok := curSeq[sourceAddress]; !ok {
		sys.finish(ctx, hash, response, Result{Err: ErrNoAccount})
		return
	}

	// queue the submission and get the channel that will emit when
	// submission is valid
	seq := sys.SubmissionQueue.Push(sourceAddress, uint64(envelope.SeqNum()))

	// update the submission queue with the source accounts current sequence value
	// which will cause the channel returned by Push() to emit if possible.
	sys.SubmissionQueue.Update(curSeq)

	select {
	case err := <-seq:
		if err == sequence.ErrBadSequence {
			// convert the internal only ErrBadSequence into the FailedTransactionError
			err = ErrBadSequence
		}

		if err != nil {
			sys.finish(ctx, hash, response, Result{Err: err})
			return
		}

		sr := sys.submitOnce(ctx, rawTx)
		sys.updateTransactionTypeMetrics(envelope)

		// if submission succeeded
		if sr.Err == nil {
			// add transactions to open list
			sys.Pending.Add(ctx, hash, response)
			// update the submission queue, allowing the next submission to proceed
			sys.SubmissionQueue.Update(map[string]uint64{
				sourceAddress: uint64(envelope.SeqNum()),
			})
			return
		}

		// any error other than "txBAD_SEQ" is a failure
		isBad, err := sr.IsBadSeq()
		if err != nil {
			sys.finish(ctx, hash, response, Result{Err: err})
			return
		}

		if !isBad {
			sys.finish(ctx, hash, response, Result{Err: sr.Err})
			return
		}

		// If error is txBAD_SEQ, check for the result again
		r = sys.Results.ResultByHash(ctx, hash)

		if r.Err == nil {
			// If the found use it as the result
			sys.finish(ctx, hash, response, r)
		} else {
			// finally, return the bad_seq error if no result was found on 2nd attempt
			sys.finish(ctx, hash, response, Result{Err: sr.Err})
		}

	case <-ctx.Done():
		sys.finish(ctx, hash, response, Result{Err: ErrCanceled})
	}

	return
}

// Submit submits the provided base64 encoded transaction envelope to the
// network using this submission system.
func (sys *System) submitOnce(ctx context.Context, env string) SubmissionResult {
	// submit to stellar-core
	sr := sys.Submitter.Submit(ctx, env)
	sys.Metrics.SubmissionTimer.Update(sr.Duration)

	// if received or duplicate, add to the open submissions list
	if sr.Err == nil {
		sys.Metrics.SuccessfulSubmissionsMeter.Mark(1)
	} else {
		sys.Metrics.FailedSubmissionsMeter.Mark(1)
	}

	return sr
}

func (sys *System) updateTransactionTypeMetrics(envelope xdr.TransactionEnvelope) {
	switch envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTxV0:
		sys.Metrics.V0TransactionsMeter.Mark(1)
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sys.Metrics.V1TransactionsMeter.Mark(1)
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		sys.Metrics.FeeBumpTransactionsMeter.Mark(1)
	}
}

// setTickInProgress sets `tickInProgress` to `true` if it's
// `false`. Returns `true` if `tickInProgress` has been switched
// to `true` inside this method and `Tick()` should continue.
func (sys *System) setTickInProgress(ctx context.Context) bool {
	sys.tickMutex.Lock()
	defer sys.tickMutex.Unlock()

	if sys.tickInProgress {
		logger := log.Ctx(ctx)
		logger.Info("ticking in progress")
		return false
	}

	sys.tickInProgress = true
	return true
}

func (sys *System) unsetTickInProgress() {
	sys.tickMutex.Lock()
	defer sys.tickMutex.Unlock()
	sys.tickInProgress = false
}

// Tick triggers the system to update itself with any new data available.
func (sys *System) Tick(ctx context.Context) {
	sys.Init()
	logger := log.Ctx(ctx)

	// Make sure Tick is not run concurrently
	if !sys.setTickInProgress(ctx) {
		return
	}

	defer sys.unsetTickInProgress()

	logger.
		WithField("queued", sys.SubmissionQueue.String()).
		Debug("ticking txsub system")

	addys := sys.SubmissionQueue.Addresses()
	if len(addys) > 0 {
		curSeq, err := sys.Sequences.GetSequenceNumbers(addys)
		if err != nil {
			logger.WithStack(err).Error(err)
			return
		} else {
			sys.SubmissionQueue.Update(curSeq)
		}
	}

	for _, hash := range sys.Pending.Pending(ctx) {
		r := sys.Results.ResultByHash(ctx, hash)

		if r.Err == nil {
			logger.WithField("hash", hash).Debug("finishing open submission")
			sys.Pending.Finish(ctx, hash, r)
			continue
		}

		_, ok := r.Err.(*FailedTransactionError)

		if ok {
			logger.WithField("hash", hash).Debug("finishing open submission")
			sys.Pending.Finish(ctx, hash, r)
			continue
		}

		if r.Err != ErrNoResults {
			logger.WithStack(r.Err).Error(r.Err)
		}
	}

	stillOpen, err := sys.Pending.Clean(ctx, sys.SubmissionTimeout)
	if err != nil {
		logger.WithStack(err).Error(err)
		return
	}

	sys.Metrics.OpenSubmissionsGauge.Update(int64(stillOpen))
	sys.Metrics.BufferedSubmissionsGauge.Update(int64(sys.SubmissionQueue.Size()))
}

// Init initializes `sys`
func (sys *System) Init() {
	sys.initializer.Do(func() {
		sys.Log = log.DefaultLogger.WithField("service", "txsub.System")

		sys.Metrics.FailedSubmissionsMeter = metrics.NewMeter()
		sys.Metrics.SuccessfulSubmissionsMeter = metrics.NewMeter()
		sys.Metrics.SubmissionTimer = metrics.NewTimer()
		sys.Metrics.OpenSubmissionsGauge = metrics.NewGauge()
		sys.Metrics.BufferedSubmissionsGauge = metrics.NewGauge()
		sys.Metrics.V0TransactionsMeter = metrics.NewMeter()
		sys.Metrics.V1TransactionsMeter = metrics.NewMeter()
		sys.Metrics.FeeBumpTransactionsMeter = metrics.NewMeter()

		if sys.SubmissionTimeout == 0 {
			// HTTP clients in SDKs usually timeout in 60 seconds. We want SubmissionTimeout
			// to be lower than that to make sure that they read the response before the client
			// timeout.
			// 30 seconds is 6 ledgers (with avg. close time = 5 sec), enough for stellar-core
			// to drop the transaction if not added to the ledger and ask client to try again
			// by sending a Timeout response.
			sys.SubmissionTimeout = 30 * time.Second
		}
	})
}

func (sys *System) finish(ctx context.Context, hash string, response chan<- Result, r Result) {
	sys.Log.Ctx(ctx).
		WithField("result", fmt.Sprintf("%+v", r)).
		WithField("hash", hash).
		Info("Submission system result")
	response <- r
	close(response)
}
