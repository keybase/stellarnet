// Package history contains database record definitions useable for
// reading rows from a the history portion of horizon's database
package history

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	// account effects

	// EffectAccountCreated effects occur when a new account is created
	EffectAccountCreated EffectType = 0 // from create_account

	// EffectAccountRemoved effects occur when one account is merged into another
	EffectAccountRemoved EffectType = 1 // from merge_account

	// EffectAccountCredited effects occur when an account receives some currency
	EffectAccountCredited EffectType = 2 // from create_account, payment, path_payment, merge_account

	// EffectAccountDebited effects occur when an account sends some currency
	EffectAccountDebited EffectType = 3 // from create_account, payment, path_payment, create_account

	// EffectAccountThresholdsUpdated effects occur when an account changes its
	// multisig thresholds.
	EffectAccountThresholdsUpdated EffectType = 4 // from set_options

	// EffectAccountHomeDomainUpdated effects occur when an account changes its
	// home domain.
	EffectAccountHomeDomainUpdated EffectType = 5 // from set_options

	// EffectAccountFlagsUpdated effects occur when an account changes its
	// account flags, either clearing or setting.
	EffectAccountFlagsUpdated EffectType = 6 // from set_options

	// EffectAccountInflationDestinationUpdated effects occur when an account changes its
	// inflation destination.
	EffectAccountInflationDestinationUpdated EffectType = 7 // from set_options

	// signer effects

	// EffectSignerCreated occurs when an account gains a signer
	EffectSignerCreated EffectType = 10 // from set_options

	// EffectSignerRemoved occurs when an account loses a signer
	EffectSignerRemoved EffectType = 11 // from set_options

	// EffectSignerUpdated occurs when an account changes the weight of one of its
	// signers.
	EffectSignerUpdated EffectType = 12 // from set_options

	// trustline effects

	// EffectTrustlineCreated occurs when an account trusts an anchor
	EffectTrustlineCreated EffectType = 20 // from change_trust

	// EffectTrustlineRemoved occurs when an account removes struct by setting the
	// limit of a trustline to 0
	EffectTrustlineRemoved EffectType = 21 // from change_trust

	// EffectTrustlineUpdated occurs when an account changes a trustline's limit
	EffectTrustlineUpdated EffectType = 22 // from change_trust, allow_trust

	// EffectTrustlineAuthorized occurs when an anchor has AUTH_REQUIRED flag set
	// to true and it authorizes another account's trustline
	EffectTrustlineAuthorized EffectType = 23 // from allow_trust

	// EffectTrustlineDeauthorized occurs when an anchor revokes access to a asset
	// it issues.
	EffectTrustlineDeauthorized EffectType = 24 // from allow_trust

	// EffectTrustlineAuthorizedToMaintainLiabilities occurs when an anchor has AUTH_REQUIRED flag set
	// to true and it authorizes another account's trustline to maintain liabilities
	EffectTrustlineAuthorizedToMaintainLiabilities EffectType = 25 // from allow_trust

	// trading effects

	// EffectOfferCreated occurs when an account offers to trade an asset
	EffectOfferCreated EffectType = 30 // from manage_offer, creat_passive_offer

	// EffectOfferRemoved occurs when an account removes an offer
	EffectOfferRemoved EffectType = 31 // from manage_offer, creat_passive_offer, path_payment

	// EffectOfferUpdated occurs when an offer is updated by the offering account.
	EffectOfferUpdated EffectType = 32 // from manage_offer, creat_passive_offer, path_payment

	// EffectTrade occurs when a trade is initiated because of a path payment or
	// offer operation.
	EffectTrade EffectType = 33 // from manage_offer, creat_passive_offer, path_payment

	// data effects

	// EffectDataCreated occurs when an account gets a new data field
	EffectDataCreated EffectType = 40 // from manage_data

	// EffectDataRemoved occurs when an account removes a data field
	EffectDataRemoved EffectType = 41 // from manage_data

	// EffectDataUpdated occurs when an account changes a data field's value
	EffectDataUpdated EffectType = 42 // from manage_data

	// EffectSequenceBumped occurs when an account bumps their sequence number
	EffectSequenceBumped EffectType = 43 // from bump_sequence

)

// Account is a row of data from the `history_accounts` table
type Account struct {
	ID      int64  `db:"id"`
	Address string `db:"address"`
}

