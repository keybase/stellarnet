package history

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stretchr/testify/mock"
)

type MockTransactionsBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTransactionsBatchInsertBuilder) Add(transaction io.LedgerTransaction, sequence uint32) error {
	a := m.Called(transaction, sequence)
	return a.Error(0)
}

func (m *MockTransactionsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
