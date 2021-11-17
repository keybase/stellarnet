// Package ledger provides useful utilities concerning ledgers within stellar,
// specifically as a central location to store a cached snapshot of the state of
// both horizon's and stellar-core's views of the ledger.  This package is
// intended to be at the lowest levels of horizon's dependency tree, please keep
// it free of dependencies to other horizon packages.
package ledger

import (
	"sync"
	"time"
)

// Status represents a snapshot of both horizon's and stellar-core's view of the
// ledger.
type Status struct {
	CoreStatus
	HorizonStatus
}

type CoreStatus struct {
	CoreLatest int32 `db:"core_latest"`
}

type HorizonStatus struct {
	HistoryLatest         int32     `db:"history_latest"`
	HistoryLatestClosedAt time.Time `db:"history_latest_closed_at"`
	HistoryElder          int32     `db:"history_elder"`
	ExpHistoryLatest      uint32    `db:"exp_history_latest"`
}

// State is an in-memory data structure which holds a snapshot of both
// horizon's and stellar-core's view of the the network
type State struct {
	sync.RWMutex
	current Status
}

// CurrentStatus returns the cached snapshot of ledger state
func (c *State) CurrentStatus() Status {
	c.RLock()
	defer c.RUnlock()
	ret := c.current
	return ret
}

// SetStatus updates the cached snapshot of the ledger state
func (c *State) SetStatus(next Status) {
	c.Lock()
	defer c.Unlock()
	c.current = next
}

// SetCoreStatus updates the cached snapshot of the ledger state of Stellar-Core
func (c *State) SetCoreStatus(next CoreStatus) {
	c.Lock()
	defer c.Unlock()
	c.current.CoreStatus = next
}

// SetHorizonStatus updates the cached snapshot of the ledger state of Horizon
func (c *State) SetHorizonStatus(next HorizonStatus) {
	c.Lock()
	defer c.Unlock()
	c.current.HorizonStatus = next
}
