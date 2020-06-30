package stellarnet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stellar/go/txnbuild"
)

func TestBumpFeeTx(t *testing.T) {
	helper, client, tnetwork := testclient.Setup(t)
	SetClientAndNetwork(client, tnetwork)
	helper.SetState(t, "feebump")

	testclient.GetTestLumens(t, helper.Alice)

	sp := ClientSequenceProvider{Client: Client()}

	tx, err := CreateAccountXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
		"100", "" /* memoText */, sp, nil /* timeBounds */, txnbuild.MinBaseFee)
	require.NoError(t, err)
	_, err = Submit(tx.Signed)
	require.NoError(t, err)

	tx, err = CreateAccountXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Charlie),
		"5", "" /* memoText */, sp, nil /* timeBounds */, txnbuild.MinBaseFee)
	require.NoError(t, err)
	_, err = Submit(tx.Signed)
	require.NoError(t, err)

	tx, err = PaymentXLMTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Charlie),
		"99", "" /* memoText */, sp, nil /* timeBounds */, 0)
	require.NoError(t, err)

	feeBump, err := FeeBumpTransaction(tx.Signed, seedStr(t, helper.Alice), 2*txnbuild.MinBaseFee)
	require.NoError(t, err)

	t.Logf("Fee bump tx: %s", feeBump.Signed)

	res, err := Submit(feeBump.Signed)
	require.NoError(t, err)
	t.Logf("Fee bump tx id is: %s", res.TxID)
}
