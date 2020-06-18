// Package horizon contains the type definitions for all of horizon's
// response resources.
package horizon

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"

	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
)

// KeyTypeNames maps from strkey version bytes into json string values to use in
// horizon responses.
var KeyTypeNames = map[strkey.VersionByte]string{
	strkey.VersionByteAccountID: "ed25519_public_key",
	strkey.VersionByteSeed:      "ed25519_secret_seed",
	strkey.VersionByteHashX:     "sha256_hash",
	strkey.VersionByteHashTx:    "preauth_tx",
}

// Account is the summary of an account
type Account struct {
	Links struct {
		Self         hal.Link `json:"self"`
		Transactions hal.Link `json:"transactions"`
		Operations   hal.Link `json:"operations"`
		Payments     hal.Link `json:"payments"`
		Effects      hal.Link `json:"effects"`
		Offers       hal.Link `json:"offers"`
		Trades       hal.Link `json:"trades"`
		Data         hal.Link `json:"data"`
	} `json:"_links"`

	ID                   string            `json:"id"`
	AccountID            string            `json:"account_id"`
	Sequence             string            `json:"sequence"`
	SubentryCount        int32             `json:"subentry_count"`
	InflationDestination string            `json:"inflation_destination,omitempty"`
	HomeDomain           string            `json:"home_domain,omitempty"`
	LastModifiedLedger   uint32            `json:"last_modified_ledger"`
	LastModifiedTime     *time.Time        `json:"last_modified_time"`
	Thresholds           AccountThresholds `json:"thresholds"`
	Flags                AccountFlags      `json:"flags"`
	Balances             []Balance         `json:"balances"`
	Signers              []Signer          `json:"signers"`
	Data                 map[string]string `json:"data"`
	PT                   string            `json:"paging_token"`
}

// PagingToken implementation for hal.Pageable
func (res Account) PagingToken() string {
	return res.PT
}

// GetAccountID returns the Stellar account ID. This is to satisfy the
// Account interface of txnbuild.
func (a Account) GetAccountID() string {
	return a.AccountID
}

// GetNativeBalance returns the native balance of the account
func (a Account) GetNativeBalance() (string, error) {
	for _, balance := range a.Balances {
		if balance.Asset.Type == "native" {
			return balance.Balance, nil
		}
	}

	return "0", errors.New("account does not have a native balance")
}

// GetCreditBalance returns the balance for given code and issuer
func (a Account) GetCreditBalance(code string, issuer string) string {
	for _, balance := range a.Balances {
		if balance.Asset.Code == code && balance.Asset.Issuer == issuer {
			return balance.Balance
		}
	}

	return "0"
}

// GetSequenceNumber returns the sequence number of the account,
// and returns it as a 64-bit integer.
func (a Account) GetSequenceNumber() (int64, error) {
	seqNum, err := strconv.ParseInt(a.Sequence, 10, 64)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse account sequence number")
	}

	return seqNum, nil
}

// IncrementSequenceNumber increments the internal record of the account's sequence
// number by 1. This is typically used after a transaction build so that the next
// transaction to be built will be valid.
func (a *Account) IncrementSequenceNumber() (int64, error) {
	seqNum, err := a.GetSequenceNumber()
	if err != nil {
		return 0, err
	}
	seqNum++
	a.Sequence = strconv.FormatInt(int64(seqNum), 10)
	return seqNum, nil
}

// MustGetData returns decoded value for a given key. If the key does
// not exist, empty slice will be returned. If there is an error
// decoding a value, it will panic.
func (a *Account) MustGetData(key string) []byte {
	bytes, err := a.GetData(key)
	if err != nil {
		panic(err)
	}
	return bytes
}

// GetData returns decoded value for a given key. If the key does
// not exist, empty slice will be returned.
func (a *Account) GetData(key string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(a.Data[key])
}

// SignerSummary returns a map of signer's keys to weights.
func (a *Account) SignerSummary() map[string]int32 {
	m := map[string]int32{}
	for _, s := range a.Signers {
		m[s.Key] = s.Weight
	}
	return m
}

// AccountSigner is the account signer information.
type AccountSigner struct {
	Links struct {
		Account hal.Link `json:"account"`
	} `json:"_links"`

	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	PT        string `json:"paging_token"`
	Signer    `json:"signer"`
}

// PagingToken implementation for hal.Pageable
func (res AccountSigner) PagingToken() string {
	return res.PT
}

// AccountFlags represents the state of an account's flags
type AccountFlags struct {
	AuthRequired  bool `json:"auth_required"`
	AuthRevocable bool `json:"auth_revocable"`
	AuthImmutable bool `json:"auth_immutable"`
}

