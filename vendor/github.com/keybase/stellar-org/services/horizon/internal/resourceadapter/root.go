package resourceadapter

import (
	"context"
	"net/url"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/support/render/hal"
)

// Populate fills in the details
func PopulateRoot(
	ctx context.Context,
	dest *horizon.Root,
	ledgerState ledger.State,
	hVersion, cVersion string,
	passphrase string,
	currentProtocolVersion int32,
	coreSupportedProtocolVersion int32,
	friendBotURL *url.URL,
) {
	dest.HorizonSequence = ledgerState.HistoryLatest
	dest.HistoryElderSequence = ledgerState.HistoryElder
	dest.CoreSequence = ledgerState.CoreLatest
	dest.HorizonVersion = hVersion
	dest.StellarCoreVersion = cVersion
	dest.NetworkPassphrase = passphrase
	dest.CurrentProtocolVersion = currentProtocolVersion
	dest.CoreSupportedProtocolVersion = coreSupportedProtocolVersion

	lb := hal.LinkBuilder{Base: httpx.BaseURL(ctx)}
	if friendBotURL != nil {
		friendbotLinkBuild := hal.LinkBuilder{Base: friendBotURL}
		l := friendbotLinkBuild.Link("{?addr}")
		dest.Links.Friendbot = &l
	}

	dest.Links.Account = lb.Link("/accounts/{account_id}")
	dest.Links.AccountTransactions = lb.PagedLink("/accounts/{account_id}/transactions")
	dest.Links.Assets = lb.Link("/assets{?asset_code,asset_issuer,cursor,limit,order}")
	dest.Links.Metrics = lb.Link("/metrics")
	dest.Links.OrderBook = lb.Link("/order_book{?selling_asset_type,selling_asset_code,selling_asset_issuer,buying_asset_type,buying_asset_code,buying_asset_issuer,limit}")
	dest.Links.Self = lb.Link("/")
	dest.Links.Transaction = lb.Link("/transactions/{hash}")
	dest.Links.Transactions = lb.PagedLink("/transactions")
}
