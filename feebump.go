package stellarnet

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

func makeFeeBumpInnerTx(env xdr.TransactionEnvelope) (res xdr.FeeBumpTransactionInnerTx, err error) {
	switch env.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		return xdr.NewFeeBumpTransactionInnerTx(env.Type, env.MustV1())
	default:
		return res, fmt.Errorf("invalid envelope type %q for FeeBumpTransactionInnerTx, V1 is required",
			res.Type.String())
	}
}

// FeeBumpTransactionWithFeeSource wraps transaction from base64 xdr encoded
// `envelope` string into a Fee Bump transaction, bumping the fee to `fee` with
// FeeSource `feeSource`. This function returns a SignResult with the fee bump
// envelope with the transaction signed by `signer`.
func FeeBumpTransactionWithFeeSource(envelope string, signer SeedStr, feeSource AddressStr, fee uint64) (res SignResult, err error) {
	// Unpack target tx from envelope
	var txEnv xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(envelope, &txEnv)
	if err != nil {
		return res, fmt.Errorf("unable to unpack envelope: %w", err)
	}

	var feeBumpTx xdr.FeeBumpTransaction
	feeBumpTx.Fee = xdr.Int64(fee)
	sourceMux, err := feeSource.MuxedAccount()
	if err != nil {
		return res, fmt.Errorf("invalid feeSource (unable to convert to MuxedAccount): %w", err)
	}
	feeBumpTx.FeeSource = sourceMux
	feeBumpTx.InnerTx, err = makeFeeBumpInnerTx(txEnv)
	if err != nil {
		return res, fmt.Errorf("unable to set fee bump inner tx: %w", err)
	}

	feeBumpHash, err := network.HashFeeBumpTransaction(feeBumpTx, NetworkPassphrase())
	if err != nil {
		return res, fmt.Errorf("unable to hash tx: %w", err)
	}

	kp, err := keypair.Parse(signer.SecureNoLogString())
	if err != nil {
		return res, fmt.Errorf("cannot parse signer keypair: %w", err)
	}
	sig, err := kp.SignDecorated(feeBumpHash[:])
	if err != nil {
		return res, fmt.Errorf("cannot sign: %w", err)
	}

	var feeBumpTxEnv xdr.FeeBumpTransactionEnvelope
	feeBumpTxEnv.Tx = feeBumpTx
	feeBumpTxEnv.Signatures = append(feeBumpTxEnv.Signatures, sig)

	outerEnvelope, err := xdr.NewTransactionEnvelope(xdr.EnvelopeTypeEnvelopeTypeTxFeeBump, feeBumpTxEnv)
	if err != nil {
		return res, fmt.Errorf("unable to create fee bump transaction envelope: %w", err)
	}
	var buf bytes.Buffer
	_, err = xdr.Marshal(&buf, outerEnvelope)
	signed := base64.StdEncoding.EncodeToString(buf.Bytes())
	txHashHex := hex.EncodeToString(feeBumpHash[:])

	return SignResult{
		Signed: signed,
		TxHash: txHashHex,
	}, nil
}

// FeeBumpTransaction wraps transaction from `envelope` in a FeeBump
// transaction. FeeBump transaction is signed by `signer` who is also the
// FeeSource.
func FeeBumpTransaction(envelope string, signer SeedStr, fee uint64) (SignResult, error) {
	addr, err := signer.Address()
	if err != nil {
		return SignResult{}, err
	}
	return FeeBumpTransactionWithFeeSource(envelope, signer, addr, fee)
}
