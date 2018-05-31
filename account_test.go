package stellarnet

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/keybase/stellarnet/testclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
)

func seedStr(t *testing.T, full *keypair.Full) SeedStr {
	ss, err := NewSeedStr(full.Seed())
	if err != nil {
		t.Fatal(err)
	}
	return ss
}

func addressStr(t *testing.T, full *keypair.Full) AddressStr {
	as, err := NewAddressStr(full.Address())
	if err != nil {
		t.Fatal(err)
	}
	return as
}

func assertPayment(t *testing.T, tx Transaction, amount, from, to string) {
	if len(tx.Operations) == 0 {
		t.Fatal("no operations")
	}
	op := tx.Operations[0]
	if op.Type != "payment" {
		t.Fatalf("op type: %s, expected payment", op.Type)
	}
	if op.Amount != amount {
		t.Fatalf("amount: %s, expected %s", op.Amount, amount)
	}
	if op.From != from {
		t.Fatalf("from: %s, expected %s", op.From, from)
	}
	if op.To != to {
		t.Fatalf("to: %s, expected %s", op.To, to)
	}
}

func assertCreateAccount(t *testing.T, tx Transaction, startingBalance, funder, account string) {
	if len(tx.Operations) == 0 {
		t.Fatal("no operations")
	}
	op := tx.Operations[0]
	if op.Type != "create_account" {
		t.Fatalf("op type: %s, expected create_account", op.Type)
	}
	if op.StartingBalance != startingBalance {
		t.Fatalf("starting balance: %s, expected %s", op.StartingBalance, startingBalance)
	}
	if op.Funder != funder {
		t.Fatalf("funder: %s, expected %s", op.Funder, funder)
	}
	if op.Account != account {
		t.Fatalf("account: %s, expected %s", op.Account, account)
	}
}

