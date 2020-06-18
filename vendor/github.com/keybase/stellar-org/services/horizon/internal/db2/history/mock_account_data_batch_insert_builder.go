package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockAccountDataBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockAccountDataBatchInsertBuilder) Add(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) error {
	a := m.Called(data, lastModifiedLedger)
	return a.Error(0)
}

func (m *MockAccountDataBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