// AccountsQ is a helper struct to aid in configuring queries that loads
// slices of account structs.
type AccountsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// AccountEntry is a row of data from the `account` table
type AccountEntry struct {
	AccountID            string `db:"account_id"`
	Balance              int64  `db:"balance"`
	BuyingLiabilities    int64  `db:"buying_liabilities"`
	SellingLiabilities   int64  `db:"selling_liabilities"`
	SequenceNumber       int64  `db:"sequence_number"`
	NumSubEntries        uint32 `db:"num_subentries"`
	InflationDestination string `db:"inflation_destination"`
	HomeDomain           string `db:"home_domain"`
	Flags                uint32 `db:"flags"`
	MasterWeight         byte   `db:"master_weight"`
	ThresholdLow         byte   `db:"threshold_low"`
	ThresholdMedium      byte   `db:"threshold_medium"`
	ThresholdHigh        byte   `db:"threshold_high"`
	LastModifiedLedger   uint32 `db:"last_modified_ledger"`
}

type AccountsBatchInsertBuilder interface {
	Add(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) error
	Exec() error
}

type IngestionQ interface {
	QAccounts
	QAssetStats
	QData
	QEffects
	QLedgers
	QOffers
	QOperations
	// QParticipants
	// Copy the small interfaces with shared methods directly, otherwise error:
	// duplicate method CreateAccounts
	NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) TransactionParticipantsBatchInsertBuilder
	NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder
	QSigners
	//QTrades
	NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder
	CreateAssets(assets []xdr.Asset, batchSize int) (map[string]Asset, error)
	QTransactions
	QTrustLines

	Begin() error
	BeginTx(*sql.TxOptions) error
	Commit() error
	CloneIngestionQ() IngestionQ
	Rollback() error
	GetTx() *sqlx.Tx
	GetExpIngestVersion() (int, error)
	UpdateExpStateInvalid(bool) error
	UpdateExpIngestVersion(int) error
	GetExpStateInvalid() (bool, error)
	GetLatestLedger() (uint32, error)
	GetOfferCompactionSequence() (uint32, error)
	TruncateExpingestStateTables() error
	DeleteRangeAll(start, end int64) error
}

// QAccounts defines account related queries.
type QAccounts interface {
	NewAccountsBatchInsertBuilder(maxBatchSize int) AccountsBatchInsertBuilder
	GetAccountsByIDs(ids []string) ([]AccountEntry, error)
	UpsertAccounts(accounts []xdr.LedgerEntry) error
	RemoveAccount(accountID string) (int64, error)
}

// AccountSigner is a row of data from the `accounts_signers` table
type AccountSigner struct {
	Account string `db:"account_id"`
	Signer  string `db:"signer"`
	Weight  int32  `db:"weight"`
}

type AccountSignersBatchInsertBuilder interface {
	Add(signer AccountSigner) error
	Exec() error
}

// accountSignersBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type accountSignersBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// Data is a row of data from the `account_data` table
type Data struct {
	AccountID          string           `db:"account_id"`
	Name               string           `db:"name"`
	Value              AccountDataValue `db:"value"`
	LastModifiedLedger uint32           `db:"last_modified_ledger"`
}

type AccountDataValue []byte

type AccountDataBatchInsertBuilder interface {
	Add(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) error
	Exec() error
}

// accountDataBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type accountDataBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// QData defines account data related queries.
type QData interface {
	NewAccountDataBatchInsertBuilder(maxBatchSize int) AccountDataBatchInsertBuilder
	CountAccountsData() (int, error)
	GetAccountDataByKeys(keys []xdr.LedgerKeyData) ([]Data, error)
	InsertAccountData(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) (int64, error)
	UpdateAccountData(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) (int64, error)
	RemoveAccountData(key xdr.LedgerKeyData) (int64, error)
}

// Asset is a row of data from the `history_assets` table
type Asset struct {
	ID     int64  `db:"id"`
	Type   string `db:"asset_type"`
	Code   string `db:"asset_code"`
	Issuer string `db:"asset_issuer"`
}

// AssetStat is a row in the asset_stats table representing the stats per Asset
type AssetStat struct {
	ID          int64  `db:"id"`
	Amount      string `db:"amount"`
	NumAccounts int32  `db:"num_accounts"`
	Flags       int8   `db:"flags"`
	Toml        string `db:"toml"`
}

