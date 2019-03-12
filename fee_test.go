package stellarnet

import (
	"testing"

	"github.com/keybase/stellarnet/testclient"
)

// TestFeeStats makes sure that the horizon fetcher
// works, is decodable, and returns non-zero values for
// some important fields.
func TestFeeStats(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "fee_stats")

	stats, err := FeeStats(&HorizonFeeStatFetcher{})
	if err != nil {
		t.Fatal(err)
	}

	if stats.LastLedger == 0 {
		t.Error("last ledger: 0, expected non-zero")
	}
	if stats.LedgerCapacityUsage == 0.0 {
		t.Error("ledger capacity usage: 0.0, expected non-zero")
	}
	if stats.MinAcceptedFee == 0 {
		t.Error("min accepted fee: 0, expected non-zero")
	}
	if stats.P95AcceptedFee == 0 {
		t.Error("p95 accepted fee: 0, expected non-zero")
	}
}
