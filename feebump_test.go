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

	// Create a transaction from Bob to Charlie with fee=0. We can't sent it
	// without someone else covering the fee.

	// NewBaseTx always clamps the fee to minimum fee, but we can change it
	// later.
	tx2 := NewBaseTx(addressStr(t, helper.Bob), sp, txnbuild.MinBaseFee)
	// Bob has 100 lumens total, trying to spend 99, there is not enough
	// balance in that account to pay the fee.
	tx2.AddPaymentOp(addressStr(t, helper.Charlie), "99")
	tx2.SetBaseFee(0)
	tx2res, err := tx2.Sign(seedStr(t, helper.Bob))
	require.NoError(t, err)

	_, err = Submit(tx2res.Signed)
	require.Error(t, err)

	feeBump, err := FeeBumpTransaction(tx2res.Signed, seedStr(t, helper.Alice), 2*txnbuild.MinBaseFee)
	require.NoError(t, err)

	t.Logf("Fee bump tx: %s", feeBump.Signed)

	res, err := Submit(feeBump.Signed)
	require.NoError(t, err)
	t.Logf("Fee bump tx id is: %s", res.TxID)
}
