package resourceadapter

import (
	"context"
	"fmt"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/support/render/hal"
)

// PopulateAccountSigner fills out the resource's fields
func PopulateAccountSigner(
	ctx context.Context,
	dest *protocol.AccountSigner,
	has history.AccountSigner,
) {
	dest.ID = has.Account
	dest.AccountID = has.Account
	dest.PT = has.Account
	dest.Signer = protocol.Signer{
		Weight: has.Weight,
		Key:    has.Signer,
		Type:   protocol.MustKeyTypeFromAddress(has.Signer),
	}

	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	account := fmt.Sprintf("/accounts/%s", has.Account)
	dest.Links.Account = lb.Link(account)
}
