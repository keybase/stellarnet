package stellarnet

import (
	"testing"
	"time"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stellar/go/keypair"
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
	SetClient(client, network)
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

	t.Logf("alice (%s) sending 10 XLM to bob (%s)", helper.Alice.Address(), helper.Bob.Address())
	if _, _, err = acctAlice.SendXLM(seedStr(t, helper.Alice), addressStr(t, helper.Bob), "10.0"); err != nil {
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

	ledger, txid, err := acctBob.SendXLM(seedStr(t, helper.Bob), addressStr(t, helper.Alice), "1.0")
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
}
