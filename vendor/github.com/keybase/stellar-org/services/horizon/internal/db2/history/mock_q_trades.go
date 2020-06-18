package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockQTrades struct {
	mock.Mock
}

func (m *MockQTrades) CreateAccounts(addresses []string, maxBatchSize int) (map[string]int64, error) {
	a := m.Called(addresses, maxBatchSize)
	return a.Get(0).(map[string]int64), a.Error(1)
}

func (m *MockQTrades) CreateAssets(assets []xdr.Asset, maxBatchSize int) (map[string]Asset, error) {
	a := m.Called(assets, maxBatchSize)
	return a.Get(0).(map[string]Asset), a.Error(1)
}

func (m *MockQTrades) NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(TradeBatchInsertBuilder)
}

type MockTradeBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockTradeBatchInsertBuilder) Add(entries ...InsertTrade) error {
	a := m.Called(entries)
	return a.Error(0)
}

func (m *MockTradeBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
