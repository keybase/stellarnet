package txnbuild

import (
	"github.com/stellar/go/amount"
	"github.com/stellar/go/price"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ManageBuyOffer represents the Stellar manage buy offer operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type ManageBuyOffer struct {
	Selling       Asset
	Buying        Asset
	Amount        string
	Price         string
	OfferID       int64
	SourceAccount Account
}

// BuildXDR for ManageBuyOffer returns a fully configured XDR Operation.
func (mo *ManageBuyOffer) BuildXDR() (xdr.Operation, error) {
	xdrSelling, err := mo.Selling.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Selling' field")
	}

	xdrBuying, err := mo.Buying.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to set XDR 'Buying' field")
	}

	xdrAmount, err := amount.Parse(mo.Amount)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Amount'")
	}

	xdrPrice, err := price.Parse(mo.Price)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to parse 'Price'")
	}

	opType := xdr.OperationTypeManageBuyOffer
	xdrOp := xdr.ManageBuyOfferOp{
		Selling:   xdrSelling,
		Buying:    xdrBuying,
		BuyAmount: xdrAmount,
		Price:     xdrPrice,
		OfferId:   xdr.Int64(mo.OfferID),
	}
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}

	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, mo.SourceAccount)
	return op, nil
}
