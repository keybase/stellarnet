package txnbuild

import (
	"github.com/stellar/go/xdr"
)

// Operation represents the operation types of the Stellar network.
type Operation interface {
	BuildXDR() (xdr.Operation, error)
}

// SetOpSourceAccount sets the source account ID on an Operation.
func SetOpSourceAccount(op *xdr.Operation, sourceAccount Account) {
	if sourceAccount == nil {
		return
	}
	var opSourceAccountID xdr.AccountId
	opSourceAccountID.SetAddress(sourceAccount.GetAccountID())
	op.SourceAccount = &opSourceAccountID
}
