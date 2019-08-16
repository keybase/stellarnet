package stellarnet

import (
	"fmt"
	"sync"
	"testing"
	"time"

	perrors "github.com/pkg/errors"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	horizonProtocol "github.com/stellar/go/protocols/horizon"
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
	require.Equal(t, "payment", op.Type)
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
	require.Equal(t, "create_account", op.Type)
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

// getHorizonError tries to find a horizon.Error buried in a stellarnet error
func getHorizonError(err error) *horizon.Error {
	err = perrors.Cause(err)
	if zerr, ok := err.(Error); ok {
		if zerr.HorizonError != nil {
			err = zerr.HorizonError
		}
	}
	err = perrors.Cause(err)
	if herr, ok := err.(*horizon.Error); ok {
		return herr
	}
	return nil
}

func assertHorizonError(t *testing.T, err error, transactionCode string) {
	if err == nil {
		t.Fatalf("Expected %q horizon error but got nil", transactionCode)
	}
	if herr := getHorizonError(err); herr != nil {
		resultCodes, xerr := herr.ResultCodes()
		if xerr != nil {
			t.Fatalf("Failed when inspecting error, ResultCodes() -> %s", xerr)
		}
		t.Logf("assertHorizonError: horizon error: %q with codes: %+v", herr, resultCodes)
		if resultCodes.TransactionCode != transactionCode {
			t.Fatalf("Unexpected transaction code %s != %s", resultCodes.TransactionCode, transactionCode)
		}
	} else {
		t.Fatalf("Error was not a horizon error, but: %s", err)
	}
}

func TestScenario(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "scenario")

	t.Log("alice key pair not an account yet")
	acctAlice := NewAccount(addressStr(t, helper.Alice))
	_, err := acctAlice.BalanceXLM()
	if err != ErrSourceAccountNotFound {
		t.Fatalf("error: %q, expected %q (ErrSourceAccountNotFound)", err, ErrSourceAccountNotFound)
	}

	_, err = AccountSeqno(addressStr(t, helper.Alice))
	if err != ErrSourceAccountNotFound {
		t.Fatalf("error: %q, expected %q (ErrSourceAccountNotFound)", err, ErrSourceAccountNotFound)
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
	if _, _, _, err = SendXLM(seedStr(t, helper.Alice), addressStr(t, helper.Bob), "10.0", "" /* empty memo */); err != nil {
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

	ledger, txid, attempt, err := SendXLM(seedStr(t, helper.Bob), addressStr(t, helper.Alice), "1.0", "a memo")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("bob sent alice 1.0 XLM: %d, %s (attempt: %d)", ledger, txid, attempt)

	aliceTx, err := acctAlice.RecentTransactionsAndOps()
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceTx) != 3 {
		// this is unfortunate
		t.Logf("retrying alice recent transactions after 1s")
		time.Sleep(1 * time.Second)
		aliceTx, err = acctAlice.RecentTransactionsAndOps()
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

	bobTx, err := acctBob.RecentTransactionsAndOps()
	if err != nil {
		t.Fatal(err)
	}
	if len(bobTx) != 2 {
		t.Errorf("# bob transactions: %d, expected 2", len(bobTx))
	}
	assertPayment(t, bobTx[0], "1.0000000", helper.Bob.Address(), helper.Alice.Address())
	assertCreateAccount(t, bobTx[1], "10.0000000", helper.Alice.Address(), helper.Bob.Address())

	alicePayments, err := acctAlice.RecentPayments("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(alicePayments) != 3 {
		t.Fatal("not 3")
	}

	bobPayments, err := acctBob.RecentPayments("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(bobPayments) != 2 {
		t.Fatal("not 2")
	}

	// try with a cursor
	nextPayments, err := acctAlice.RecentPayments(alicePayments[0].PagingToken, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(nextPayments) != 2 {
		t.Fatal("not 2")
	}
	if nextPayments[0].ID != alicePayments[1].ID {
		t.Fatalf("id: %q, expected: %q", nextPayments[0].ID, alicePayments[1].ID)
	}
	if nextPayments[1].ID != alicePayments[2].ID {
		t.Fatalf("id: %q, expected: %q", nextPayments[1].ID, alicePayments[2].ID)
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

	txdetails, err := TxDetails(bobTx[0].Internal.ID)
	require.NoError(t, err)
	require.Equal(t, bobTx[0].Internal.ID, txdetails.ID)
	require.Equal(t, "a memo", txdetails.Memo)
	require.Equal(t, "text", txdetails.MemoType)

	_, err = TxPayments(bobTx[0].Internal.ID[:5])
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid transaction ID")

	txid2, err := CheckTxID(bobTx[0].Internal.ID)
	require.NoError(t, err)
	require.Equal(t, bobTx[0].Internal.ID, txid)

	var tx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(bobTx[0].Internal.EnvelopeXdr, &tx)
	require.NoError(t, err)
	txid3, err := HashTx(tx.Tx)
	require.NoError(t, err)
	require.Equal(t, txid2, txid3)

	t.Logf("bob merges account into alice's account")
	sig, err := AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Alice), Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.NoError(t, err)
	_, err = Submit(sig.Signed)
	require.NoError(t, err)

	t.Log("bob's account has been merged away")
	_, err = acctBob.BalanceXLM()
	require.Error(t, err)
	require.Equal(t, ErrSourceAccountNotFound, err)

	t.Log("alice got bob's balance")
	balance, err = acctAlice.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "9999.9999600", balance)

	t.Logf("alice merges into an unfunded account")
	var nines uint64 = 999
	sig, err = RelocateTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Charlie), false, &nines, Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.NoError(t, err)
	_, err = Submit(sig.Signed)
	require.NoError(t, err)

	t.Logf("charlie merges into a funded account")
	lip := helper.Keypair(t, "Lip")
	testclient.GetTestLumens(t, lip)
	sig, err = RelocateTransaction(seedStr(t, helper.Charlie), addressStr(t, lip), true, &nines, Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.NoError(t, err)
}

func TestAccountMergeAmount(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "TestAccountMergeAmount")

	t.Logf("gift -> alice")
	testclient.GetTestLumens(t, helper.Alice)

	t.Logf("alice -> bob")
	transferAmount := "123.456"
	transferAmountMinusMergeFee := "123.4559900"
	_, _, _, err := SendXLM(seedStr(t, helper.Alice), addressStr(t, helper.Bob), transferAmount, "")
	require.NoError(t, err)

	t.Logf("bob merges back to alice")
	sig, err := AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Alice), Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.NoError(t, err)
	_, err = Submit(sig.Signed)
	require.NoError(t, err)

	if !testclient.IsPlayback() {
		t.Logf("wait for the merge transaction to propagate from horizon to itself")
		time.Sleep(1 * time.Second)
	}

	t.Logf("read history of alice")
	acctAlice := NewAccount(addressStr(t, helper.Alice))
	payments, err := acctAlice.RecentPayments("", 1)
	require.NoError(t, err)
	require.Len(t, payments, 1)
	require.Equal(t, "account_merge", payments[0].Type)

	amount, err := AccountMergeAmount(payments[0].ID)
	require.NoError(t, err)
	require.Equal(t, transferAmountMinusMergeFee, amount)

	t.Logf("read history of bob")
	acctBob := NewAccount(addressStr(t, helper.Bob))
	payments, err = acctBob.RecentPayments("", 50)
	require.NoError(t, err)
	require.Len(t, payments, 2)
	require.Equal(t, "account_merge", payments[0].Type)
	require.Equal(t, "create_account", payments[1].Type)

	amount, err = AccountMergeAmount(payments[0].ID)
	require.NoError(t, err)
	require.Equal(t, transferAmountMinusMergeFee, amount)
}

