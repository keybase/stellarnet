package expingest

import (
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

var _ orderbook.OBGraph = (*mockOrderBookGraph)(nil)

type mockOrderBookGraph struct {
	mock.Mock
}

func (m *mockOrderBookGraph) AddOffer(offer xdr.OfferEntry) {
	m.Called(offer)
}

func (m *mockOrderBookGraph) Apply(ledger uint32) error {
	args := m.Called(ledger)
	return args.Error(0)

}

func (m *mockOrderBookGraph) Pending() ([]xdr.OfferEntry, []xdr.Int64) {
	args := m.Called()
	return args.Get(0).([]xdr.OfferEntry), args.Get(1).([]xdr.Int64)
}

func (m *mockOrderBookGraph) Discard() {
	m.Called()
}

func (m *mockOrderBookGraph) Offers() []xdr.OfferEntry {
	args := m.Called()
	return args.Get(0).([]xdr.OfferEntry)
}

func (m *mockOrderBookGraph) OffersMap() map[xdr.Int64]xdr.OfferEntry {
	args := m.Called()
	return args.Get(0).(map[xdr.Int64]xdr.OfferEntry)
}

func (m *mockOrderBookGraph) RemoveOffer(id xdr.Int64) orderbook.OBGraph {
	args := m.Called(id)
	return args.Get(0).(orderbook.OBGraph)
}

func (m *mockOrderBookGraph) Clear() {
	m.Called()
}