// ExpAssetStat is a row in the exp_asset_stats table representing the stats per Asset
type ExpAssetStat struct {
	AssetType   xdr.AssetType `db:"asset_type"`
	AssetCode   string        `db:"asset_code"`
	AssetIssuer string        `db:"asset_issuer"`
	Amount      string        `db:"amount"`
	NumAccounts int32         `db:"num_accounts"`
}

// PagingToken returns a cursor for this asset stat
func (e ExpAssetStat) PagingToken() string {
	return fmt.Sprintf(
		"%s_%s_%s",
		e.AssetCode,
		e.AssetIssuer,
		xdr.AssetTypeToString[e.AssetType],
	)
}

// QAssetStats defines exp_asset_stats related queries.
type QAssetStats interface {
	InsertAssetStats(stats []ExpAssetStat, batchSize int) error
	InsertAssetStat(stat ExpAssetStat) (int64, error)
	UpdateAssetStat(stat ExpAssetStat) (int64, error)
	GetAssetStat(assetType xdr.AssetType, assetCode, assetIssuer string) (ExpAssetStat, error)
	RemoveAssetStat(assetType xdr.AssetType, assetCode, assetIssuer string) (int64, error)
	GetAssetStats(assetCode, assetIssuer string, page db2.PageQuery) ([]ExpAssetStat, error)
	CountTrustLines() (int, error)
}

type QCreateAccountsHistory interface {
	CreateAccounts(addresses []string, maxBatchSize int) (map[string]int64, error)
}

// Effect is a row of data from the `history_effects` table
type Effect struct {
	HistoryAccountID   int64       `db:"history_account_id"`
	Account            string      `db:"address"`
	HistoryOperationID int64       `db:"history_operation_id"`
	Order              int32       `db:"order"`
	Type               EffectType  `db:"type"`
	DetailsString      null.String `db:"details"`
}

// TradeEffectDetails is a struct of data from `effects.DetailsString`
// when the effect type is trade
type TradeEffectDetails struct {
	Seller            string `json:"seller"`
	OfferID           int64  `json:"offer_id"`
	SoldAmount        string `json:"sold_amount"`
	SoldAssetType     string `json:"sold_asset_type"`
	SoldAssetCode     string `json:"sold_asset_code,omitempty"`
	SoldAssetIssuer   string `json:"sold_asset_issuer,omitempty"`
	BoughtAmount      string `json:"bought_amount"`
	BoughtAssetType   string `json:"bought_asset_type"`
	BoughtAssetCode   string `json:"bought_asset_code,omitempty"`
	BoughtAssetIssuer string `json:"bought_asset_issuer,omitempty"`
}

// SequenceBumped is a struct of data from `effects.DetailsString`
// when the effect type is sequence bumped.
type SequenceBumped struct {
	NewSeq int64 `json:"new_seq"`
}

// EffectsQ is a helper struct to aid in configuring queries that loads
// slices of Ledger structs.
type EffectsQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// EffectType is the numeric type for an effect, used as the `type` field in the
// `history_effects` table.
type EffectType int

// FeeStats is a row of data from the min, mode, percentile aggregate functions over the
// `history_transactions` table.
type FeeStats struct {
	FeeChargedMax  null.Int `db:"fee_charged_max"`
	FeeChargedMin  null.Int `db:"fee_charged_min"`
	FeeChargedMode null.Int `db:"fee_charged_mode"`
	FeeChargedP10  null.Int `db:"fee_charged_p10"`
	FeeChargedP20  null.Int `db:"fee_charged_p20"`
	FeeChargedP30  null.Int `db:"fee_charged_p30"`
	FeeChargedP40  null.Int `db:"fee_charged_p40"`
	FeeChargedP50  null.Int `db:"fee_charged_p50"`
	FeeChargedP60  null.Int `db:"fee_charged_p60"`
	FeeChargedP70  null.Int `db:"fee_charged_p70"`
	FeeChargedP80  null.Int `db:"fee_charged_p80"`
	FeeChargedP90  null.Int `db:"fee_charged_p90"`
	FeeChargedP95  null.Int `db:"fee_charged_p95"`
	FeeChargedP99  null.Int `db:"fee_charged_p99"`
	MaxFeeMax      null.Int `db:"max_fee_max"`
	MaxFeeMin      null.Int `db:"max_fee_min"`
	MaxFeeMode     null.Int `db:"max_fee_mode"`
	MaxFeeP10      null.Int `db:"max_fee_p10"`
	MaxFeeP20      null.Int `db:"max_fee_p20"`
	MaxFeeP30      null.Int `db:"max_fee_p30"`
	MaxFeeP40      null.Int `db:"max_fee_p40"`
	MaxFeeP50      null.Int `db:"max_fee_p50"`
	MaxFeeP60      null.Int `db:"max_fee_p60"`
	MaxFeeP70      null.Int `db:"max_fee_p70"`
	MaxFeeP80      null.Int `db:"max_fee_p80"`
	MaxFeeP90      null.Int `db:"max_fee_p90"`
	MaxFeeP95      null.Int `db:"max_fee_p95"`
	MaxFeeP99      null.Int `db:"max_fee_p99"`
}

