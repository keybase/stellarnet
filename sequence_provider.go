package stellarnet

import (
	"strconv"

	"github.com/pkg/errors"
	horizon "github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/xdr"
)

// SequenceProvider contains SequenceForAccount to look up latest sequence
// number of a Stellar account.
type SequenceProvider interface {
	// Look up a sequence by address
	SequenceForAccount(aid string) (xdr.SequenceNumber, error)
}

// ClientSequenceProvider implements SequenceProvider using horizon.Client.
type ClientSequenceProvider struct {
	Client *horizon.Client
}

// SequenceForAccount implements build.SequenceProvider
// Deprecated: use clients/horizonclient instead
func (c ClientSequenceProvider) SequenceForAccount(
	accountID string,
) (xdr.SequenceNumber, error) {

	a, err := c.Client.AccountDetail(horizon.AccountRequest{AccountID: accountID})
	if err != nil {
		return 0, errors.Wrap(err, "load account failed")
	}

	seq, err := strconv.ParseUint(a.Sequence, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "parse sequence failed")
	}

	return xdr.SequenceNumber(seq), nil
}
