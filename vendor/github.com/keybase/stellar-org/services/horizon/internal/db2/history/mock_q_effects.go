package history

import (
	"github.com/stretchr/testify/mock"
)

// MockQEffects is a mock implementation of the QEffects interface
type MockQEffects struct {
	mock.Mock
}

func (m *MockQEffects) NewEffectBatchInsertBuilder(maxBatchSize int) EffectBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(EffectBatchInsertBuilder)
}

func (m *MockQEffects) CreateAccounts(addresses []string, maxBatchSize int) (map[string]int64, error) {
	a := m.Called(addresses, maxBatchSize)
	return a.Get(0).(map[string]int64), a.Error(1)
}
