package io

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// LedgerTransaction represents the data for a single transaction within a ledger.
type LedgerTransaction struct {
	Index    uint32
	Envelope xdr.TransactionEnvelope
	Result   xdr.TransactionResultPair
	// FeeChanges and Meta are low level values.
	// Use LedgerTransaction.GetChanges() for higher level access to ledger
	// entry changes.
	FeeChanges xdr.LedgerEntryChanges
	Meta       xdr.TransactionMeta
}

func (t *LedgerTransaction) txInternalError() bool {
	return t.Result.Result.Result.Code == xdr.TransactionResultCodeTxInternalError
}

// GetFeeChanges returns a developer friendly representation of LedgerEntryChanges
// connected to fees.
func (t *LedgerTransaction) GetFeeChanges() []Change {
	return getChangesFromLedgerEntryChanges(t.FeeChanges)
}

// GetChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains transaction changes and operation changes in that order. If the
// transaction failed with TxInternalError, operations and txChangesAfter are
// omitted. It doesn't support legacy TransactionMeta.V=0.
func (t *LedgerTransaction) GetChanges() ([]Change, error) {
	var changes []Change

	// Transaction meta
	switch t.Meta.V {
	case 0:
		return changes, errors.New("TransactionMeta.V=0 not supported")
	case 1:
		v1Meta := t.Meta.MustV1()
		txChanges := getChangesFromLedgerEntryChanges(v1Meta.TxChanges)
		changes = append(changes, txChanges...)

		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		for _, operationMeta := range v1Meta.Operations {
			opChanges := getChangesFromLedgerEntryChanges(
				operationMeta.Changes,
			)
			changes = append(changes, opChanges...)
		}

	case 2:
		v2Meta := t.Meta.MustV2()
		txChangesBefore := getChangesFromLedgerEntryChanges(v2Meta.TxChangesBefore)
		changes = append(changes, txChangesBefore...)

		// Ignore operations meta and txChangesAfter if txInternalError
		// https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		for _, operationMeta := range v2Meta.Operations {
			opChanges := getChangesFromLedgerEntryChanges(
				operationMeta.Changes,
			)
			changes = append(changes, opChanges...)
		}

		txChangesAfter := getChangesFromLedgerEntryChanges(v2Meta.TxChangesAfter)
		changes = append(changes, txChangesAfter...)
	default:
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return changes, nil
}

// GetOperationChanges returns a developer friendly representation of LedgerEntryChanges.
// It contains only operation changes.
func (t *LedgerTransaction) GetOperationChanges(operationIndex uint32) ([]Change, error) {
	changes := []Change{}

	// Transaction meta
	switch t.Meta.V {
	case 0:
		return changes, errors.New("TransactionMeta.V=0 not supported")
	case 1:
		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		v1Meta := t.Meta.MustV1()
		changes = operationChanges(v1Meta.Operations, operationIndex)
	case 2:
		// Ignore operations meta if txInternalError https://github.com/stellar/go/issues/2111
		if t.txInternalError() {
			return changes, nil
		}

		v2Meta := t.Meta.MustV2()
		changes = operationChanges(v2Meta.Operations, operationIndex)
	default:
		return changes, errors.New("Unsupported TransactionMeta version")
	}

	return changes, nil
}

func operationChanges(ops []xdr.OperationMeta, index uint32) []Change {
	if len(ops) == 0 || int(index) >= len(ops) {
		return []Change{}
	}

	operationMeta := ops[index]
	return getChangesFromLedgerEntryChanges(
		operationMeta.Changes,
	)
}

// getChangesFromLedgerEntryChanges transforms LedgerEntryChanges to []Change.
// Each `update` and `removed` is preceded with `state` and `create` changes
// are alone, without `state`. The transformation we're doing is to move each
// change (state/update, state/removed or create) to an array of pre/post pairs.
// Then:
// - for create, pre is null and post is a new entry,
// - for update, pre is previous state and post is the current state,
// - for removed, pre is previous state and post is null.
//
// stellar-core source:
// https://github.com/stellar/stellar-core/blob/e584b43/src/ledger/LedgerTxn.cpp#L582
func getChangesFromLedgerEntryChanges(ledgerEntryChanges xdr.LedgerEntryChanges) []Change {
	changes := []Change{}

	for i, entryChange := range ledgerEntryChanges {
		switch entryChange.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			created := entryChange.MustCreated()
			changes = append(changes, Change{
				Type: created.Data.Type,
				Pre:  nil,
				Post: &created,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			state := ledgerEntryChanges[i-1].MustState()
			updated := entryChange.MustUpdated()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state,
				Post: &updated,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			state := ledgerEntryChanges[i-1].MustState()
			changes = append(changes, Change{
				Type: state.Data.Type,
				Pre:  &state,
				Post: nil,
			})
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			continue
		default:
			panic("Invalid LedgerEntryChangeType")
		}
	}

	return changes
}
