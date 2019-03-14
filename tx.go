package stellarnet

import (
	"encoding/hex"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/build"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
)

// Tx is a data structure used for making a Stellar transaction.
// After creating one with NewBaseTx(), add to it with the various
// Add* functions, and finally, Sign() it.
//
// Any errors that occur during Add* functions are delayed to return
// when the Sign() function is called in order to make the transaction
// building code cleaner.
//
// Since this struct contains the secret seed, it should be disposed of
// and not held in memory for any longer than necessary.
type Tx struct {
	from SeedStr
	muts []build.TransactionMutator
	err  error

	accountID AddressStr
	internal  xdr.Transaction
	seqnoProv build.SequenceProvider
}

// NewBaseTx creates a Tx with the common transaction elements.
func NewBaseTx(from AddressStr, seqnoProvider build.SequenceProvider, baseFee uint64) *Tx {
	if baseFee < build.DefaultBaseFee {
		baseFee = build.DefaultBaseFee
	}
	t := &Tx{
		accountID: from,
		internal: xdr.Transaction{
			Fee: xdr.Uint32(baseFee),
		},
		seqnoProv: seqnoProvider,
	}
	t.internal.SourceAccount.SetAddress(from.String())
	return t
}

// newBaseTxSeed is a convenience function to get the address out of from before
// calling NewBaseTx.
func newBaseTxSeed(from SeedStr, seqnoProvider build.SequenceProvider, baseFee uint64) (*Tx, error) {
	fromAddress, err := from.Address()
	if err != nil {
		return nil, err
	}
	return NewBaseTx(fromAddress, seqnoProvider, baseFee), nil
}

// AddPaymentOp adds a payment operation to the transaction.
func (t *Tx) AddPaymentOp(to AddressStr, amt string) {
	if t.err != nil {
		return
	}
	if t.IsFull() {
		t.err = ErrTxOpFull
		return
	}

	var op xdr.PaymentOp
	op.Amount, t.err = amount.Parse(amt)
	if t.err != nil {
		return
	}
	op.Destination.SetAddress(to.String())

	body, err := xdr.NewOperationBody(xdr.OperationTypePayment, op)
	if err != nil {
		t.err = err
		return
	}
	wop := xdr.Operation{
		Body: body,
	}
	t.internal.Operations = append(t.internal.Operations, wop)
}

// AddCreateAccountOp adds a create_account operation to the transaction.
func (t *Tx) AddCreateAccountOp(to AddressStr, amt string) {
	if t.err != nil {
		return
	}
	if t.IsFull() {
		t.err = ErrTxOpFull
		return
	}

	var op xdr.CreateAccountOp
	op.StartingBalance, t.err = amount.Parse(amt)
	if t.err != nil {
		return
	}
	op.Destination.SetAddress(to.String())

	body, err := xdr.NewOperationBody(xdr.OperationTypeCreateAccount, op)
	if err != nil {
		t.err = err
		return
	}
	wop := xdr.Operation{
		Body: body,
	}
	t.internal.Operations = append(t.internal.Operations, wop)
}

// AddAccountMergeOp adds an account_merge operation to the transaction.
func (t *Tx) AddAccountMergeOp(to AddressStr) {
	if t.err != nil {
		return
	}
	if t.IsFull() {
		t.err = ErrTxOpFull
		return
	}

	t.muts = append(t.muts, build.AccountMerge(
		build.Destination{AddressOrSeed: to.String()},
	))
}

// AddInflationDestinationOp adds a set_options operation for the inflation
// destination to the transaction.
func (t *Tx) AddInflationDestinationOp(to AddressStr) {
	if t.err != nil {
		return
	}
	if t.IsFull() {
		t.err = ErrTxOpFull
		return
	}

	t.muts = append(t.muts, build.SetOptions(build.InflationDest(to.String())))
}

func (t *Tx) haveMemo() bool {
	return t.internal.Memo.Type != xdr.MemoTypeMemoNone
}