func TestScenario(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "scenario")

	t.Log("alice key pair not an account yet")
	acctAlice := NewAccount(addressStr(t, helper.Alice))
	_, err := acctAlice.BalanceXLM()
	if err != ErrAccountNotFound {
		t.Fatalf("error: %q, expected %q (ErrAccountNotFound)", err, ErrAccountNotFound)
	}

	_, err = AccountSeqno(addressStr(t, helper.Alice))
	if err != ErrAccountNotFound {
		t.Fatalf("error: %q, expected %q (ErrAccountNotFound)", err, ErrAccountNotFound)
	}

	active, err := IsMasterKeyActive(addressStr(t, helper.Alice))
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("not active")
	}

	testclient.GetTestLumens(t, helper.Alice)

	t.Log("alice account has been funded")
	balance, err := acctAlice.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != "10000.0000000" {
		t.Errorf("balance: %s, expected 10000.0000000", balance)
	}

	seqno, err := AccountSeqno(addressStr(t, helper.Alice))
	if err != nil {
		t.Fatal(err)
	}
	if seqno == 0 {
		t.Fatal("alice seqno: 0, expected non-zero")
	}

	details, err := acctAlice.Details()
	if err != nil {
		t.Fatal(err)
	}
	if len(details.Seqno) == 0 || details.Seqno == "0" {
		t.Errorf("details seqno should not be empty")
	}
	if details.SubentryCount != 0 {
		t.Errorf("subentries: %d, expected 0", details.SubentryCount)
	}
	if details.Available != "9999.0000000" {
		t.Errorf("available balance: %q, expected 9999.0000000", details.Available)
	}
	if len(details.Balances) != 1 {
		t.Fatalf("num balances: %d, expected 1", len(details.Balances))
	}
	if details.Balances[0].Balance != "10000.0000000" {
		t.Errorf("balance: %s, expected 10000.0000000", details.Balances[0].Balance)
	}
	if details.Balances[0].Type != "native" {
		t.Errorf("balance type: %s, expected native", details.Balances[0].Type)
	}

	t.Logf("alice (%s) sending 10 XLM to bob (%s)", helper.Alice.Address(), helper.Bob.Address())
	if _, _, err = SendXLM(seedStr(t, helper.Alice), addressStr(t, helper.Bob), "10.0"); err != nil {
		t.Fatal(err)
	}

	aliceExpected := "9989.9999800"
	balance, err = acctAlice.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != aliceExpected {
		t.Errorf("alice balance: %s, expected %s", balance, aliceExpected)
	}

	bobExpected := "10.0000000"
	acctBob := NewAccount(addressStr(t, helper.Bob))
	balance, err = acctBob.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != bobExpected {
		t.Errorf("bob balance: %s, expected %s", balance, bobExpected)
	}

	ledger, txid, err := SendXLM(seedStr(t, helper.Bob), addressStr(t, helper.Alice), "1.0")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("bob sent alice 1.0 XLM: %d, %s", ledger, txid)

	aliceTx, err := acctAlice.RecentTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceTx) != 3 {
		// this is unfortunate
		t.Logf("retrying alice recent transactions after 1s")
		time.Sleep(1 * time.Second)
		aliceTx, err = acctAlice.RecentTransactions()
		if err != nil {
			t.Fatal(err)
		}

		if len(aliceTx) != 3 {
			t.Errorf("# alice transactions: %d, expected 3", len(aliceTx))
		}
	}
	assertPayment(t, aliceTx[0], "1.0000000", helper.Bob.Address(), helper.Alice.Address())
	assertCreateAccount(t, aliceTx[1], "10.0000000", helper.Alice.Address(), helper.Bob.Address())
	assertCreateAccount(t, aliceTx[2], "10000.0000000", testclient.FriendbotAddress, helper.Alice.Address())

	bobTx, err := acctBob.RecentTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(bobTx) != 2 {
		t.Errorf("# bob transactions: %d, expected 2", len(bobTx))
	}
	assertPayment(t, bobTx[0], "1.0000000", helper.Bob.Address(), helper.Alice.Address())
	assertCreateAccount(t, bobTx[1], "10.0000000", helper.Alice.Address(), helper.Bob.Address())

	alicePayments, err := acctAlice.RecentPayments()
	if err != nil {
		t.Fatal(err)
	}
	if len(alicePayments) != 3 {
		t.Fatal("not 3")
	}

	bobPayments, err := acctBob.RecentPayments()
	if err != nil {
		t.Fatal(err)
	}
	if len(bobPayments) != 2 {
		t.Fatal("not 2")
	}

	active, err = IsMasterKeyActive(addressStr(t, helper.Alice))
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("not active")
	}

	payments, err := TxPayments(bobTx[0].Internal.ID)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Equal(t, bobTx[0].Internal.ID, payments[0].TransactionHash)
	require.Equal(t, helper.Bob.Address(), payments[0].SourceAccount)
	require.Equal(t, payments[0].SourceAccount, payments[0].From)
	require.Equal(t, helper.Alice.Address(), payments[0].To)
	require.Equal(t, "native", payments[0].AssetType)
	require.Equal(t, "1.0000000", payments[0].Amount)
	_, err = TxPayments(bobTx[0].Internal.ID[:5])
	require.Error(t, err)
	require.Contains(t, err.Error(), "error decoding transaction ID")

	txid2, err := CheckTxID(bobTx[0].Internal.ID)
	require.NoError(t, err)
	require.Equal(t, bobTx[0].Internal.ID, txid)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(bobTx[0].Internal.EnvelopeXdr, &tx)
	require.NoError(t, err)
	txid3, err := HashTx(tx.Tx)
	require.Equal(t, txid2, txid3)

	t.Logf("bob merges account into alice's account")
	sig, err := AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Alice), Client())
	require.NoError(t, err)
	_, _, err = Submit(sig.Signed)
	require.NoError(t, err)

	t.Log("bob's account has been merged away")
	_, err = acctBob.BalanceXLM()
	require.Error(t, err)
	require.Equal(t, ErrAccountNotFound, err)

	t.Log("alice got bob's balance")
	balance, err = acctAlice.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "9999.9999600", balance)

	t.Logf("alice merges into an unfunded account")
	sig, err = RelocateTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Charlie), false, Client())
	require.NoError(t, err)
	t.Logf("xxx - %v", spew.Sdump(sig))
	_, _, err = Submit(sig.Signed)
	require.NoError(t, err)

}