// AccountThresholds represents an accounts "thresholds", the numerical values
// needed to satisfy the authorization of a given operation.
type AccountThresholds struct {
	LowThreshold  byte `json:"low_threshold"`
	MedThreshold  byte `json:"med_threshold"`
	HighThreshold byte `json:"high_threshold"`
}

// Asset represents a single asset
type Asset base.Asset

// AssetStat represents the statistics for a single Asset
type AssetStat struct {
	Links struct {
		Toml hal.Link `json:"toml"`
	} `json:"_links"`

	base.Asset
	PT          string       `json:"paging_token"`
	Amount      string       `json:"amount"`
	NumAccounts int32        `json:"num_accounts"`
	Flags       AccountFlags `json:"flags"`
}

// PagingToken implementation for hal.Pageable
func (res AssetStat) PagingToken() string {
	return res.PT
}

// Balance represents an account's holdings for a single currency type
type Balance struct {
	Balance                           string `json:"balance"`
	Limit                             string `json:"limit,omitempty"`
	BuyingLiabilities                 string `json:"buying_liabilities"`
	SellingLiabilities                string `json:"selling_liabilities"`
	LastModifiedLedger                uint32 `json:"last_modified_ledger,omitempty"`
	IsAuthorized                      *bool  `json:"is_authorized,omitempty"`
	IsAuthorizedToMaintainLiabilities *bool  `json:"is_authorized_to_maintain_liabilities,omitempty"`
	base.Asset
}

// Ledger represents a single closed ledger
type Ledger struct {
	Links struct {
		Self         hal.Link `json:"self"`
		Transactions hal.Link `json:"transactions"`
		Operations   hal.Link `json:"operations"`
		Payments     hal.Link `json:"payments"`
		Effects      hal.Link `json:"effects"`
	} `json:"_links"`
	ID                         string    `json:"id"`
	PT                         string    `json:"paging_token"`
	Hash                       string    `json:"hash"`
	PrevHash                   string    `json:"prev_hash,omitempty"`
	Sequence                   int32     `json:"sequence"`
	SuccessfulTransactionCount int32     `json:"successful_transaction_count"`
	FailedTransactionCount     *int32    `json:"failed_transaction_count"`
	OperationCount             int32     `json:"operation_count"`
	ClosedAt                   time.Time `json:"closed_at"`
	TotalCoins                 string    `json:"total_coins"`
	FeePool                    string    `json:"fee_pool"`
	BaseFee                    int32     `json:"base_fee_in_stroops"`
	BaseReserve                int32     `json:"base_reserve_in_stroops"`
	MaxTxSetSize               int32     `json:"max_tx_set_size"`
	ProtocolVersion            int32     `json:"protocol_version"`
	HeaderXDR                  string    `json:"header_xdr"`
}

func (l Ledger) PagingToken() string {
	return l.PT
}

// Offer is the display form of an offer to trade currency.
type Offer struct {
	Links struct {
		Self       hal.Link `json:"self"`
		OfferMaker hal.Link `json:"offer_maker"`
	} `json:"_links"`

	ID                 int64      `json:"id,string"`
	PT                 string     `json:"paging_token"`
	Seller             string     `json:"seller"`
	Selling            Asset      `json:"selling"`
	Buying             Asset      `json:"buying"`
	Amount             string     `json:"amount"`
	PriceR             Price      `json:"price_r"`
	Price              string     `json:"price"`
	LastModifiedLedger int32      `json:"last_modified_ledger"`
	LastModifiedTime   *time.Time `json:"last_modified_time"`
}

func (o Offer) PagingToken() string {
	return o.PT
}

// OrderBookSummary represents a snapshot summary of a given order book
type OrderBookSummary struct {
	Bids    []PriceLevel `json:"bids"`
	Asks    []PriceLevel `json:"asks"`
	Selling Asset        `json:"base"`
	Buying  Asset        `json:"counter"`
}

// Path represents a single payment path.
type Path struct {
	SourceAssetType        string  `json:"source_asset_type"`
	SourceAssetCode        string  `json:"source_asset_code,omitempty"`
	SourceAssetIssuer      string  `json:"source_asset_issuer,omitempty"`
	SourceAmount           string  `json:"source_amount"`
	DestinationAssetType   string  `json:"destination_asset_type"`
	DestinationAssetCode   string  `json:"destination_asset_code,omitempty"`
	DestinationAssetIssuer string  `json:"destination_asset_issuer,omitempty"`
	DestinationAmount      string  `json:"destination_amount"`
	Path                   []Asset `json:"path"`
}