func TestSetInflationDestination(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "inflation")

	t.Log("alice key pair not an account yet")
	acctAlice := NewAccount(addressStr(t, helper.Alice))
	_, err := acctAlice.BalanceXLM()
	require.Error(t, err)
	require.Equal(t, ErrSourceAccountNotFound, err)

	_, err = AccountSeqno(addressStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrSourceAccountNotFound, err)

	active, err := IsMasterKeyActive(addressStr(t, helper.Alice))
	require.NoError(t, err)
	require.True(t, active)

	_, _, _, err = setInflationDestination(seedStr(t, helper.Alice), addressStr(t, helper.Alice))
	require.Error(t, err)
	require.Equal(t, ErrResourceNotFound, err)

	testclient.GetTestLumens(t, helper.Alice)

	t.Log("alice account has been funded")

	details, err := acctAlice.Details()
	require.NoError(t, err)
	require.Equal(t, "", details.InflationDestination)

	_, _, _, err = setInflationDestination(seedStr(t, helper.Alice), addressStr(t, helper.Alice))
	require.NoError(t, err)

	balance, err := acctAlice.BalanceXLM()
	require.NoError(t, err)
	require.Equal(t, "9999.9999900", balance)

	details, err = acctAlice.Details()
	require.NoError(t, err)
	require.Equal(t, addressStr(t, helper.Alice).String(), details.InflationDestination)
}

