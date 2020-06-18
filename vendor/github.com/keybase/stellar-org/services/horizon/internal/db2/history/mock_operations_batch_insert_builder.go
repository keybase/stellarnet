package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

// MockOperationsBatchInsertBuilder OperationsBatchInsertBuilder mock
type MockOperationsBatchInsertBuilder struct {
	mock.Mock
}

// Add mock
func (m *MockOperationsBatchInsertBuilder) Add(
	id int64,
	transactionID int64,
	applicationOrder uint32,
	operationType xdr.OperationType,
	details []byte,
	sourceAccount string,
) error {
	a := m.Called(
		id,
		transactionID,
		applicationOrder,
		operationType,
		details,
		sourceAccount,
	)
	return a.Error(0)
}

// Exec mock
func (m *MockOperationsBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
