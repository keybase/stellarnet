package processors

import (
	"math/big"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type assetStatKey struct {
	assetType   xdr.AssetType
	assetCode   string
	assetIssuer string
}
type assetStatValue struct {
	amount      *big.Int
	numAccounts int32
}

// AssetStatSet represents a collection of asset stats
type AssetStatSet map[assetStatKey]*assetStatValue

// Add updates the set with a trustline entry from a history archive snapshot
// if the trustline is authorized.
func (s AssetStatSet) Add(trustLine xdr.TrustLineEntry) error {
	if !xdr.TrustLineFlags(trustLine.Flags).IsAuthorized() {
		return nil
	}

	return s.AddDelta(trustLine.Asset, int64(trustLine.Balance), 1)
}

// AddDelta adds a delta balance and delta accounts to a given asset.
func (s AssetStatSet) AddDelta(asset xdr.Asset, deltaBalance int64, deltaAccounts int32) error {
	if deltaBalance == 0 && deltaAccounts == 0 {
		return nil
	}

	var key assetStatKey
	if err := asset.Extract(&key.assetType, &key.assetCode, &key.assetIssuer); err != nil {
		return errors.Wrap(err, "could not extract asset info from trustline")
	}

	current, ok := s[key]
	if !ok {
		s[key] = &assetStatValue{
			amount:      big.NewInt(int64(deltaBalance)),
			numAccounts: deltaAccounts,
		}
	} else {
		current.amount.Add(current.amount, big.NewInt(int64(deltaBalance)))
		current.numAccounts += deltaAccounts
		// Note: it's possible that after operations above:
		// numAccounts != 0 && amount == 0 (ex. two accounts send some of their assets to third account)
		//  OR
		// numAccounts == 0 && amount != 0 (ex. issuer issued an asset)
		if current.numAccounts == 0 && current.amount.Cmp(big.NewInt(0)) == 0 {
			delete(s, key)
		}
	}

	return nil
}

// Remove deletes an asset stat from the set
func (s AssetStatSet) Remove(assetType xdr.AssetType, assetCode string, assetIssuer string) (history.ExpAssetStat, bool) {
	key := assetStatKey{assetType: assetType, assetIssuer: assetIssuer, assetCode: assetCode}
	value, ok := s[key]
	if !ok {
		return history.ExpAssetStat{}, false
	}

	delete(s, key)

	return history.ExpAssetStat{
		AssetType:   key.assetType,
		AssetCode:   key.assetCode,
		AssetIssuer: key.assetIssuer,
		Amount:      value.amount.String(),
		NumAccounts: value.numAccounts,
	}, true
}

// All returns a list of all `history.ExpAssetStat` contained within the set
func (s AssetStatSet) All() []history.ExpAssetStat {
	assetStats := make([]history.ExpAssetStat, 0, len(s))
	for key, value := range s {
		assetStats = append(assetStats, history.ExpAssetStat{
			AssetType:   key.assetType,
			AssetCode:   key.assetCode,
			AssetIssuer: key.assetIssuer,
			Amount:      value.amount.String(),
			NumAccounts: value.numAccounts,
		})
	}
	return assetStats
}