// stub implementation to satisfy pageable interface
func (p Path) PagingToken() string {
	return ""
}

// Price represents a price
type Price base.Price

// PriceLevel represents an aggregation of offers that share a given price
type PriceLevel struct {
	PriceR Price  `json:"price_r"`
	Price  string `json:"price"`
	Amount string `json:"amount"`
}

// Root is the initial map of links into the api.
type Root struct {
	Links struct {
		Account             hal.Link  `json:"account"`
		Accounts            *hal.Link `json:"accounts,omitempty"`
		AccountTransactions hal.Link  `json:"account_transactions"`
		Assets              hal.Link  `json:"assets"`
		Effects             hal.Link  `json:"effects"`
		FeeStats            hal.Link  `json:"fee_stats"`
		Friendbot           *hal.Link `json:"friendbot,omitempty"`
		Ledger              hal.Link  `json:"ledger"`
		Ledgers             hal.Link  `json:"ledgers"`
		Offer               *hal.Link `json:"offer,omitempty"`
		Offers              *hal.Link `json:"offers,omitempty"`
		Operation           hal.Link  `json:"operation"`
		Operations          hal.Link  `json:"operations"`
		OrderBook           hal.Link  `json:"order_book"`
		Payments            hal.Link  `json:"payments"`
		Self                hal.Link  `json:"self"`
		StrictReceivePaths  *hal.Link `json:"strict_receive_paths"`
		StrictSendPaths     *hal.Link `json:"strict_send_paths"`
		TradeAggregations   hal.Link  `json:"trade_aggregations"`
		Trades              hal.Link  `json:"trades"`
		Transaction         hal.Link  `json:"transaction"`
		Transactions        hal.Link  `json:"transactions"`
	} `json:"_links"`

	HorizonVersion               string `json:"horizon_version"`
	StellarCoreVersion           string `json:"core_version"`
	IngestSequence               uint32 `json:"ingest_latest_ledger"`
	HorizonSequence              int32  `json:"history_latest_ledger"`
	HistoryElderSequence         int32  `json:"history_elder_ledger"`
	CoreSequence                 int32  `json:"core_latest_ledger"`
	NetworkPassphrase            string `json:"network_passphrase"`
	CurrentProtocolVersion       int32  `json:"current_protocol_version"`
	CoreSupportedProtocolVersion int32  `json:"core_supported_protocol_version"`
}

// Signer represents one of an account's signers.
type Signer struct {
	Weight int32  `json:"weight"`
	Key    string `json:"key"`
	Type   string `json:"type"`
}

// Trade represents a horizon digested trade
type Trade struct {
	Links struct {
		Self      hal.Link `json:"self"`
		Base      hal.Link `json:"base"`
		Counter   hal.Link `json:"counter"`
		Operation hal.Link `json:"operation"`
	} `json:"_links"`

	ID                 string    `json:"id"`
	PT                 string    `json:"paging_token"`
	LedgerCloseTime    time.Time `json:"ledger_close_time"`
	OfferID            string    `json:"offer_id"`
	BaseOfferID        string    `json:"base_offer_id"`
	BaseAccount        string    `json:"base_account"`
	BaseAmount         string    `json:"base_amount"`
	BaseAssetType      string    `json:"base_asset_type"`
	BaseAssetCode      string    `json:"base_asset_code,omitempty"`
	BaseAssetIssuer    string    `json:"base_asset_issuer,omitempty"`
	CounterOfferID     string    `json:"counter_offer_id"`
	CounterAccount     string    `json:"counter_account"`
	CounterAmount      string    `json:"counter_amount"`
	CounterAssetType   string    `json:"counter_asset_type"`
	CounterAssetCode   string    `json:"counter_asset_code,omitempty"`
	CounterAssetIssuer string    `json:"counter_asset_issuer,omitempty"`
	BaseIsSeller       bool      `json:"base_is_seller"`
	Price              *Price    `json:"price"`
}

// PagingToken implementation for hal.Pageable
func (res Trade) PagingToken() string {
	return res.PT
}

