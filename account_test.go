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

func TestScenario(t *testing.T) {
	helper := setup(t)

	helper.SetState(t, "scenario")

	t.Log("alice key pair not an account yet")
	acctAlice := NewAccount(helper.alice.Address())
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
	if _, err := acctAlice.Send(helper.alice.Seed(), helper.bob.Address(), "10.0"); err != nil {
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
	acctBob := NewAccount(helper.bob.Address())
	balance, err = acctBob.BalanceXLM()
	if err != nil {
		t.Fatal(err)
	}
	if balance != bobExpected {
		t.Errorf("bob balance: %s, expected %s", balance, bobExpected)
	}

	if _, err := acctBob.Send(helper.bob.Seed(), helper.alice.Address(), "1.0"); err != nil {
		t.Fatal(err)
	}

	aliceTx, err := acctAlice.RecentTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceTx) != 3 {
		t.Errorf("# alice transactions: %d, expected 3", len(aliceTx))
	}
	if aliceTx[0].Operations[0].Type != "payment" {
		t.Errorf("tx 0 type: %s, expected payment", aliceTx[0].Operations[0].Type)
	}
	if aliceTx[1].Operations[0].Type != "create_account" {
		t.Errorf("tx 1 type: %s, expected create_account", aliceTx[1].Operations[0].Type)
	}
	if aliceTx[2].Operations[0].Type != "create_account" {
		t.Errorf("tx 2 type: %s, expected create_account", aliceTx[2].Operations[0].Type)
	}

	bobTx, err := acctBob.RecentTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(bobTx) != 2 {
		t.Errorf("# bob transactions: %d, expected 2", len(bobTx))
	}
	if bobTx[0].Operations[0].Type != "payment" {
		t.Errorf("tx 0 type: %s, expected payment", bobTx[0].Operations[0].Type)
	}
	if bobTx[1].Operations[0].Type != "create_account" {
		t.Errorf("tx 1 type: %s, expected create_account", bobTx[1].Operations[0].Type)
	}
}
