package ledger

import (
	"context"
	"time"
)

// Source exposes two helpers methods to help you find out the current
// ledger and yield every time there is a new ledger.
type Source interface {
	CurrentLedger() uint32
	NextLedger(currentSequence uint32) chan uint32
}

type currentStateFunc func() State

// HistoryDBSource utility struct to pass the SSE update frequency and a
// function to get the current ledger state.
type HistoryDBSource struct {
	updateFrequency time.Duration
	currentState    currentStateFunc
}

// NewHistoryDBSource constructs a new instance of HistoryDBSource
func NewHistoryDBSource(updateFrequency time.Duration) HistoryDBSource {
	return HistoryDBSource{
		updateFrequency: updateFrequency,
		currentState:    CurrentState,
	}
}

// CurrentLedger returns the current ledger.
func (source HistoryDBSource) CurrentLedger() uint32 {
	return source.currentState().ExpHistoryLatest
}

// NextLedger returns a channel which yields every time there is a new ledger with a sequence number larger than currentSequence.
func (source HistoryDBSource) NextLedger(currentSequence uint32) chan uint32 {
	// Make sure this is buffered channel of size 1. Otherwise, the go routine below
	// will never return if `newLedgers` channel is not read. From Effective Go:
	// > If the channel is unbuffered, the sender blocks until the receiver has received the value.
	newLedgers := make(chan uint32, 1)
	go func() {
		for {
			if source.updateFrequency > 0 {
				time.Sleep(source.updateFrequency)
			}

			currentLedgerState := source.currentState()
			if currentLedgerState.ExpHistoryLatest > currentSequence {
				newLedgers <- currentLedgerState.ExpHistoryLatest
				return
			}
		}
	}()

	return newLedgers
}

// TestingSource is helper struct which implements the LedgerSource
// interface.
type TestingSource struct {
	currentLedger uint32
	newLedgers    chan uint32
}

// NewTestingSource returns a TestingSource.
func NewTestingSource(currentLedger uint32) *TestingSource {
	return &TestingSource{
		currentLedger: currentLedger,
		newLedgers:    make(chan uint32),
	}
}

// CurrentLedger returns the current ledger.
func (source *TestingSource) CurrentLedger() uint32 {
	return source.currentLedger
}

// AddLedger adds a new sequence to the newLedgers channel. AddLedger()
// will block until the new sequence is read
func (source *TestingSource) AddLedger(nextSequence uint32) {
	source.newLedgers <- nextSequence
}

// TryAddLedger sends a new sequence to the newLedgers channel. TryAddLedger()
// will block until whichever of the following events occur first:
// * the given ctx terminates
// * timeout has elapsed
// * the new sequence is read from the newLedgers channel
// TryAddLedger() returns true if the new sequence was read from the newLedgers channel
func (source *TestingSource) TryAddLedger(
	ctx context.Context,
	nextSequence uint32,
	timeout time.Duration,
) bool {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
	defer cancel()
	select {
	case source.newLedgers <- nextSequence:
		return true
	case <-ctx.Done():
		return false
	}
}

// NextLedger returns a channel which yields every time there is a new ledger.
func (source *TestingSource) NextLedger(currentSequence uint32) chan uint32 {
	return source.newLedgers
}