func TestTimeBounds(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "timebounds")

	testclient.GetTestLumens(t, helper.Alice)

	type TimeBoundTest struct {
		tb      build.Timebounds
		txError string
	}
	badTbs := []TimeBoundTest{
		{MakeTimeboundsWithMaxTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)), "tx_too_late"},
		{MakeTimeboundsFromTime(
			time.Date(2030, time.November, 10, 23, 0, 0, 0, time.UTC),
			time.Date(2030, time.December, 10, 23, 0, 0, 0, time.UTC)), "tx_too_early"},
	}

	for _, tc := range badTbs {
		tx, err := CreateAccountXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
			"10.0", "", Client(), &tc.tb, build.DefaultBaseFee)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Submit(tx.Signed)
		assertHorizonError(t, err, tc.txError)
	}

	tb := MakeTimeboundsWithMaxTime(time.Date(2030, time.November, 10, 23, 0, 0, 0, time.UTC))
	tx, err := CreateAccountXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
		"10.0", "", Client(), &tb, build.DefaultBaseFee)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Submit(tx.Signed)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range badTbs {
		tx, err := PaymentXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
			"15.0", "", Client(), &tc.tb, build.DefaultBaseFee)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Submit(tx.Signed)
		assertHorizonError(t, err, tc.txError)
	}

	tx, err = PaymentXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
		"15.0", "", Client(), &tb, build.DefaultBaseFee)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Submit(tx.Signed)
	if err != nil {
		t.Fatal(err)
	}

	acctBob := NewAccount(addressStr(t, helper.Bob))
	balance, err := acctBob.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != "25.0000000" {
		t.Errorf("balance: %s, expected 25.0000000", balance)
	}

	for _, tc := range badTbs {
		tx, err := RelocateTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Alice),
			true, nil, Client(), &tc.tb, build.DefaultBaseFee)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Submit(tx.Signed)
		assertHorizonError(t, err, tc.txError)
	}
}

type sres struct {
	Index   int
	Attempt int
	Ledger  int32
	TxID    string
	Error   error
}

// TestConcurrentSubmit gets rate limited very quickly on testnet (rate limit is 100 operations
// per hour?).  It is skipped, but can be useful locally.
func TestConcurrentSubmit(t *testing.T) {
	t.Skip("this only works with -live")
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "concurrent")

	testclient.GetTestLumens(t, helper.Alice)
	testclient.GetTestLumens(t, helper.Bob)

	aliceSeqno, err := AccountSeqno(addressStr(t, helper.Alice))
	if err != nil {
		t.Fatal(err)
	}

	sprov := &testSeqnoProv{seqno: aliceSeqno}

	n := 20
	prepared := make([]SignResult, n)
	for i := 0; i < n; i++ {
		sig, err := PaymentXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob), fmt.Sprintf("%d", i+1), "", sprov, nil, build.DefaultBaseFee)
		if err != nil {
			t.Fatal(err)
		}
		prepared[i] = sig
	}

	results := make(chan sres, n)
	for i := 0; i < n; i++ {
		go func(index int) {
			res, err := Submit(prepared[index].Signed)
			fmt.Printf("index: %d, result: %+v, err: %v\n", index, res, err)
			if xerr, ok := err.(Error); ok {
				resultCodes, zerr := xerr.HorizonError.ResultCodes()
				if zerr == nil {
					fmt.Printf("index: %d, horizon error transaction code: %s\n", index, resultCodes.TransactionCode)
				} else {
					fmt.Printf("index: %d, zerr: %s (%s)\n", index, zerr, xerr.Details)
				}
			}
			results <- sres{Index: index, Attempt: res.Attempt, Ledger: res.Ledger, TxID: res.TxID, Error: err}
		}(i)
	}

	for i := 0; i < n; i++ {
		r := <-results
		if r.Error != nil {
			t.Errorf("payment %d failed (attempt = %d), err: %s", r.Index, r.Attempt, r.Error)
		} else {
			t.Logf("payment %d success (attempt = %d)", r.Index, r.Attempt)
			fmt.Printf("payment %d success (attempt = %d) ledger: %d\ttx id: %s\n", r.Index, r.Attempt, r.Ledger, r.TxID)
		}
	}
}

