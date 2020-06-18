package io

import "github.com/stretchr/testify/mock"

var _ ChangeProcessor = (*MockChangeProcessor)(nil)

type MockChangeProcessor struct {
	mock.Mock
}

func (m *MockChangeProcessor) ProcessChange(change Change) error {
	args := m.Called(change)
	return args.Error(0)
}
