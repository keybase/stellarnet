package stellarnetwork

import "github.com/stellar/go/network"

// StellarNetwork establishes the stellar network that a transaction should apply to.
// This modifier influences how a transaction is hashed for the purposes of signature generation.
type StellarNetwork struct {
	Passphrase string
}

// ID returns the network ID derived from this struct's Passphrase
func (n *StellarNetwork) ID() [32]byte {
	return network.ID(n.Passphrase)
}

var (
	// PublicNetwork is a mutator that configures the transaction for submission
	// to the main public stellar network.
	PublicNetwork = StellarNetwork{network.PublicNetworkPassphrase}

	// TestNetwork is a mutator that configures the transaction for submission
	// to the test stellar network (often called testnet).
	TestNetwork = StellarNetwork{network.TestNetworkPassphrase}

	// DefaultNetwork is a mutator that configures the
	// transaction for submission to the default stellar
	// network.  Integrators may change this value to
	// another `Network` mutator if they would like to
	// effect the default in a process-global manner.
	// Replace or set your own custom passphrase on this
	// var to set the default network for the process.
	DefaultNetwork = StellarNetwork{}
)
