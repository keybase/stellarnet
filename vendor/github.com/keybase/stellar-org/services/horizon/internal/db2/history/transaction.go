package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TransactionByHash is a query that loads a single row from the
// `history_transactions` table based upon the provided hash.
func (q *Q) TransactionByHash(dest interface{}, hash string) error {
	byHash := selectTransaction.
		Where("ht.transaction_hash = ?", hash)
	byInnerHash := selectTransaction.
		Where("ht.inner_transaction_hash = ?", hash)

	byInnerHashString, args, err := byInnerHash.ToSql()
	if err != nil {
		return errors.Wrap(err, "could not get string for inner hash sql query")
	}
	union := byHash.Suffix("UNION ALL "+byInnerHashString, args...)

	return q.Get(dest, union)
}

// TransactionsByIDs fetches transactions from the `history_transactions` table
// which match the given ids
func (q *Q) TransactionsByIDs(ids ...int64) (map[int64]Transaction, error) {
	if len(ids) == 0 {
		return nil, errors.New("no id arguments provided")
	}

	sql := selectTransaction.Where(map[string]interface{}{
		"ht.id": ids,
	})

	var transactions []Transaction
	if err := q.Select(&transactions, sql); err != nil {
		return nil, err
	}

	byID := map[int64]Transaction{}
	for _, transaction := range transactions {
		byID[transaction.TotalOrderID.ID] = transaction
	}

	return byID, nil
}

// Transactions provides a helper to filter rows from the `history_transactions`
// table with pre-defined filters.  See `TransactionsQ` methods for the
// available filters.
func (q *Q) Transactions() *TransactionsQ {
	return &TransactionsQ{
		parent:        q,
		sql:           selectTransaction,
		includeFailed: false,
	}
}

// ForAccount filters the transactions collection to a specific account
func (q *TransactionsQ) ForAccount(aid string) *TransactionsQ {
	var account Account
	q.Err = q.parent.AccountByAddress(&account, aid)
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.
		Join("history_transaction_participants htp ON htp.history_transaction_id = ht.id").
		Where("htp.history_account_id = ?", account.ID)

	return q
}

// ForLedger filters the query to a only transactions in a specific ledger,
// specified by its sequence.
func (q *TransactionsQ) ForLedger(seq int32) *TransactionsQ {
	var ledger Ledger
	q.Err = q.parent.LedgerBySequence(&ledger, seq)
	if q.Err != nil {
		return q
	}

	start := toid.ID{LedgerSequence: seq}
	end := toid.ID{LedgerSequence: seq + 1}
	q.sql = q.sql.Where(
		"ht.id >= ? AND ht.id < ?",
		start.ToInt64(),
		end.ToInt64(),
	)

	return q
}

// IncludeFailed changes the query to include failed transactions.
func (q *TransactionsQ) IncludeFailed() *TransactionsQ {
	q.includeFailed = true
	return q
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *TransactionsQ) Page(page db2.PageQuery) *TransactionsQ {
	if q.Err != nil {
		return q
	}

	q.sql, q.Err = page.ApplyTo(q.sql, "ht.id")
	return q
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *TransactionsQ) Select(dest interface{}) error {
	if q.Err != nil {
		return q.Err
	}

	if !q.includeFailed {
		q.sql = q.sql.
			Where("(ht.successful = true OR ht.successful IS NULL)")
	}

	q.Err = q.parent.Select(dest, q.sql)
	if q.Err != nil {
		return q.Err
	}

	transactions, ok := dest.(*[]Transaction)
	if !ok {
		return errors.New("dest is not *[]Transaction")
	}

	for _, t := range *transactions {
		var resultXDR xdr.TransactionResult
		err := xdr.SafeUnmarshalBase64(t.TxResult, &resultXDR)
		if err != nil {
			return err
		}

		if !q.includeFailed {
			if !t.Successful {
				return errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s", t.TransactionHash)
			}

			if !resultXDR.Successful() {
				return errors.Errorf("Corrupted data! `include_failed=false` but returned transaction is failed: %s %s", t.TransactionHash, t.TxResult)
			}
		}

		// Check if `successful` equals resultXDR
		if t.Successful && !resultXDR.Successful() {
			return errors.Errorf("Corrupted data! `successful=true` but returned transaction is not success: %s %s", t.TransactionHash, t.TxResult)
		}

		if !t.Successful && resultXDR.Successful() {
			return errors.Errorf("Corrupted data! `successful=false` but returned transaction is success: %s %s", t.TransactionHash, t.TxResult)
		}
	}

	return nil
}

// QTransactions defines transaction related queries.
type QTransactions interface {
	NewTransactionBatchInsertBuilder(maxBatchSize int) TransactionBatchInsertBuilder
}

var selectTransaction = sq.Select(
	"ht.id, " +
		"ht.transaction_hash, " +
		"ht.ledger_sequence, " +
		"ht.application_order, " +
		"ht.account, " +
		"ht.account_sequence, " +
		"ht.max_fee, " +
		// `fee_charged` is NULL by default, DB needs to be reingested
		// to populate the value. If value is not present display `max_fee`.
		"COALESCE(ht.fee_charged, ht.max_fee) as fee_charged, " +
		"ht.operation_count, " +
		"ht.tx_envelope, " +
		"ht.tx_result, " +
		"ht.tx_meta, " +
		"ht.tx_fee_meta, " +
		"ht.created_at, " +
		"ht.updated_at, " +
		"COALESCE(ht.successful, true) as successful, " +
		"ht.signatures, " +
		"ht.memo_type, " +
		"ht.memo, " +
		"time_bounds, " +
		"hl.closed_at AS ledger_close_time, " +
		"ht.inner_transaction_hash, " +
		"ht.fee_account, " +
		"ht.new_max_fee, " +
		"ht.inner_signatures").
	From("history_transactions ht").
	LeftJoin("history_ledgers hl ON ht.ledger_sequence = hl.sequence")
