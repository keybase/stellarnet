package mocks

import (
	"github.com/stellar/go/clients/stellartoml"
	"github.com/stretchr/testify/mock"
)

// MockStellartomlResolver ...
type MockStellartomlResolver struct {
	mock.Mock
}

// GetStellarToml is a mocking a method
func (m *MockStellartomlResolver) GetStellarToml(domain string) (resp *stellartoml.Response, err error) {
	a := m.Called(domain)
	return a.Get(0).(*stellartoml.Response), a.Error(1)
}

// GetStellarTomlByAddress is a mocking a method
func (m *MockStellartomlResolver) GetStellarTomlByAddress(addy string) (*stellartoml.Response, error) {
	a := m.Called(addy)
	return a.Get(0).(*stellartoml.Response), a.Error(1)
}
