package history

import (
	"fmt"
	"math"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
)

// PagingToken returns a cursor for this trade
func (r *Trade) PagingToken() string {
	return fmt.Sprintf("%d-%d", r.HistoryOperationID, r.Order)
}

// HasPrice returns true if the trade has non-null price data
func (r *Trade) HasPrice() bool {
	return r.PriceN.Valid && r.PriceD.Valid
}

// Trades provides a helper to filter rows from the `history_trades` table
// with pre-defined filters.  See `TradesQ` methods for the available filters.
func (q *Q) Trades() *TradesQ {
	return &TradesQ{
		parent: q,
		sql: joinTradeAssets(
			joinTradeAccounts(
				selectTradeFields.From("history_trades htrd"),
				"history_accounts",
			),
			"history_assets",
		),
	}
}

// ReverseTrades provides a helper to filter rows from the `history_trades` table
// with pre-defined filters and reversed base/counter.  See `TradesQ` methods for the available filters.
func (q *Q) ReverseTrades() *TradesQ {
	return &TradesQ{
		parent: q,
		sql: joinTradeAssets(
			joinTradeAccounts(
				selectReverseTradeFields.From("history_trades htrd"),
				"history_accounts",
			),
			"history_assets",
		),
	}
}

// TradesForAssetPair provides a helper to filter rows from the `history_trades` table
// with the base filter of a specific asset pair.  See `TradesQ` methods for further available filters.
func (q *Q) TradesForAssetPair(baseAssetId int64, counterAssetId int64) *TradesQ {
	orderPreserved, baseAssetId, counterAssetId := getCanonicalAssetOrder(baseAssetId, counterAssetId)
	var trades *TradesQ
	if orderPreserved {
		trades = q.Trades()
	} else {
		trades = q.ReverseTrades()
	}
	return trades.forAssetPair(baseAssetId, counterAssetId)
}

// ForOffer filters the query results by the offer id.
func (q *TradesQ) ForOffer(id int64) *TradesQ {
	q.sql = q.sql.Where("(htrd.base_offer_id = ? OR htrd.counter_offer_id = ?)", id, id)
	return q
}

//Filter by asset pair. This function is private to ensure that correct order and proper select statement are coupled
func (q *TradesQ) forAssetPair(baseAssetId int64, counterAssetId int64) *TradesQ {
	q.sql = q.sql.Where(sq.Eq{"base_asset_id": baseAssetId, "counter_asset_id": counterAssetId})
	return q
}

// ForLedger adds a filter which only includes trades within the given ledger sequence
func (q *TradesQ) ForLedger(sequence int32, order string) *TradesQ {
	from := toid.ID{LedgerSequence: sequence}.ToInt64()
	to := toid.ID{LedgerSequence: sequence + 1}.ToInt64()

	q.sql = q.sql.Where(
		"htrd.history_operation_id >= ? AND htrd.history_operation_id <= ? ",
		from,
		to,
	).OrderBy(
		"htrd.history_operation_id " + order + ", htrd.order " + order,
	)

	return q
}