func TestTrustlines(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "trustlines")

	acctAlice := NewAccount(addressStr(t, helper.Alice))
	_, err := acctAlice.Trustlines()
	if err == nil {
		t.Errorf("expected an error getting trustlines on unestablished account")
	}

	testclient.GetTestLumens(t, helper.Alice)

	tlines, err := acctAlice.Trustlines()
	if err != nil {
		t.Errorf("Trustlines error: %s, expected no error getting trustlines on established account", err)
	}

	if len(tlines) != 1 {
		t.Errorf("num trustlines: %d, expected 1", len(tlines))
	}
	if tlines[0].Type != "native" {
		t.Errorf("trustline type: %q, expected native", tlines[0].Type)
	}

	asset := findBestAsset(t, "USD")
	issuer, err := NewAddressStr(asset.AssetIssuer)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreateTrustline(seedStr(t, helper.Alice), asset.AssetCode, issuer, "10000", 200)
	if err != nil {
		t.Errorf("error creating trustline: %s, expected none", err)
	}

	tlines, err = acctAlice.Trustlines()
	if err != nil {
		t.Errorf("Trustlines error: %s, expected no error getting trustlines on established account", err)
	}

	if len(tlines) != 2 {
		t.Errorf("num trustlines: %d, expected 2", len(tlines))
	}

	_, err = DeleteTrustline(seedStr(t, helper.Alice), asset.AssetCode, issuer, 200)
	if err != nil {
		t.Fatal(err)
	}
	tlines, err = acctAlice.Trustlines()
	if err != nil {
		t.Errorf("Trustlines error: %s, expected no error getting trustlines on established account", err)
	}

	if len(tlines) != 1 {
		t.Errorf("num trustlines: %d, expected 1", len(tlines))
	}
}

