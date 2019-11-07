package history

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockOffersBatchInsertBuilder struct {
	mock.Mock
}

func (m *MockOffersBatchInsertBuilder) Add(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) error {
	a := m.Called(offer, lastModifiedLedger)
	return a.Error(0)
}

func (m *MockOffersBatchInsertBuilder) Exec() error {
	a := m.Called()
	return a.Error(0)
}
