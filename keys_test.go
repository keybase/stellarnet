package stellarnet

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
)

// Build, pack, and then unpack a transaction to see if our keys.go utility
// functions work correctly.

func setupTxWithOp(t *testing.T) (*keypair.Full, *keypair.Full, xdr.Transaction) {
	kp1, err := NewKeyPair()
	require.NoError(t, err)
	kp2, err := NewKeyPair()
	require.NoError(t, err)

	var tx xdr.Transaction
	err = tx.SourceAccount.SetAddress(kp1.Address())
	require.NoError(t, err)

	var op xdr.PaymentOp
	op.Amount = 1000
	err = op.Destination.SetAddress(kp2.Address())
	require.NoError(t, err)

	body, err := xdr.NewOperationBody(xdr.OperationTypePayment, op)
	require.NoError(t, err)

	tx.Operations = append(tx.Operations, xdr.Operation{Body: body})
	tx.SeqNum = 1
	tx.Fee = txnbuild.MinBaseFee

	return kp1, kp2, tx
}

func marshalTx(envType xdr.EnvelopeType, value interface{}) (string, error) {
	outerEnvelope, err := xdr.NewTransactionEnvelope(envType, value)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	_, err = xdr.Marshal(&buf, outerEnvelope)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return encoded, nil
}

func TestUnpackGoodTransaction(t *testing.T) {
	kp1, kp2, tx := setupTxWithOp(t)

	encoded, err := marshalTx(xdr.EnvelopeTypeEnvelopeTypeTx, xdr.TransactionV1Envelope{Tx: tx})
	require.NoError(t, err)
	t.Logf("%s", encoded)

	var txEnv xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(encoded, &txEnv)
	require.NoError(t, err)

	txUnpacked := txEnv.MustV1().Tx
	require.Equal(t, kp1.Address(), MuxedAccountToAccountString(txUnpacked.SourceAccount))
	require.Equal(t, kp2.Address(), MuxedAccountToAccountString(
		txUnpacked.Operations[0].Body.MustPaymentOp().Destination))
}
