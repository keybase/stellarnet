package io

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

var _ LedgerReader = (*MockLedgerReader)(nil)

type MockLedgerReader struct {
	mock.Mock
}

func (m *MockLedgerReader) GetSequence() uint32 {
	args := m.Called()
	return args.Get(0).(uint32)
}

func (m *MockLedgerReader) GetHeader() xdr.LedgerHeaderHistoryEntry {
	args := m.Called()
	return args.Get(0).(xdr.LedgerHeaderHistoryEntry)
}

func (m *MockLedgerReader) Read() (LedgerTransaction, error) {
	args := m.Called()
	return args.Get(0).(LedgerTransaction), args.Error(1)
}

func (m *MockLedgerReader) ReadUpgradeChange() (Change, error) {
	args := m.Called()
	return args.Get(0).(Change), args.Error(1)
}

func (m *MockLedgerReader) GetUpgradeChanges() []Change {
	args := m.Called()
	return args.Get(0).([]Change)
}

func (m *MockLedgerReader) IgnoreUpgradeChanges() {
	m.Called()
}

func (m *MockLedgerReader) Close() error {
	args := m.Called()
	return args.Error(0)
}
