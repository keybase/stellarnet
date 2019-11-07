package history

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Add adds a new offer entry to the batch. `lastModifiedLedger` is another
// parameter because `xdr.OfferEntry` does not have a field to hold this value.
func (i *offersBatchInsertBuilder) Add(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) error {
	var price float64
	if offer.Price.D == 0 {
		return errors.New("offer price denominator is zero")
	} else if offer.Price.N > 0 {
		price = float64(offer.Price.N) / float64(offer.Price.D)
	}
	buyingAsset, err := xdr.MarshalBase64(offer.Buying)
	if err != nil {
		return errors.Wrap(err, "cannot marshal buying asset in offer")
	}
	sellingAsset, err := xdr.MarshalBase64(offer.Selling)
	if err != nil {
		return errors.Wrap(err, "cannot marshal selling asset in offer")
	}

	return i.builder.Row(map[string]interface{}{
		"sellerid":             offer.SellerId.Address(),
		"offerid":              offer.OfferId,
		"sellingasset":         sellingAsset,
		"buyingasset":          buyingAsset,
		"amount":               offer.Amount,
		"pricen":               offer.Price.N,
		"priced":               offer.Price.D,
		"price":                price,
		"flags":                offer.Flags,
		"last_modified_ledger": lastModifiedLedger,
	})
}

func (i *offersBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
