package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockTrustLinesBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTrustLinesBatchInsertBuilder) Add(trustLines xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) error {
	a := m.Called(trustLines, lastModifiedLedger)
	return a.Error(0)
}

func (m *MockTrustLinesBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
