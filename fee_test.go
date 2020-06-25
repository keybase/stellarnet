package stellarnet

import (
	"testing"

	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/require"

	"github.com/keybase/stellarnet/testclient"
)

// TestFeeStats makes sure that the horizon fetcher works, is decodable, and
// returns non-zero values for some important fields.
func TestFeeStats(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "fee_stats")

	stats, err := FeeStats(&HorizonFeeStatFetcher{})
	require.NoError(t, err)
	require.Greater(t, stats.LastLedger, int32(0))
	require.Greater(t, stats.LedgerCapacityUsage, 0.0)
	require.Greater(t, stats.P95AcceptedFee, uint64(0))
	require.Greater(t, stats.MinAcceptedFee, uint64(0))
	require.Greater(t, stats.ModeAcceptedFee, uint64(0))

	// Fee can't be lower than MinBaseFee, expect to see that in the stats.
	require.EqualValues(t, txnbuild.MinBaseFee, stats.MinAcceptedFee)

	// This one is usually the case on testnet as well.
	require.EqualValues(t, txnbuild.MinBaseFee, stats.P10AcceptedFee)
}
