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

	row := Offer{
		SellerID:           offer.SellerId.Address(),
		OfferID:            offer.OfferId,
		SellingAsset:       offer.Selling,
		BuyingAsset:        offer.Buying,
		Amount:             offer.Amount,
		Pricen:             int32(offer.Price.N),
		Priced:             int32(offer.Price.D),
		Price:              price,
		Flags:              uint32(offer.Flags),
		Deleted:            false,
		LastModifiedLedger: uint32(lastModifiedLedger),
	}

	return i.builder.RowStruct(row)
}

func (i *offersBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
