package io

import "github.com/stretchr/testify/mock"

var _ LedgerTransactionProcessor = (*MockLedgerTransactionProcessor)(nil)

type MockLedgerTransactionProcessor struct {
	mock.Mock
}

func (m *MockLedgerTransactionProcessor) ProcessTransaction(transaction LedgerTransaction) error {
	args := m.Called(transaction)
	return args.Error(0)
}
