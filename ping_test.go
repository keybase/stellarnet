package stellarnet

import (
	"strings"
	"testing"

	"github.com/keybase/stellarnet/testclient"
)

func TestPing(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClient(client, network)
	helper.SetState(t, "ping")

	ping, err := Ping()
	if err != nil {
		t.Fatal(err)
	}
	expected := "horizon ver: snapshot"
	if !strings.HasPrefix(ping, expected) {
		t.Errorf("ping: %q, expected to start with %q", ping, expected)
	}
}
