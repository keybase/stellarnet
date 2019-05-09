package stellarnet

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// OpSummary returns a string summary of an operation.
func OpSummary(op xdr.Operation) string {
	var opSource string
	if op.SourceAccount != nil {
		opSource = op.SourceAccount.Address()
	}

	// TODO: show this
	_ = opSource

	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		iop := op.Body.MustCreateAccountOp()
		return fmt.Sprintf("Create account %s with starting balance of %s XLM", iop.Destination.Address(), StringFromStellarXdrAmount(iop.StartingBalance))
	case xdr.OperationTypePayment:
		iop := op.Body.MustPaymentOp()
		return fmt.Sprintf("Pay %s to account %s", XDRAssetAmountSummary(iop.Amount, iop.Asset), iop.Destination.Address())
	case xdr.OperationTypePathPayment:
		iop := op.Body.MustPathPaymentOp()
		return fmt.Sprintf("Pay %s to account %s using at most %s", XDRAssetAmountSummary(iop.DestAmount, iop.DestAsset), iop.Destination.Address(), XDRAssetAmountSummary(iop.SendMax, iop.SendAsset))
	case xdr.OperationTypeManageOffer:
		iop := op.Body.MustManageOfferOp()
		if iop.OfferId == 0 {
			return fmt.Sprintf("Create offer selling %s for %s to buy %s", XDRAssetAmountSummary(iop.Amount, iop.Selling), XDRPriceString(iop.Price), XDRAssetSummary(iop.Buying))
		} else if iop.Amount == 0 {
			return fmt.Sprintf("Remove offer selling %s for %s to buy %s (id %d)", XDRAssetSummary(iop.Selling), XDRPriceString(iop.Price), XDRAssetSummary(iop.Buying), iop.OfferId)
		} else {
			return fmt.Sprintf("Update offer selling %s for %s to buy %s (id %d)", XDRAssetAmountSummary(iop.Amount, iop.Selling), XDRPriceString(iop.Price), XDRAssetSummary(iop.Buying), iop.OfferId)
		}
	case xdr.OperationTypeCreatePassiveOffer:
		iop := op.Body.MustCreatePassiveOfferOp()
		if iop.Amount == 0 {
			return fmt.Sprintf("Remove passive offer selling %s for %s to buy %s", XDRAssetSummary(iop.Selling), XDRPriceString(iop.Price), XDRAssetSummary(iop.Buying))
		}
		return fmt.Sprintf("Create passive offer selling %s for %s to buy %s", XDRAssetAmountSummary(iop.Amount, iop.Selling), XDRPriceString(iop.Price), XDRAssetSummary(iop.Buying))
	case xdr.OperationTypeSetOptions:
		iop := op.Body.MustSetOptionsOp()
		var all []string
		if iop.InflationDest != nil {
			all = append(all, fmt.Sprintf("Set inflation destination to %s", iop.InflationDest.Address()))
		}
		if iop.ClearFlags != nil {
			all = append(all, fmt.Sprintf("Clear account flags %b", *iop.ClearFlags))
		}
		if iop.SetFlags != nil {
			all = append(all, fmt.Sprintf("Set account flags %b", *iop.SetFlags))
		}
		if iop.MasterWeight != nil {
			all = append(all, fmt.Sprintf("Set master key weight to %d", *iop.MasterWeight))
		}
		if iop.LowThreshold != nil {
			all = append(all, fmt.Sprintf("Set low threshold to %d", *iop.LowThreshold))
		}
		if iop.MedThreshold != nil {
			all = append(all, fmt.Sprintf("Set medium threshold to %d", *iop.MedThreshold))
		}
		if iop.HighThreshold != nil {
			all = append(all, fmt.Sprintf("Set high threshold to %d", *iop.HighThreshold))
		}
		if iop.HomeDomain != nil {
			all = append(all, fmt.Sprintf("Set home domain to %q", *iop.HomeDomain))
		}
		if iop.Signer != nil {
			all = append(all, fmt.Sprintf("Set signer key %s with weight %d", iop.Signer.Key.Address(), iop.Signer.Weight))
		}
	case xdr.OperationTypeChangeTrust:
		iop := op.Body.MustChangeTrustOp()
		_ = iop
	case xdr.OperationTypeAllowTrust:
		iop := op.Body.MustAllowTrustOp()
		_ = iop
	case xdr.OperationTypeAccountMerge:
		// oh of cource, MustDestination...why would it possibly match
		// everything else?
		iop := op.Body.MustDestination()
		_ = iop
	case xdr.OperationTypeManageData:
		iop := op.Body.MustManageDataOp()
		_ = iop
	default:
		return "invalid operation type"
	}

	return "something went wrong"
}
