package stellarnet

import (
	"testing"
	"vcr"

	"github.com/stellar/go/clients/horizon"
)

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