// KeyValueStoreRow represents a row in key value store.
type KeyValueStoreRow struct {
	Key   string `db:"key"`
	Value string `db:"value"`
}

// LatestLedger represents a response from the raw LatestLedgerBaseFeeAndSequence
// query.
type LatestLedger struct {
	BaseFee  int32 `db:"base_fee"`
	Sequence int32 `db:"sequence"`
}

// Ledger is a row of data from the `history_ledgers` table
type Ledger struct {
	TotalOrderID
	Sequence                   int32       `db:"sequence"`
	ImporterVersion            int32       `db:"importer_version"`
	LedgerHash                 string      `db:"ledger_hash"`
	PreviousLedgerHash         null.String `db:"previous_ledger_hash"`
	TransactionCount           int32       `db:"transaction_count"`
	SuccessfulTransactionCount *int32      `db:"successful_transaction_count"`
	FailedTransactionCount     *int32      `db:"failed_transaction_count"`
	OperationCount             int32       `db:"operation_count"`
	ClosedAt                   time.Time   `db:"closed_at"`
	CreatedAt                  time.Time   `db:"created_at"`
	UpdatedAt                  time.Time   `db:"updated_at"`
	TotalCoins                 int64       `db:"total_coins"`
	FeePool                    int64       `db:"fee_pool"`
	BaseFee                    int32       `db:"base_fee"`
	BaseReserve                int32       `db:"base_reserve"`
	MaxTxSetSize               int32       `db:"max_tx_set_size"`
	ProtocolVersion            int32       `db:"protocol_version"`
	LedgerHeaderXDR            null.String `db:"ledger_header"`
}

// LedgerCapacityUsageStats contains ledgers fullness stats.
type LedgerCapacityUsageStats struct {
	CapacityUsage null.String `db:"ledger_capacity_usage"`
}

// LedgerCache is a helper struct to load ledger data related to a batch of
// sequences.
type LedgerCache struct {
	Records map[int32]Ledger

	lock   sync.Mutex
	queued map[int32]struct{}
}

// LedgersQ is a helper struct to aid in configuring queries that loads
// slices of Ledger structs.
type LedgersQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// Operation is a row of data from the `history_operations` table
type Operation struct {
	TotalOrderID
	TransactionID         int64             `db:"transaction_id"`
	TransactionHash       string            `db:"transaction_hash"`
	TxResult              string            `db:"tx_result"`
	ApplicationOrder      int32             `db:"application_order"`
	Type                  xdr.OperationType `db:"type"`
	DetailsString         null.String       `db:"details"`
	SourceAccount         string            `db:"source_account"`
	TransactionSuccessful bool              `db:"transaction_successful"`
}

// ManageOffer is a struct of data from `operations.DetailsString`
// when the operation type is manage sell offer or manage buy offer
type ManageOffer struct {
	OfferID int64 `json:"offer_id"`
}

// Offer is row of data from the `offers` table from horizon DB
type Offer struct {
	SellerID string    `db:"seller_id"`
	OfferID  xdr.Int64 `db:"offer_id"`

	SellingAsset xdr.Asset `db:"selling_asset"`
	BuyingAsset  xdr.Asset `db:"buying_asset"`

	Amount             xdr.Int64 `db:"amount"`
	Pricen             int32     `db:"pricen"`
	Priced             int32     `db:"priced"`
	Price              float64   `db:"price"`
	Flags              uint32    `db:"flags"`
	Deleted            bool      `db:"deleted"`
	LastModifiedLedger uint32    `db:"last_modified_ledger"`
}

