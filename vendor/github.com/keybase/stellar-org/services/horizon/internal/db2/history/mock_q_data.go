package history

import (
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQData is a mock implementation of the QAccounts interface
type MockQData struct {
	mock.Mock
}

func (m *MockQData) GetAccountDataByKeys(keys []xdr.LedgerKeyData) ([]Data, error) {
	a := m.Called()
	return a.Get(0).([]Data), a.Error(1)
}

func (m *MockQData) CountAccountsData() (int, error) {
	a := m.Called()
	return a.Get(0).(int), a.Error(1)
}

func (m *MockQData) NewAccountDataBatchInsertBuilder(maxBatchSize int) AccountDataBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(AccountDataBatchInsertBuilder)
}

func (m *MockQData) InsertAccountData(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(data, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQData) UpdateAccountData(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(data, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQData) RemoveAccountData(key xdr.LedgerKeyData) (int64, error) {
	a := m.Called(key)
	return a.Get(0).(int64), a.Error(1)
}
