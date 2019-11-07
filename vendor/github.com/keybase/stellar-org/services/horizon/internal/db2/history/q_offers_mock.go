package history

import (
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
)

// MockQOffers is a mock implementation of the QOffers interface
type MockQOffers struct {
	mock.Mock
}

func (m *MockQOffers) GetAllOffers() ([]Offer, error) {
	a := m.Called()
	return a.Get(0).([]Offer), a.Error(1)
}

func (m *MockQOffers) NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder {
	a := m.Called(maxBatchSize)
	return a.Get(0).(OffersBatchInsertBuilder)
}

func (m *MockQOffers) InsertOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(offer, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQOffers) UpdateOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	a := m.Called(offer, lastModifiedLedger)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQOffers) RemoveOffer(offerID xdr.Int64) (int64, error) {
	a := m.Called(offerID)
	return a.Get(0).(int64), a.Error(1)
}
