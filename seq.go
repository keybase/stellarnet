package stellarnet

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/xdr"
)

// SequenceProvider is the interface that other packages may implement to be
// used with the `AutoSequence` mutator.
type SequenceProvider interface {
	SequenceForAccount(aid string) (xdr.SequenceNumber, error)
}

type legacyClient struct {
	*horizonclient.Client
}

func (c *legacyClient) SequenceForAccount(aid string) (xdr.SequenceNumber, error) {

	acct, err := c.AccountDetail(horizonclient.AccountRequest{AccountID: aid})
	if err != nil {
		return xdr.SequenceNumber{}, err
	}

	return acct.GetSequenceNumber(), nil
}
