package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockAccountsBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockAccountsBatchInsertBuilder) Add(accounts xdr.AccountEntry, lastModifiedLedger xdr.Uint32) error {
	a := m.Called(accounts, lastModifiedLedger)
	return a.Error(0)
}

func (m *MockAccountsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
