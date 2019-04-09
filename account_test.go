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
	sig, err := AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Alice), Client(), build.DefaultBaseFee)
	require.NoError(t, err)
	_, _, _, err = Submit(sig.Signed)
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
	_, _, _, err = Submit(sig.Signed)
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
	sig, err := AccountMergeTransaction(seedStr(t, helper.Bob), addressStr(t, helper.Alice), Client(), build.DefaultBaseFee)
	require.NoError(t, err)
	_, _, _, err = Submit(sig.Signed)
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
		_, _, _, err = Submit(tx.Signed)
		assertHorizonError(t, err, tc.txError)
	}

	tb := MakeTimeboundsWithMaxTime(time.Date(2030, time.November, 10, 23, 0, 0, 0, time.UTC))
	tx, err := CreateAccountXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
		"10.0", "", Client(), &tb, build.DefaultBaseFee)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = Submit(tx.Signed)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range badTbs {
		tx, err := PaymentXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
			"15.0", "", Client(), &tc.tb, build.DefaultBaseFee)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, err = Submit(tx.Signed)
		assertHorizonError(t, err, tc.txError)
	}

	tx, err = PaymentXLMTransaction(seedStr(t, helper.Alice), addressStr(t, helper.Bob),
		"15.0", "", Client(), &tb, build.DefaultBaseFee)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = Submit(tx.Signed)
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
		_, _, _, err = Submit(tx.Signed)
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
			ledger, txid, attempt, err := Submit(prepared[index].Signed)
			fmt.Printf("index: %d, ledger: %d, txid: %s, attempt: %d, err: %v\n", index, ledger, txid, attempt, err)
			if xerr, ok := err.(Error); ok {
				resultCodes, zerr := xerr.HorizonError.ResultCodes()
				if zerr == nil {
					fmt.Printf("index: %d, horizon error transaction code: %s\n", index, resultCodes.TransactionCode)
				} else {
					fmt.Printf("index: %d, zerr: %s (%s)\n", index, zerr, xerr.Details)
				}
			}
			results <- sres{Index: index, Attempt: attempt, Ledger: ledger, TxID: txid, Error: err}
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
	_, err = CreateTrustline(seedStr(t, helper.Alice), asset.AssetCode, issuer, 10000, 200)
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

	asset := findBestAsset(t, "USD")
	issuer, err := NewAddressStr(asset.AssetIssuer)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreateTrustline(seedStr(t, helper.Alice), asset.AssetCode, issuer, 10000, 200)
	if err != nil {
		t.Errorf("error creating trustline: %s, expected none", err)
	}

	// alice is going to do a path payment to herself to acquire some of this asset
	paths, err := acctAlice.FindPaymentPaths(acctAlice.address, asset.AssetCode, issuer, "10")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("paths: %+v\n", paths)
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
