package resourceadapter

import (
	"context"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
)

// Populate fills out the fields of the signer, using one of an account's
// secondary signers.
func PopulateSigner(ctx context.Context, dest *protocol.Signer, row core.Signer) {
	dest.Weight = row.Weight
	dest.Key = row.Publickey
	dest.Type = protocol.MustKeyTypeFromAddress(dest.Key)
}

// PopulateMaster fills out the fields of the signer, using a stellar account to
// provide the data.
func PopulateMasterSigner(dest *protocol.Signer, row core.Account) {
	dest.Weight = int32(row.Thresholds[0])
	dest.Key = row.Accountid
	dest.Type = protocol.MustKeyTypeFromAddress(dest.Key)
}
