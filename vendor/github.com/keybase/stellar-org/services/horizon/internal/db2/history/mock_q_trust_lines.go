package history

import (
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQTrustLines is a mock implementation of the QOffers interface
type MockQTrustLines struct {
	mock.Mock
}

func (m *MockQTrustLines) NewTrustLinesBatchInsertBuilder(maxBatchSize int) TrustLinesBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(TrustLinesBatchInsertBuilder)
}

func (m *MockQTrustLines) GetTrustLinesByKeys(keys []xdr.LedgerKeyTrustLine) ([]TrustLine, error) {
	a := m.Called(keys)
	return a.Get(0).([]TrustLine), a.Error(1)
}

func (m *MockQTrustLines) InsertTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(trustLine, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQTrustLines) UpdateTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(trustLine, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQTrustLines) UpsertTrustLines(trustLines []xdr.LedgerEntry) error {
	a := m.Called(trustLines)
	return a.Error(0)
}

func (m *MockQTrustLines) RemoveTrustLine(key xdr.LedgerKeyTrustLine) (int64, error) {
	a := m.Called(key)
	return a.Get(0).(int64), a.Error(1)
}