// TradeEffect represents a trade effect resource.  NOTE (scott, 2017-12-08):
// this resource is being added back in temporarily to deal with a deploy snafu.
// I didn't properly message the community that we were changing the response
// format, and so we're adding this back in to allow transition.
type TradeEffect struct {
	Links struct {
		Self      hal.Link `json:"self"`
		Seller    hal.Link `json:"seller"`
		Buyer     hal.Link `json:"buyer"`
		Operation hal.Link `json:"operation"`
	} `json:"_links"`

	ID                string    `json:"id"`
	PT                string    `json:"paging_token"`
	OfferID           string    `json:"offer_id"`
	Seller            string    `json:"seller"`
	SoldAmount        string    `json:"sold_amount"`
	SoldAssetType     string    `json:"sold_asset_type"`
	SoldAssetCode     string    `json:"sold_asset_code,omitempty"`
	SoldAssetIssuer   string    `json:"sold_asset_issuer,omitempty"`
	Buyer             string    `json:"buyer"`
	BoughtAmount      string    `json:"bought_amount"`
	BoughtAssetType   string    `json:"bought_asset_type"`
	BoughtAssetCode   string    `json:"bought_asset_code,omitempty"`
	BoughtAssetIssuer string    `json:"bought_asset_issuer,omitempty"`
	LedgerCloseTime   time.Time `json:"created_at"`
}

// TradeAggregation represents trade data aggregation over a period of time
type TradeAggregation struct {
	Timestamp     int64     `json:"timestamp,string"`
	TradeCount    int64     `json:"trade_count,string"`
	BaseVolume    string    `json:"base_volume"`
	CounterVolume string    `json:"counter_volume"`
	Average       string    `json:"avg"`
	High          string    `json:"high"`
	HighR         xdr.Price `json:"high_r"`
	Low           string    `json:"low"`
	LowR          xdr.Price `json:"low_r"`
	Open          string    `json:"open"`
	OpenR         xdr.Price `json:"open_r"`
	Close         string    `json:"close"`
	CloseR        xdr.Price `json:"close_r"`
}

// PagingToken implementation for hal.Pageable. Not actually used
func (res TradeAggregation) PagingToken() string {
	return string(res.Timestamp)
}

// Transaction represents a single, successful transaction
type Transaction struct {
	Links struct {
		Self       hal.Link `json:"self"`
		Account    hal.Link `json:"account"`
		Ledger     hal.Link `json:"ledger"`
		Operations hal.Link `json:"operations"`
		Effects    hal.Link `json:"effects"`
		Precedes   hal.Link `json:"precedes"`
		Succeeds   hal.Link `json:"succeeds"`
		// Temporarily include Transaction as a link so that Transaction
		// can be fully compatible with TransactionSuccess
		// When TransactionSuccess is removed from the SDKs we can remove this HAL link
		Transaction hal.Link `json:"transaction"`
	} `json:"_links"`
	ID                 string              `json:"id"`
	PT                 string              `json:"paging_token"`
	Successful         bool                `json:"successful"`
	Hash               string              `json:"hash"`
	Ledger             int32               `json:"ledger"`
	LedgerCloseTime    time.Time           `json:"created_at"`
	Account            string              `json:"source_account"`
	AccountSequence    string              `json:"source_account_sequence"`
	FeeAccount         string              `json:"fee_account"`
	FeeCharged         int64               `json:"fee_charged,string"`
	MaxFee             int64               `json:"max_fee,string"`
	OperationCount     int32               `json:"operation_count"`
	EnvelopeXdr        string              `json:"envelope_xdr"`
	ResultXdr          string              `json:"result_xdr"`
	ResultMetaXdr      string              `json:"result_meta_xdr"`
	FeeMetaXdr         string              `json:"fee_meta_xdr"`
	MemoType           string              `json:"memo_type"`
	MemoBytes          string              `json:"memo_bytes,omitempty"`
	Memo               string              `json:"memo,omitempty"`
	Signatures         []string            `json:"signatures"`
	ValidAfter         string              `json:"valid_after,omitempty"`
	ValidBefore        string              `json:"valid_before,omitempty"`
	FeeBumpTransaction *FeeBumpTransaction `json:"fee_bump_transaction,omitempty"`
	InnerTransaction   *InnerTransaction   `json:"inner_transaction,omitempty"`
}

// FeeBumpTransaction contains information about a fee bump transaction
type FeeBumpTransaction struct {
	Hash       string   `json:"hash"`
	Signatures []string `json:"signatures"`
}

// InnerTransaction contains information about the inner transaction contained
// within a fee bump transaction
type InnerTransaction struct {
	Hash       string   `json:"hash"`
	Signatures []string `json:"signatures"`
	MaxFee     int64    `json:"max_fee,string"`
}

