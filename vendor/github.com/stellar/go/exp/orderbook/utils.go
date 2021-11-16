package orderbook

import (
	"github.com/stellar/go/xdr"
)

// contains searches for a string needle in a haystack
func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// getPoolAssets retrieves string representations of a pool's reserves
func getPoolAssets(pool xdr.LiquidityPoolEntry) (string, string) {
	params := pool.Body.MustConstantProduct().Params
	return params.AssetA.String(), params.AssetB.String()
}

func max(a, b xdr.Int64) xdr.Int64 {
	if a < b {
		return b
	}
	return a
}

// positiveMin returns the smallest positive value possible
func positiveMin(a, b xdr.Int64) xdr.Int64 {
	if b <= 0 {
		return a
	}

	if b < a || a <= 0 {
		return b
	}

	return a
}
