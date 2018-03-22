package testclient

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

// FriendbotAddress is the last known public address for the test network friendbot.
const FriendbotAddress = "GAIH3ULLFQ4DGSECF2AR555KZ4KNDGEKN4AFI4SU2M7B43MGK3QJZNSR"

var live = flag.Bool("live", false, "use test server, do not update testdata")
var record = flag.Bool("record", false, "use test server, update testdata")

var tvcr *vcr.VCR

// Config contains the account seeds for the test users.
type Config struct {
	AliceSeed   string
	BobSeed     string
	CharlieSeed string
	RebeccaSeed string
}

// Helper makes managing the test users and state easier.
type Helper struct {
	Config  *Config
	Alice   *keypair.Full
	Bob     *keypair.Full
	Charlie *keypair.Full
	Rebecca *keypair.Full
}

// NewHelper creates a new Helper.
func NewHelper() *Helper {
	return &Helper{}
}

func (h *Helper) setConfig(t *testing.T, c *Config) {
	h.Config = c
	h.Alice = fullFromSeed(t, c.AliceSeed)
	h.Bob = fullFromSeed(t, c.BobSeed)
	h.Charlie = fullFromSeed(t, c.CharlieSeed)
	h.Rebecca = fullFromSeed(t, c.RebeccaSeed)
}

// SetState changes the directory where the http responses are stored.
// If record is on, it will clear out any existing files in the directory.
func (h *Helper) SetState(t *testing.T, name string) {
	dir := filepath.Join("testdata", name)
	os.MkdirAll(dir, 0755)

	conf := loadConfig(t, name)

	if *record {
		existing, err := filepath.Glob(filepath.Join(dir, "*.vcr"))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range existing {
			os.Remove(e)
		}
	}

	h.setConfig(t, conf)
	tvcr.SetDir(dir)
}

func testClient(t *testing.T, live, record bool) (*horizon.Client, *vcr.VCR) {
	v := vcr.New("testdata")
	if record {
		t.Logf("recording http requests")
		v.Record()
	} else if live {
		t.Logf("live http requests")
		v.Live()
	} else {
		t.Logf("playing recorded http requests")
	}

	return &horizon.Client{
		URL:  "https://horizon-testnet.stellar.org",
		HTTP: v,
	}, v
}

func loadConfig(t *testing.T, subdir string) *Config {
	var conf Config
	filename := filepath.Join("testdata", subdir, "config.json")

	if *live || *record {
		// make new key pairs since this is live or recording new live data
		conf.AliceSeed = newSeed(t)
		conf.BobSeed = newSeed(t)
		conf.CharlieSeed = newSeed(t)
		conf.RebeccaSeed = newSeed(t)

		if *record {
			// recording, so save key pairs
			data, err := json.Marshal(conf)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("saving config file %s", filename)
			if err := ioutil.WriteFile(filename, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		t.Logf("loading config file %s", filename)
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &conf); err != nil {
			t.Fatal(err)
		}
	}

	return &conf
}

// Setup is the primary entry point for testclient.  It creates a Helper
// and the horizon client.
func Setup(t *testing.T) (*Helper, *horizon.Client, build.Network) {
	var client *horizon.Client
	client, tvcr = testClient(t, *live, *record)

	h := NewHelper()
	conf := loadConfig(t, "")
	h.setConfig(t, conf)

	return h, client, build.TestNetwork
}

// GetTestLumens will use the friendbot to get some lumens into kp's account.
// If not record or live, it is a no-op.
func GetTestLumens(t *testing.T, kp keypair.KP) {
	if *record || *live {
		t.Logf("getting test lumens from friendbot for %s", kp.Address())
		resp, err := http.Get("https://friendbot.stellar.org/?addr=" + kp.Address())
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("friendbot response: %+v", resp)
		t.Logf("friendbot body: %s", body)
	}
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

func newSeed(t *testing.T) string {
	kp, err := keypair.Random()
	if err != nil {
		t.Fatal(err)
	}
	return kp.Seed()
}
