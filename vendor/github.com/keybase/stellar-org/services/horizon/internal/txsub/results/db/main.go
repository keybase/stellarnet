// Package results provides an implementation of the txsub.ResultProvider interface
// backed using the SQL databases used by both stellar core and horizon
package results

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/xdr"
)

// DB provides transaction submission results by querying the
// connected horizon and, if set, stellar core databases.
type DB struct {
	Core    *core.Q
	History *history.Q
	// SkipCoreChecks makes DB skip checking transaction result in Core DB if `true`.
	SkipCoreChecks bool
}

var _ txsub.ResultProvider = &DB{}

// ResultByHash implements txsub.ResultProvider
func (rp *DB) ResultByHash(ctx context.Context, hash string) txsub.Result {
	historyLatest := ledger.CurrentState().HistoryLatest

	// query history database
	var hr history.Transaction
	err := rp.History.TransactionByHash(&hr, hash)
	if err == nil {
		return txResultFromHistory(hr)
	}

	if !rp.History.NoRows(err) {
		return txsub.Result{Err: err}
	}

	if !rp.SkipCoreChecks {
		// query core database
		var cr core.Transaction
		// In the past we were searching for the transaction in core DB *after* the
		// latest ingested ledger. This was incorrect because history DB contains
		// successful transactions only. So it was possible that the transaction was
		// never found and clients were receiving Timeout errors.
		// However we can't change it to simply find a transaction by hash because
		// `txhistory` table does not have an index on `txid` field. Because of this
		// we query the last 120 ledgers (~10 minutes) to not kill the DB by searching
		// for a value on a table with millions of rows but also to support returning
		// the failed tx result (when resubmitting) for 10 minutes (or before core
		// clears `txhistory` table, whatever is first).
		// If you are modifying the code here, please do not make this error again.
		err = rp.Core.TransactionByHashAfterLedger(&cr, hash, historyLatest-120)
		if err == nil {
			if err = txSucceeded(cr); err == nil {
				// The transaction succeeded and was found in Stellar Core. The transaction should also
				// be present in Horizon's DB but it could be that horizon ingestion is behind Stellar Core.
				// We return ErrNoResults so the txsub will retry the horizon db lookup
				return txsub.Result{Err: txsub.ErrNoResults}
			} else {
				return txsub.Result{Err: err}
			}
		} else if !rp.Core.NoRows(err) {
			return txsub.Result{Err: err}
		}
	}

	// if no result was found in either db, return ErrNoResults
	return txsub.Result{Err: txsub.ErrNoResults}
}

func txResultFromHistory(tx history.Transaction) txsub.Result {
	var txResult xdr.TransactionResult
	err := xdr.SafeUnmarshalBase64(tx.TxResult, &txResult)
	if err == nil {
		if !txResult.Successful() {
			err = &txsub.FailedTransactionError{
				ResultXDR: tx.TxResult,
			}
		}
	}

	return txsub.Result{Err: err, Transaction: tx}
}

// txSucceeded returns nil if the the transaction succeeded
// otherwise, it returns an error
func txSucceeded(tx core.Transaction) error {
	// re-encode result to base64
	var raw bytes.Buffer
	_, err := xdr.Marshal(&raw, tx.Result.Result)
	if err != nil {
		return err
	}

	trx := base64.StdEncoding.EncodeToString(raw.Bytes())

	// if result is success, send a normal resposne
	if tx.Result.Successful() {
		return nil
	}

	// if failed, produce a FailedTransactionError
	return &txsub.FailedTransactionError{
		ResultXDR: trx,
	}
}