func TestPathPayments(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "pathpayments")

	acctAlice := NewAccount(addressStr(t, helper.Alice))
	testclient.GetTestLumens(t, helper.Alice)

	acctBob := NewAccount(addressStr(t, helper.Bob))
	testclient.GetTestLumens(t, helper.Bob)

	// alice is going to make a new asset
	t.Logf("issuer address: %s", helper.Issuer.Address())
	t.Logf("distributor address: %s", helper.Distributor.Address())
	assetCode := helper.Config.AssetCode
	issuer, distributor, err := CreateCustomAssetWithKPs(seedStr(t, helper.Alice), helper.Issuer, helper.Distributor, assetCode, "10000", "keybase.io/blueasset", "2.3", 200)
	if err != nil {
		t.Logf("CreateCustomAsset error type: %T", err)
		if serr, ok := err.(Error); ok {
			t.Logf(serr.Verbose())
			if serr.HorizonError != nil {
				t.Logf("horizon error: %+v", serr.HorizonError)
				t.Logf("horizon error problem: %+v", serr.HorizonError.Problem)
				t.Logf("horizon error extras %s", serr.HorizonError.Problem.Extras["result_codes"])
			}
		}
		t.Fatal(err)
	}
	issuerAddr, err := issuer.Address()
	if err != nil {
		t.Fatal(err)
	}
	_ = distributor

	t.Logf("returned issuer address: %s", issuerAddr)
	if issuerAddr.String() != helper.Issuer.Address() {
		t.Errorf("issuerAddr returned by CreateCustomAsset did not match address in helper.Issuer (%q != %q)", issuerAddr, helper.Issuer.Address())
	}

	// check that the asset is available
	assets, err := AssetsWithCode(assetCode)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d assets with code %q", len(assets), assetCode)
	match := false
	for _, a := range assets {
		t.Logf("asset: %+v", a)
		if a.AssetIssuer == issuerAddr.String() {
			match = true
		}
	}
	if !match {
		t.Fatalf("no asset with code %q and issuer %q found", assetCode, issuerAddr)
	}

	_, err = CreateTrustline(seedStr(t, helper.Alice), assetCode, issuerAddr, "10000", 200)
	if err != nil {
		t.Errorf("error creating trustline: %s, expected none", err)
	}

	// bob is going to do a path payment to alice with this asset as the destination
	// asset

	// first bob finds the paths available
	paths, err := acctBob.FindPaymentPaths(acctAlice.address, assetCode, issuerAddr, "10")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no paths available from bob to alice for %s/%s", assetCode, issuerAddr)
	}

	// select the path to use
	path := paths[0]

	// calculate the max send amount
	sendAmountMax, err := PathPaymentMaxValue(path.SourceAmount)
	if err != nil {
		t.Fatal(err)
	}

	// then bob makes the path payment
	_, txID, _, err := pathPayment(seedStr(t, helper.Bob), acctAlice.address, path.SourceAsset(), sendAmountMax, path.DestinationAsset(), path.DestinationAmount, PathAssetSliceToAssetBase(path.Path), "pub memo path pay")
	if err != nil {
		t.Fatal(err)
	}

	aliceTx, _, err := acctAlice.Transactions("", 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(aliceTx) != 5 {
		t.Fatalf("alice tx count: %d, expected 5", len(aliceTx))
	}

	match = false
	var pathTx horizonProtocol.Transaction
	for _, tx := range aliceTx {
		if tx.ID == txID {
			match = true
			pathTx = tx
			break
		}
	}
	if !match {
		t.Fatal("path payment to alice not in recent txs")
	}

	var unpackedTx xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(pathTx.EnvelopeXdr, &unpackedTx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("envelope: %+v", unpackedTx)

	if len(unpackedTx.Tx.Operations) != 1 {
		t.Fatalf("operations: %d, expected 1", len(unpackedTx.Tx.Operations))
	}
	op := unpackedTx.Tx.Operations[0]
	if op.Body.Type != xdr.OperationTypePathPayment {
		t.Fatalf("operation type: %v, expected path payment (%v)", op.Body.Type, xdr.OperationTypePathPayment)
	}
	t.Logf("path payment op: %+v", op.Body.PathPaymentOp)
	pathOp := op.Body.PathPaymentOp
	if pathOp.Destination.Address() != acctAlice.address.String() {
		t.Errorf("destination: %s, expected alice %s", pathOp.Destination.Address(), acctAlice.address)
	}
}

type testSeqnoProv struct {
	seqno uint64
	sync.Mutex
}

func (x *testSeqnoProv) SequenceForAccount(s string) (xdr.SequenceNumber, error) {
	x.Lock()
	defer x.Unlock()
	result := xdr.SequenceNumber(x.seqno)
	x.seqno++
	return result, nil
}

func findBestAsset(t *testing.T, code string) AssetSummary {
	assets, err := AssetsWithCode(code)
	if err != nil {
		t.Fatal(err)
	}
	maxAccounts := -1
	var best *AssetSummary
	for _, a := range assets {
		if a.NumAccounts > maxAccounts {
			maxAccounts = a.NumAccounts
			a := a
			best = &a
		}
	}
	if best == nil {
		t.Fatalf("no suitable assets found for %q", code)
	}

	return *best
}

func TestAccountMergeFull(t *testing.T) {
	// set up alice with an account that has native and non-native assets
	// merge it into bob (who must also have trustlines to the same assets)
	// verify that bob has the assets, and alice no longer exists
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "accountmerge")

	acctAlice := NewAccount(addressStr(t, helper.Alice))
	acctBob := NewAccount(addressStr(t, helper.Bob))
	// acctCharlie := NewAccount(addressStr(t, helper.Charlie))
	var err error

	// create an asset
	source := helper.Keypair(t, "Source")
	acctSource := NewAccount(addressStr(t, source))
	testclient.GetTestLumens(t, source)
	sourceSeed := SeedStr(source.Seed())
	assetCode := "SOMN"
	issuerPair := helper.Keypair(t, "issuer")
	issuerAddr, err := NewAddressStr(issuerPair.Address())
	require.NoError(t, err)
	distPair := helper.Keypair(t, "dist")
	_, _, err = CreateCustomAssetWithKPs(sourceSeed, issuerPair, distPair, assetCode, "10000", "keybase.io/blueasset", "2.3", 200)
	require.NoError(t, err)

	// send 10 lumens from source to alice and bob to create their accounts
	_, _, _, err = SendXLM(sourceSeed, acctAlice.address, "10.0", "" /* empty memo */)
	require.NoError(t, err)
	_, _, _, err = SendXLM(sourceSeed, acctBob.address, "10.0", "" /* empty memo */)
	require.NoError(t, err)
	// create trustlines for our new asset to Alice and Bob
	_, err = CreateTrustline(seedStr(t, helper.Alice), assetCode, issuerAddr, "10000", 200)
	require.NoError(t, err)
	_, err = CreateTrustline(seedStr(t, helper.Bob), assetCode, issuerAddr, "10000", 200)
	require.NoError(t, err)
	// send a path payment from the source to Alice so alice can also have some of the custom asset
	paths, err := acctSource.FindPaymentPaths(acctAlice.address, assetCode, issuerAddr, "10")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(paths), 1)
	path := paths[0]
	sendAmountMax, err := PathPaymentMaxValue(path.SourceAmount)
	require.NoError(t, err)
	_, _, _, err = pathPayment(seedStr(t, source), acctAlice.address, path.SourceAsset(), sendAmountMax, path.DestinationAsset(), path.DestinationAmount, PathAssetSliceToAssetBase(path.Path), "pub memo path pay")
	require.NoError(t, err)
	// verify the balances are what we expect before attempting the actual merge transaction
	balances, err := acctAlice.Balances()
	require.NoError(t, err)
	require.Equal(t, 2, len(balances))
	for _, balance := range balances {
		if balance.Asset.Type == "native" {
			require.Regexp(t, `^9.9999`, balance.Balance)
		} else if balance.Asset.Code == assetCode {
			require.Equal(t, "10.0000000", balance.Balance)
		} else {
			t.Fatal("unexpected asset in these balances")
		}
	}

	// do the merge
	sig, err := AccountMergeTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob), Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.NoError(t, err)
	_, err = Submit(sig.Signed)
	require.NoError(t, err)

	// bob now has all the assets
	balances, err = acctBob.Balances()
	require.NoError(t, err)
	require.Equal(t, 2, len(balances))
	for _, balance := range balances {
		if balance.Asset.Type == "native" {
			require.Equal(t, "19.9999500", balance.Balance)
		} else if balance.Asset.Code == assetCode {
			require.Equal(t, "10.0000000", balance.Balance)
		} else {
			t.Fatal("unexpected asset in these balances")
		}
	}

	// and alice's account is gone
	_, err = acctAlice.BalanceXLM()
	require.Error(t, err)
	require.Equal(t, ErrSourceAccountNotFound, err)
	t.Logf("successfully merged an account with an XLM and non-native asset balance")

	// if we try to merge Bob's account into another account that doesn't support
	// the custom asset, it will fail
	sig, err = AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Charlie), Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot merge")

	// if we eliminate the balance on that asset (just send it back to the issuer) and then try again, it should then work
	paths, err = acctBob.FindPaymentPaths(issuerAddr, assetCode, issuerAddr, "10")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(paths), 1)
	path = paths[0]
	sendAmountMax, err = PathPaymentMaxValue(path.SourceAmount)
	require.NoError(t, err)
	_, _, _, err = pathPayment(seedStr(t, helper.Bob), issuerAddr, path.SourceAsset(), sendAmountMax, path.DestinationAsset(), path.DestinationAmount, PathAssetSliceToAssetBase(path.Path), "pub memo path pay")
	require.NoError(t, err)
	// attempt the merge again from bob into charlie
	sig, err = AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Charlie), Client(), nil /* timeBounds */, build.DefaultBaseFee)
	require.NoError(t, err)
	_, err = Submit(sig.Signed)
	require.NoError(t, err)
}
