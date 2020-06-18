package history

import (
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

// MockQAssetStats is a mock implementation of the QAssetStats interface
type MockQAssetStats struct {
	mock.Mock
}

func (m *MockQAssetStats) InsertAssetStats(assetStats []ExpAssetStat, batchSize int) error {
	a := m.Called(assetStats, batchSize)
	return a.Error(0)
}

func (m *MockQAssetStats) InsertAssetStat(assetStat ExpAssetStat) (int64, error) {
	a := m.Called(assetStat)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQAssetStats) UpdateAssetStat(assetStat ExpAssetStat) (int64, error) {
	a := m.Called(assetStat)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQAssetStats) GetAssetStat(assetType xdr.AssetType, assetCode, assetIssuer string) (ExpAssetStat, error) {
	a := m.Called(assetType, assetCode, assetIssuer)
	return a.Get(0).(ExpAssetStat), a.Error(1)
}

func (m *MockQAssetStats) RemoveAssetStat(assetType xdr.AssetType, assetCode, assetIssuer string) (int64, error) {
	a := m.Called(assetType, assetCode, assetIssuer)
	return a.Get(0).(int64), a.Error(1)
}

func (m *MockQAssetStats) GetAssetStats(assetCode, assetIssuer string, page db2.PageQuery) ([]ExpAssetStat, error) {
	a := m.Called(assetCode, assetIssuer, page)
	return a.Get(0).([]ExpAssetStat), a.Error(1)
}

func (m *MockQAssetStats) CountTrustLines() (int, error) {
	a := m.Called()
	return a.Get(0).(int), a.Error(1)
}
