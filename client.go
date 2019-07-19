package stellarnet

import (
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
)

// A StellarClient is a client to the stellar network.  Internally,
// it uses a horizon client for talking to the network.
type StellarClient struct {
	url     string
	network build.Network
	client  *horizon.Client
}

var DefaultStellarClient *StellarClient

func init() {
	DefaultStellarClient = NewStellarClient(horizon.DefaultPublicNetClient.URL, build.PublicNetwork)
}

// NewStellarClient creates a StellarClient with a horizon URL and network.
func NewStellarClient(url string, network build.Network) *StellarClient {
	return &StellarClient{
		url:     url,
		network: network,
		client:  MakeClient(url),
	}
}
