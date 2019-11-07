package history

import (
	"github.com/stretchr/testify/mock"
)

type MockAccountSignersBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockAccountSignersBatchInsertBuilder) Add(signer AccountSigner) error {
	a := m.Called(signer)
	return a.Error(0)
}

func (m *MockAccountSignersBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
