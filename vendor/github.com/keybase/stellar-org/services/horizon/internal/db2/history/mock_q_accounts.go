package history

import (
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQAccounts is a mock implementation of the QAccounts interface
type MockQAccounts struct {
	mock.Mock
}

func (m *MockQAccounts) GetAccountsByIDs(ids []string) ([]AccountEntry, error) {
	a := m.Called(ids)
	return a.Get(0).([]AccountEntry), a.Error(1)
}

func (m *MockQAccounts) NewAccountsBatchInsertBuilder(maxBatchSize int) AccountsBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(AccountsBatchInsertBuilder)
}

func (m *MockQAccounts) InsertAccount(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(account, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQAccounts) UpdateAccount(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(account, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQAccounts) UpsertAccounts(accounts []xdr.LedgerEntry) error {
	a := m.Called(accounts)
	return a.Error(0)
}

func (m *MockQAccounts) RemoveAccount(accountID string) (int64, error) {
	a := m.Called(accountID)
	return a.Get(0).(int64), a.Error(1)
}
