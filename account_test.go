package stellarnet

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/keybase/vcr"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
)

const friendbotAddress = "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR"

type Config struct {
	AliceSeed   string
	BobSeed     string
	CharlieSeed string
}

type Helper struct {
	config  *Config
	alice   *keypair.Full
	bob     *keypair.Full
	charlie *keypair.Full
}

func fullFromSeed(t *testing.T, seed string) *keypair.Full {
	kp, err := keypair.Parse(seed)
	if err != nil {
		t.Fatal(err)
	}
	full, ok := kp.(*keypair.Full)
	if !ok {
		t.Fatalf("keypair not full: %T", kp)
	}
	return full
}

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

func NewHelper(t *testing.T, c *Config) *Helper {
	return &Helper{
		config:  c,
		alice:   fullFromSeed(t, c.AliceSeed),
		bob:     fullFromSeed(t, c.BobSeed),
		charlie: fullFromSeed(t, c.CharlieSeed),
	}
}

func (h *Helper) SetState(t *testing.T, name string) {
	dir := filepath.Join("testdata", name)
	os.MkdirAll(dir, 0755)

	if *record {
		existing, err := filepath.Glob(filepath.Join(dir, "*.vcr"))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range existing {
			os.Remove(e)
		}
	}

	tvcr.SetDir(dir)
}

var live = flag.Bool("live", false, "use test server, do not update testdata")
var record = flag.Bool("record", false, "use test server, update testdata")

var tvcr *vcr.VCR

func newSeed(t *testing.T) string {
	kp, err := NewKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	return kp.Seed()
}

func setup(t *testing.T) *Helper {
	client, tvcr = testClient(t, *live, *record)
	network = build.TestNetwork

	var conf Config
	filename := filepath.Join("testdata", "config.json")

	if *live || *record {
		// make new key pairs since this is live or recording new live data
		conf.AliceSeed = newSeed(t)
		conf.BobSeed = newSeed(t)
		conf.CharlieSeed = newSeed(t)

		if *record {
			// recording, so save key pairs
			data, err := json.Marshal(conf)
			if err != nil {
				t.Fatal(err)
			}
			if err := ioutil.WriteFile(filename, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &conf); err != nil {
			t.Fatal(err)
		}
	}

	return NewHelper(t, &conf)
}

func getTestLumens(t *testing.T, kp keypair.KP) {
	t.Logf("getting test lumens from friendbot for %s", kp.Address())
	resp, err := http.Get("https://horizon-testnet.stellar.org/friendbot?addr=" + kp.Address())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
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
	helper := setup(t)

	helper.SetState(t, "scenario")

	t.Log("alice key pair not an account yet")
	acctAlice := NewAccount(addressStr(t, helper.alice))
	_, err := acctAlice.BalanceXLM()
	if err != ErrAccountNotFound {
		t.Fatalf("error: %q, expected %q (ErrAccountNotFound)", err, ErrAccountNotFound)
	}

	if *live || *record {
		getTestLumens(t, helper.alice)
	}

	t.Log("alice account has been funded")
	balance, err := acctAlice.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != "10000.0000000" {
		t.Errorf("balance: %s, expected 10000.0000000", balance)
	}

	t.Logf("alice (%s) sending 10 XLM to bob (%s)", helper.alice.Address(), helper.bob.Address())
	if _, err = acctAlice.SendXLM(seedStr(t, helper.alice), addressStr(t, helper.bob), "10.0"); err != nil {
		herr, ok := err.(*horizon.Error)
		if ok {
			t.Logf("horizon problem: %+v", herr.Problem)
			t.Logf("horizon extras: %s", string(herr.Problem.Extras["result_codes"]))
		}
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
	acctBob := NewAccount(addressStr(t, helper.bob))
	balance, err = acctBob.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != bobExpected {
		t.Errorf("bob balance: %s, expected %s", balance, bobExpected)
	}

	if _, err = acctBob.SendXLM(seedStr(t, helper.bob), addressStr(t, helper.alice), "1.0"); err != nil {
		t.Fatal(err)
	}

	aliceTx, err := acctAlice.RecentTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceTx) != 3 {
		t.Errorf("# alice transactions: %d, expected 3", len(aliceTx))
	}
	assertPayment(t, aliceTx[0], "1.0000000", helper.bob.Address(), helper.alice.Address())
	assertCreateAccount(t, aliceTx[1], "10.0000000", helper.alice.Address(), helper.bob.Address())
	assertCreateAccount(t, aliceTx[2], "10000.0000000", friendbotAddress, helper.alice.Address())

	bobTx, err := acctBob.RecentTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(bobTx) != 2 {
		t.Errorf("# bob transactions: %d, expected 2", len(bobTx))
	}
	assertPayment(t, bobTx[0], "1.0000000", helper.bob.Address(), helper.alice.Address())
	assertCreateAccount(t, bobTx[1], "10.0000000", helper.alice.Address(), helper.bob.Address())

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
}
