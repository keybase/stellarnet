package stellarnet

import (
	"testing"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stellar/go/build"
	"github.com/stretchr/testify/require"
)

func TestMultipleOps(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "multiple_ops")

	testclient.GetTestLumens(t, helper.Alice)
	t.Log("alice account has been funded")

	// make a tx with two create account operations
	tx := NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	tx.AddCreateAccountOp(addressStr(t, helper.Bob), "10")
	tx.AddCreateAccountOp(addressStr(t, helper.Charlie), "20")
	r, err := tx.Sign(seedStr(t, helper.Alice))
	require.NoError(t, err)
	t.Logf("sign result: %+v", r)
	_, err = Submit(r.Signed)
	require.NoError(t, err)

	acctAlice := NewAccount(addressStr(t, helper.Alice))
	acctBob := NewAccount(addressStr(t, helper.Bob))
	acctCharlie := NewAccount(addressStr(t, helper.Charlie))
	balance, err := acctAlice.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "9969.9999800", balance)
	balance, err = acctBob.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "10.0000000", balance)
	balance, err = acctCharlie.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "20.0000000", balance)

	tx = NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	for i := 0; i < 50; i++ {
		tx.AddPaymentOp(addressStr(t, helper.Bob), "1")
		tx.AddPaymentOp(addressStr(t, helper.Charlie), "2")
	}
	r, err = tx.Sign(seedStr(t, helper.Alice))
	require.NoError(t, err)
	t.Logf("sign result: %+v", r)
	_, err = Submit(r.Signed)
	require.NoError(t, err)

	balance, err = acctBob.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "60.0000000", balance)
	balance, err = acctCharlie.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "120.0000000", balance)
	balance, err = acctAlice.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "9819.9979800", balance)

	tx = NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	for i := 0; i < 100; i++ {
		tx.AddPaymentOp(addressStr(t, helper.Bob), "1")
		tx.AddPaymentOp(addressStr(t, helper.Charlie), "2")
	}
	_, err = tx.Sign(seedStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrTxOpFull, err)

	tx = NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	tx.AddMemoText("memo 1")
	tx.AddMemoText("memo 2")
	_, err = tx.Sign(seedStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrMemoExists, err)

	tx = NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	id := uint64(123123123)
	tx.AddMemoID(&id)
	tx.AddMemoText("memo text")
	_, err = tx.Sign(seedStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrMemoExists, err)

	tx = NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	tx.AddTimeBounds(1000, 5000)
	tx.AddTimeBounds(4000, 5000)
	_, err = tx.Sign(seedStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrTimeBoundsExist, err)

	tx = NewBaseTx(addressStr(t, helper.Alice), Client(), build.DefaultBaseFee*2)
	tx.AddTimeBounds(1000, 5000)
	tx.AddMemoText("memo 1")
	_, err = tx.Sign(seedStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrNoOps, err)
}
