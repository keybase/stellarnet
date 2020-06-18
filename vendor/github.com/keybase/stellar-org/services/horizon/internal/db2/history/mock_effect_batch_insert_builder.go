package history

import (
	"github.com/stretchr/testify/mock"
)

// MockEffectBatchInsertBuilder mock EffectBatchInsertBuilder
type MockEffectBatchInsertBuilder struct {
	mock.Mock
}

// Add mock
func (m *MockEffectBatchInsertBuilder) Add(
	accountID int64,
	operationID int64,
	order uint32,
	effectType EffectType,
	details []byte,
) error {
	a := m.Called(
		accountID,
		operationID,
		order,
		effectType,
		details,
	)
	return a.Error(0)
}

// Exec mock
func (m *MockEffectBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
