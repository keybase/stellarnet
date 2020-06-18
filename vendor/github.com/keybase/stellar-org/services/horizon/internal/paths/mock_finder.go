package paths

import (
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

var _ Finder = (*MockFinder)(nil)

// MockFinder is a mock implementation of the Finder interface
type MockFinder struct {
	mock.Mock
}

func (m *MockFinder) Find(q Query, maxLength uint) ([]Path, uint32, error) {
	args := m.Called(q, maxLength)

	return args.Get(0).([]Path), args.Get(1).(uint32), args.Error(2)
}

func (m *MockFinder) FindFixedPaths(
	sourceAsset xdr.Asset,
	amountToSpend xdr.Int64,
	destinationAssets []xdr.Asset,
	maxLength uint,
) ([]Path, uint32, error) {
	args := m.Called(sourceAsset, amountToSpend, destinationAssets, maxLength)

	return args.Get(0).([]Path), args.Get(1).(uint32), args.Error(2)
}
