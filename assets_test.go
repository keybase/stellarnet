package stellarnet

import (
	"testing"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stretchr/testify/require"
)

func TestAsset(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "asset")

	t.Logf("If this test fails, test asset might have disappeared.  see assets_test.go.")
	// Note: there is no guarantee that this asset code/issuer combo will
	// continue to work on horizon testnet.  It is recorded, but if you change
	// this test and it fails, find a new asset to check.
	//
	// https://horizon-testnet.stellar.org/assets?asset_code=EUR
	//
	// will show a list of EUR assets.  Pick one.
	//
	summary, err := Asset("EUR", "GA4QRYQ43TFNT6JCH4AVVZD6RHR2I3KC55UENZBP3H2Z6FH6JJDSFUDW")
	require.NoError(t, err)

	require.Equal(t, "credit_alphanum4", summary.AssetType)
	require.Equal(t, "EUR", summary.AssetCode)
	require.Equal(t, "EUR", summary.AssetCode)
	require.Equal(t, "GA4QRYQ43TFNT6JCH4AVVZD6RHR2I3KC55UENZBP3H2Z6FH6JJDSFUDW", summary.AssetIssuer)
	require.Empty(t, summary.UnverifiedWellKnownLink)
}