// AddMemoText adds a text memo to the transaction.  There can only
// be one memo.
func (t *Tx) AddMemoText(memo string) {
	if t.err != nil {
		return
	}
	if t.haveMemo() {
		t.err = ErrMemoExists
		return
	}

	t.muts = append(t.muts, build.MemoText{Value: memo})
}

// AddMemoID adds an ID memo to the transaction.  There can only
// be one memo.
func (t *Tx) AddMemoID(id *uint64) {
	if t.err != nil {
		return
	}
	if id == nil {
		return
	}
	if t.haveMemo() {
		t.err = ErrMemoExists
		return
	}

	t.muts = append(t.muts, build.MemoID{Value: *id})
}

// AddTimebounds adds time bounds to the transaction.
func (t *Tx) AddTimebounds(min, max uint64) {
	t.AddBuiltTimebounds(&build.Timebounds{MinTime: min, MaxTime: max})
}

// AddBuiltTimebounds adds time bounds to the transaction with a *build.Timebounds.
func (t *Tx) AddBuiltTimebounds(bt *build.Timebounds) {
	if t.err != nil {
		return
	}
	if bt == nil {
		return
	}
	if t.internal.TimeBounds != nil {
		t.err = ErrTimeboundsExist
		return
	}

	t.muts = append(t.muts, *bt)
}

// Sign builds the transaction and signs it.
func (t *Tx) Sign() (SignResult, error) {
	if t.err != nil {
		return SignResult{}, errMap(t.err)
	}
	if len(t.internal.Operations) == 0 {
		return SignResult{}, errMap(ErrNoOps)
	}
	b, err := t.builder()
	if err != nil {
		return SignResult{}, errMap(err)
	}
	return sign(t.from, b)
}

// IsFull returns true if there are already 100 operations in the transaction.
func (t *Tx) IsFull() bool {
	return len(t.internal.Operations) >= 100
}

func (t *Tx) builder() (*build.TransactionBuilder, error) {
	b, err := build.Transaction(t.muts...)
	if err != nil {
		return nil, errMap(err)
	}
	return b, nil
}

// SignResult contains the result of signing a transaction.
type SignResult struct {
	Seqno  uint64
	Signed string // signed transaction (base64)
	TxHash string // transaction hash (hex)
}

func (t *Tx) sign(from SeedStr) (SignResult, error) {
	var err error
	t.internal.SeqNum, err = t.seqnoProv.SequenceForAccount(t.accountID.String())
	if err != nil {
		return SignResult{}, err
	}
	envelope := xdr.TransactionEnvelope{Tx: t.internal}
	hash, err := network.HashTransaction(&t.internal, t.NetworkPassphrase)
	if err != nil {
		return SignResult{}, err
	}

	kp, err := keypair.Parse(from.SecureNoLogString())
	if err != nil {
		return SignResult{}, err
	}
	sig, err := kp.SignDecorated(hash[:])
	if err != nil {
		return SignResult{}, err
	}
	envelope.Signatures = append(envelope.Signatures, sig)

	signed, err := envelope.Base64()
	if err != nil {
		return SignResult{}, err
	}
	txHashHex := hex.EncodeToString(hash[:])

	return SignResult{
		Seqno:  seqno,
		Signed: signed,
		TxHash: txHashHex,
	}, nil

}

// sign signs and base64-encodes a transaction.
func sign(from SeedStr, tx *build.TransactionBuilder) (res SignResult, err error) {
	txe, err := tx.Sign(from.SecureNoLogString())
	if err != nil {
		return res, errMap(err)
	}
	seqno := uint64(txe.E.Tx.SeqNum)
	signed, err := txe.Base64()
	if err != nil {
		return res, errMap(err)
	}
	txHashHex, err := tx.HashHex()
	if err != nil {
		return res, errMap(err)
	}
	return SignResult{
		Seqno:  seqno,
		Signed: signed,
		TxHash: txHashHex,
	}, nil
}
