package stellarnet

import (
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	horizonProtocol "github.com/stellar/go/protocols/horizon"
)

// A StellarClient is a client to the stellar network.  Internally,
// it uses a horizon client for talking to the network.
type StellarClient struct {
	url     string
	network build.Network
	client  *horizon.Client
}

// DefaultStellarClient might not be necessary, but it's here as a placeholder.
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

func newStellarClientWithHorizon(client *horizon.Client, network build.Network) *StellarClient {
	return &StellarClient{
		url:     client.URL,
		network: network,
		client:  client,
	}
}

// LoadAccount loads account information for accountID.
func (s *StellarClient) LoadAccount(accountID AddressStr) (*horizonProtocol.Account, error) {
	acct, err := s.client.LoadAccount(accountID.String())
	if err != nil {
		return nil, errMapAccount(err)
	}
	return &acct, nil
}

// GetJSON performs an HTTP Get on url and decodes the json response into dest.
func (s *StellarClient) GetJSON(url string, dest interface{}) error {
	err := getDecodeJSONStrict(s.url+url, s.client.HTTP.Get, dest)
	if err != nil {
		return errMap(err)
	}
	return nil
}
