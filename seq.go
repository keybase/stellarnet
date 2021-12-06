package stellarnet

import (
	"github.com/stellar/go/clients/horizonclient"
)

// SequenceProvider is the interface that other packages may implement to be
// used with the `AutoSequence` mutator.
type SequenceProvider interface {
	SequenceForAccount(aid string) (int64, error)
}

// LegacyClient is a legacy horizon client.
type LegacyClient struct {
	*horizonclient.Client
}

// SequenceForAccount provides the sequence number for a given account id.
func (c *LegacyClient) SequenceForAccount(aid string) (int64, error) {

	acct, err := c.AccountDetail(horizonclient.AccountRequest{AccountID: aid})
	if err != nil {
		return 0, err
	}

	return acct.GetSequenceNumber()
}
