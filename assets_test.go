package stellarnet

import (
	"fmt"
	"testing"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stretchr/testify/require"
)

func TestAsset(t *testing.T) {
	t.Skip("no guarantee what assets are on testnet, this test just here for manual verification")
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "asset")

	summary, err := Asset("EUR", "GA4QRYQ43TFNT6JCH4AVVZD6RHR2I3KC55UENZBP3H2Z6FH6JJDSFUDW")
	require.NoError(t, err)

	fmt.Printf("summary: %+v\n", summary)
}
