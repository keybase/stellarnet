package corestate

import (
	"sync"

	"github.com/stellar/go/protocols/stellarcore"
)

type State struct {
	Synced                       bool
	CurrentProtocolVersion       int32
	CoreSupportedProtocolVersion int32
	CoreVersion                  string
}

type Store struct {
	sync.RWMutex
	state State
}

func (c *Store) Set(resp *stellarcore.InfoResponse) {
	c.Lock()
	defer c.Unlock()
	c.state.Synced = resp.IsSynced()
	c.state.CoreVersion = resp.Info.Build
	c.state.CurrentProtocolVersion = int32(resp.Info.Ledger.Version)
	c.state.CoreSupportedProtocolVersion = int32(resp.Info.ProtocolVersion)
}

func (c *Store) SetState(state State) {
	c.Lock()
	defer c.Unlock()
	c.state = state
}

func (c *Store) Get() State {
	c.RLock()
	defer c.RUnlock()
	return c.state
}