// MarshalJSON implements a custom marshaler for Transaction.
// The memo field should be omitted if and only if the
// memo_type is "none".
func (t Transaction) MarshalJSON() ([]byte, error) {
	type Alias Transaction
	v := &struct {
		Memo      *string `json:"memo,omitempty"`
		MemoBytes *string `json:"memo_bytes,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(&t),
	}
	if t.MemoType != "none" {
		v.Memo = &t.Memo
	}

	if t.MemoType == "text" {
		v.MemoBytes = &t.MemoBytes
	}

	return json.Marshal(v)
}

// UnmarshalJSON implements a custom unmarshaler for Transaction
// which can handle a max_fee field which can be a string of int
func (t *Transaction) UnmarshalJSON(data []byte) error {
	type Alias Transaction // we define Alias to avoid infinite recursion when calling UnmarshalJSON()
	v := &struct {
		FeeCharged json.Number `json:"fee_charged"`
		MaxFee     json.Number `json:"max_fee"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}

	t.FeeCharged, err = v.FeeCharged.Int64()
	if err != nil {
		return err
	}
	t.MaxFee, err = v.MaxFee.Int64()
	if err != nil {
		return err
	}
	return nil
}

// PagingToken implementation for hal.Pageable
func (t Transaction) PagingToken() string {
	return t.PT
}

// TransactionResultCodes represent a summary of result codes returned from
// a single xdr TransactionResult
type TransactionResultCodes struct {
	TransactionCode string   `json:"transaction"`
	OperationCodes  []string `json:"operations,omitempty"`
}

// KeyTypeFromAddress converts the version byte of the provided strkey encoded
// value (for example an account id or a signer key) and returns the appropriate
// horizon-specific type name.
func KeyTypeFromAddress(address string) (string, error) {
	vb, err := strkey.Version(address)
	if err != nil {
		return "", errors.Wrap(err, "invalid address")
	}

	result, ok := KeyTypeNames[vb]
	if !ok {
		result = "unknown"
	}

	return result, nil
}

// MustKeyTypeFromAddress is the panicking variant of KeyTypeFromAddress.
func MustKeyTypeFromAddress(address string) string {
	ret, err := KeyTypeFromAddress(address)
	if err != nil {
		panic(err)
	}

	return ret
}

// AccountData represents a single data object stored on by an account
type AccountData struct {
	Value string `json:"value"`
}

// AccountsPage returns a list of account records
type AccountsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Account `json:"records"`
	} `json:"_embedded"`
}

// TradeAggregationsPage returns a list of aggregated trade records, aggregated by resolution
type TradeAggregationsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []TradeAggregation `json:"records"`
	} `json:"_embedded"`
}

// TradesPage returns a list of trade records
type TradesPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Trade `json:"records"`
	} `json:"_embedded"`
}

// OffersPage returns a list of offers
type OffersPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Offer `json:"records"`
	} `json:"_embedded"`
}

// AssetsPage contains page of assets returned by Horizon.
type AssetsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []AssetStat
	} `json:"_embedded"`
}

// LedgersPage contains page of ledger information returned by Horizon
type LedgersPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Ledger
	} `json:"_embedded"`
}

type FeeDistribution struct {
	Max  int64 `json:"max,string"`
	Min  int64 `json:"min,string"`
	Mode int64 `json:"mode,string"`
	P10  int64 `json:"p10,string"`
	P20  int64 `json:"p20,string"`
	P30  int64 `json:"p30,string"`
	P40  int64 `json:"p40,string"`
	P50  int64 `json:"p50,string"`
	P60  int64 `json:"p60,string"`
	P70  int64 `json:"p70,string"`
	P80  int64 `json:"p80,string"`
	P90  int64 `json:"p90,string"`
	P95  int64 `json:"p95,string"`
	P99  int64 `json:"p99,string"`
}

// FeeStats represents a response of fees from horizon
// To do: implement fee suggestions if agreement is reached in https://github.com/stellar/go/issues/926
type FeeStats struct {
	LastLedger          uint32  `json:"last_ledger,string"`
	LastLedgerBaseFee   int64   `json:"last_ledger_base_fee,string"`
	LedgerCapacityUsage float64 `json:"ledger_capacity_usage,string"`

	FeeCharged FeeDistribution `json:"fee_charged"`
	MaxFee     FeeDistribution `json:"max_fee"`
}

// TransactionsPage contains records of transaction information returned by Horizon
type TransactionsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Transaction
	} `json:"_embedded"`
}

// PathsPage contains records of payment paths found by horizon
type PathsPage struct {
	Links    hal.Links `json:"_links"`
	Embedded struct {
		Records []Path
	} `json:"_embedded"`
}