type OffersBatchInsertBuilder interface {
	Add(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) error
	Exec() error
}

// offersBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type offersBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// OperationsQ is a helper struct to aid in configuring queries that loads
// slices of Operation structs.
type OperationsQ struct {
	Err                 error
	parent              *Q
	sql                 sq.SelectBuilder
	opIdCol             string
	includeFailed       bool
	includeTransactions bool
}

// Q is a helper struct on which to hang common_trades queries against a history
// portion of the horizon database.
type Q struct {
	*db.Session
}

// QSigners defines signer related queries.
type QSigners interface {
	GetLastLedgerExpIngestNonBlocking() (uint32, error)
	GetLastLedgerExpIngest() (uint32, error)
	UpdateLastLedgerExpIngest(ledgerSequence uint32) error
	AccountsForSigner(signer string, page db2.PageQuery) ([]AccountSigner, error)
	NewAccountSignersBatchInsertBuilder(maxBatchSize int) AccountSignersBatchInsertBuilder
	CreateAccountSigner(account, signer string, weight int32) (int64, error)
	RemoveAccountSigner(account, signer string) (int64, error)
	SignersForAccounts(accounts []string) ([]AccountSigner, error)
	CountAccounts() (int, error)
}

// OffersQuery is a helper struct to configure queries to offers
type OffersQuery struct {
	PageQuery db2.PageQuery
	SellerID  string
	Selling   *xdr.Asset
	Buying    *xdr.Asset
}

// TotalOrderID represents the ID portion of rows that are identified by the
// "TotalOrderID".  See total_order_id.go in the `db` package for details.
type TotalOrderID struct {
	ID int64 `db:"id"`
}

// Trade represents a trade from the trades table, joined with asset information from the assets table and account
// addresses from the accounts table
type Trade struct {
	HistoryOperationID int64     `db:"history_operation_id"`
	Order              int32     `db:"order"`
	LedgerCloseTime    time.Time `db:"ledger_closed_at"`
	OfferID            int64     `db:"offer_id"`
	BaseOfferID        *int64    `db:"base_offer_id"`
	BaseAccount        string    `db:"base_account"`
	BaseAssetType      string    `db:"base_asset_type"`
	BaseAssetCode      string    `db:"base_asset_code"`
	BaseAssetIssuer    string    `db:"base_asset_issuer"`
	BaseAmount         xdr.Int64 `db:"base_amount"`
	CounterOfferID     *int64    `db:"counter_offer_id"`
	CounterAccount     string    `db:"counter_account"`
	CounterAssetType   string    `db:"counter_asset_type"`
	CounterAssetCode   string    `db:"counter_asset_code"`
	CounterAssetIssuer string    `db:"counter_asset_issuer"`
	CounterAmount      xdr.Int64 `db:"counter_amount"`
	BaseIsSeller       bool      `db:"base_is_seller"`
	PriceN             null.Int  `db:"price_n"`
	PriceD             null.Int  `db:"price_d"`
}

// TradesQ is a helper struct to aid in configuring queries that loads
// slices of trade structs.
type TradesQ struct {
	Err    error
	parent *Q
	sql    sq.SelectBuilder
}

// Transaction is a row of data from the `history_transactions` table
type Transaction struct {
	LedgerCloseTime time.Time `db:"ledger_close_time"`
	TransactionWithoutLedger
}

// TransactionsQ is a helper struct to aid in configuring queries that loads
// slices of transaction structs.
type TransactionsQ struct {
	Err           error
	parent        *Q
	sql           sq.SelectBuilder
	includeFailed bool
}

// TrustLine is row of data from the `trust_lines` table from horizon DB
type TrustLine struct {
	AccountID          string        `db:"account_id"`
	AssetType          xdr.AssetType `db:"asset_type"`
	AssetIssuer        string        `db:"asset_issuer"`
	AssetCode          string        `db:"asset_code"`
	Balance            int64         `db:"balance"`
	Limit              int64         `db:"trust_line_limit"`
	BuyingLiabilities  int64         `db:"buying_liabilities"`
	SellingLiabilities int64         `db:"selling_liabilities"`
	Flags              uint32        `db:"flags"`
	LastModifiedLedger uint32        `db:"last_modified_ledger"`
}