// ForAccount filter Trades by account id
func (q *TradesQ) ForAccount(aid string) *TradesQ {
	var account Account
	q.Err = q.parent.AccountByAddress(&account, aid)
	if q.Err != nil {
		return q
	}

	q.sql = q.sql.Where("(htrd.base_account_id = ? OR htrd.counter_account_id = ?)", account.ID, account.ID)
	return q
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *TradesQ) Page(page db2.PageQuery) *TradesQ {
	if q.Err != nil {
		return q
	}

	op, idx, err := page.CursorInt64Pair(db2.DefaultPairSep)
	if err != nil {
		q.Err = err
		return q
	}

	// constrain the second portion of the cursor pair to 32-bits
	if idx > math.MaxInt32 {
		idx = math.MaxInt32
	}

	// NOTE: Remember to test the queries below with EXPLAIN / EXPLAIN ANALYZE
	// before changing them.
	// This condition is using multicolumn index and it's easy to write it in a way that
	// DB will perform a full table scan.
	switch page.Order {
	case "asc":
		q.sql = q.sql.
			Where(`(
					 htrd.history_operation_id >= ?
				AND (
					 htrd.history_operation_id > ? OR
					(htrd.history_operation_id = ? AND htrd.order > ?)
				))`, op, op, op, idx).
			OrderBy("htrd.history_operation_id asc, htrd.order asc")
	case "desc":
		q.sql = q.sql.
			Where(`(
					 htrd.history_operation_id <= ?
				AND (
					 htrd.history_operation_id < ? OR
					(htrd.history_operation_id = ? AND htrd.order < ?)
				))`, op, op, op, idx).
			OrderBy("htrd.history_operation_id desc, htrd.order desc")
	}

	q.sql = q.sql.Limit(page.Limit)
	return q
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *TradesQ) Select(dest interface{}) error {
	if q.Err != nil {
		return q.Err
	}

	q.Err = q.parent.Select(dest, q.sql)
	return q.Err
}

func joinTradeAccounts(selectBuilder sq.SelectBuilder, historyAccountsTable string) sq.SelectBuilder {
	return selectBuilder.
		Join(historyAccountsTable + " base_accounts ON base_account_id = base_accounts.id").
		Join(historyAccountsTable + " counter_accounts ON counter_account_id = counter_accounts.id")
}

func joinTradeAssets(selectBuilder sq.SelectBuilder, historyAssetsTable string) sq.SelectBuilder {
	return selectBuilder.
		Join(historyAssetsTable + " base_assets ON base_asset_id = base_assets.id").
		Join(historyAssetsTable + " counter_assets ON counter_asset_id = counter_assets.id")
}

var selectTradeFields = sq.Select(
	"history_operation_id",
	"htrd.\"order\"",
	"htrd.ledger_closed_at",
	"htrd.offer_id",
	"htrd.base_offer_id",
	"base_accounts.address as base_account",
	"base_assets.asset_type as base_asset_type",
	"base_assets.asset_code as base_asset_code",
	"base_assets.asset_issuer as base_asset_issuer",
	"htrd.base_amount",
	"htrd.counter_offer_id",
	"counter_accounts.address as counter_account",
	"counter_assets.asset_type as counter_asset_type",
	"counter_assets.asset_code as counter_asset_code",
	"counter_assets.asset_issuer as counter_asset_issuer",
	"htrd.counter_amount",
	"htrd.base_is_seller",
	"htrd.price_n",
	"htrd.price_d",
)

var selectReverseTradeFields = sq.Select(
	"history_operation_id",
	"htrd.\"order\"",
	"htrd.ledger_closed_at",
	"htrd.offer_id",
	"htrd.counter_offer_id as base_offer_id",
	"counter_accounts.address as base_account",
	"counter_assets.asset_type as base_asset_type",
	"counter_assets.asset_code as base_asset_code",
	"counter_assets.asset_issuer as base_asset_issuer",
	"htrd.counter_amount as base_amount",
	"htrd.base_offer_id as counter_offer_id",
	"base_accounts.address as counter_account",
	"base_assets.asset_type as counter_asset_type",
	"base_assets.asset_code as counter_asset_code",
	"base_assets.asset_issuer as counter_asset_issuer",
	"htrd.base_amount as counter_amount",
	"NOT(htrd.base_is_seller) as base_is_seller",
	"htrd.price_d as price_n",
	"htrd.price_n as price_d",
)

func getCanonicalAssetOrder(assetId1 int64, assetId2 int64) (orderPreserved bool, baseAssetId int64, counterAssetId int64) {
	if assetId1 < assetId2 {
		return true, assetId1, assetId2
	} else {
		return false, assetId2, assetId1
	}
}

type QTrades interface {
	QCreateAccountsHistory
	NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder
	CreateAssets(assets []xdr.Asset, maxBatchSize int) (map[string]Asset, error)
}
