package stellarnet

import (
	"errors"

	"github.com/stellar/go/keypair"
	snetwork "github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

// VerifyEnvelope verifies that there is a SourceAccount signature in the
// envelope.
func VerifyEnvelope(txEnv xdr.TransactionEnvelope) error {
	sourceMuxedAccount := txEnv.SourceAccount()
	sourceAccount := sourceMuxedAccount.ToAccountId()
	kp, err := keypair.Parse(sourceAccount.Address())
	if err != nil {
		return err
	}
	hash, err := snetwork.HashTransactionInEnvelope(txEnv, NetworkPassphrase())
	if err != nil {
		return err
	}

	var found bool
	for _, sig := range txEnv.Signatures() {
		if sig.Hint != kp.Hint() {
			continue
		}
		if err := kp.Verify(hash[:], sig.Signature); err != nil {
			return err
		}
		found = true
	}

	if !found {
		return errors.New("no signature found for source account")
	}

	return nil
}