// QTrustLines defines trust lines related queries.
type QTrustLines interface {
	NewTrustLinesBatchInsertBuilder(maxBatchSize int) TrustLinesBatchInsertBuilder
	GetTrustLinesByKeys(keys []xdr.LedgerKeyTrustLine) ([]TrustLine, error)
	InsertTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error)
	UpdateTrustLine(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) (int64, error)
	UpsertTrustLines(trustLines []xdr.LedgerEntry) error
	RemoveTrustLine(key xdr.LedgerKeyTrustLine) (int64, error)
}

type TrustLinesBatchInsertBuilder interface {
	Add(trustLine xdr.TrustLineEntry, lastModifiedLedger xdr.Uint32) error
	Exec() error
}

// trustLinesBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type trustLinesBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func (q *Q) NewAccountsBatchInsertBuilder(maxBatchSize int) AccountsBatchInsertBuilder {
	return &accountsBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("accounts"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

func (q *Q) NewAccountSignersBatchInsertBuilder(maxBatchSize int) AccountSignersBatchInsertBuilder {
	return &accountSignersBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("accounts_signers"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

func (q *Q) NewAccountDataBatchInsertBuilder(maxBatchSize int) AccountDataBatchInsertBuilder {
	return &accountDataBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("accounts_data"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

func (q *Q) NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder {
	return &offersBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("offers"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

func (q *Q) NewTrustLinesBatchInsertBuilder(maxBatchSize int) TrustLinesBatchInsertBuilder {
	return &trustLinesBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("trust_lines"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// ElderLedger loads the oldest ledger known to the history database
func (q *Q) ElderLedger(dest interface{}) error {
	return q.GetRaw(dest, `SELECT COALESCE(MIN(sequence), 0) FROM history_ledgers`)
}

// GetLatestLedger loads the latest known ledger. Returns 0 if no ledgers in
// `history_ledgers` table.
func (q *Q) GetLatestLedger() (uint32, error) {
	var value uint32
	err := q.LatestLedger(&value)
	return value, err
}

// LatestLedger loads the latest known ledger
func (q *Q) LatestLedger(dest interface{}) error {
	return q.GetRaw(dest, `SELECT COALESCE(MAX(sequence), 0) FROM history_ledgers`)
}

// LatestLedgerBaseFeeAndSequence loads the latest known ledger's base fee and
// sequence number.
func (q *Q) LatestLedgerBaseFeeAndSequence(dest interface{}) error {
	return q.GetRaw(dest, `
		SELECT base_fee, sequence
		FROM history_ledgers
		WHERE sequence = (SELECT COALESCE(MAX(sequence), 0) FROM history_ledgers)
	`)
}

// OldestOutdatedLedgers populates a slice of ints with the first million
// outdated ledgers, based upon the provided `currentVersion` number
func (q *Q) OldestOutdatedLedgers(dest interface{}, currentVersion int) error {
	return q.SelectRaw(dest, `
		SELECT sequence
		FROM history_ledgers
		WHERE importer_version < $1
		ORDER BY sequence ASC
		LIMIT 1000000`, currentVersion)
}

// CloneIngestionQ clones underlying db.Session and returns IngestionQ
func (q *Q) CloneIngestionQ() IngestionQ {
	return &Q{q.Clone()}
}

// DeleteRangeAll deletes a range of rows from all history tables between
// `start` and `end` (exclusive).
func (q *Q) DeleteRangeAll(start, end int64) error {
	err := q.DeleteRange(start, end, "history_effects", "history_operation_id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_effects")
	}
	err = q.DeleteRange(start, end, "history_operation_participants", "history_operation_id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_operation_participants")
	}
	err = q.DeleteRange(start, end, "history_operations", "id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_operations")
	}
	err = q.DeleteRange(start, end, "history_transaction_participants", "history_transaction_id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_transaction_participants")
	}
	err = q.DeleteRange(start, end, "history_transactions", "id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_transactions")
	}
	err = q.DeleteRange(start, end, "history_ledgers", "id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_ledgers")
	}
	err = q.DeleteRange(start, end, "history_trades", "history_operation_id")
	if err != nil {
		return errors.Wrap(err, "Error clearing history_trades")
	}

	return nil
}
