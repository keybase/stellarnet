package history

import (
	"time"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// InsertTrade represents the arguments to TradeBatchInsertBuilder.Add() which is used to insert
// rows into the history_trades table
type InsertTrade struct {
	HistoryOperationID int64
	Order              int32
	LedgerCloseTime    time.Time
	BuyOfferExists     bool
	BuyOfferID         int64
	SellerAccountID    int64
	BuyerAccountID     int64
	SoldAssetID        int64
	BoughtAssetID      int64
	Trade              xdr.ClaimOfferAtom
	SellPrice          xdr.Price
}

// TradeBatchInsertBuilder is used to insert trades into the
// history_trades table
type TradeBatchInsertBuilder interface {
	Add(entries ...InsertTrade) error
	Exec() error
}

// tradeBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type tradeBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewTradeBatchInsertBuilder constructs a new TradeBatchInsertBuilder instance
func (q *Q) NewTradeBatchInsertBuilder(maxBatchSize int) TradeBatchInsertBuilder {
	return &tradeBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_trades"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Exec flushes all outstanding trades to the database
func (i *tradeBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}

// Add adds a new trade to the batch
func (i *tradeBatchInsertBuilder) Add(entries ...InsertTrade) error {
	for _, entry := range entries {
		sellOfferID := EncodeOfferId(uint64(entry.Trade.OfferId), CoreOfferIDType)

		// if the buy offer exists, encode the stellar core generated id as the offer id
		// if not, encode the toid as the offer id
		var buyOfferID int64
		if entry.BuyOfferExists {
			buyOfferID = EncodeOfferId(uint64(entry.BuyOfferID), CoreOfferIDType)
		} else {
			buyOfferID = EncodeOfferId(uint64(entry.HistoryOperationID), TOIDType)
		}

		orderPreserved, baseAssetID, counterAssetID := getCanonicalAssetOrder(
			entry.SoldAssetID, entry.BoughtAssetID,
		)

		var baseAccountID, counterAccountID int64
		var baseAmount, counterAmount xdr.Int64
		var baseOfferID, counterOfferID int64

		if orderPreserved {
			baseAccountID = entry.SellerAccountID
			baseAmount = entry.Trade.AmountSold
			counterAccountID = entry.BuyerAccountID
			counterAmount = entry.Trade.AmountBought
			baseOfferID = sellOfferID
			counterOfferID = buyOfferID
		} else {
			baseAccountID = entry.BuyerAccountID
			baseAmount = entry.Trade.AmountBought
			counterAccountID = entry.SellerAccountID
			counterAmount = entry.Trade.AmountSold
			baseOfferID = buyOfferID
			counterOfferID = sellOfferID
			entry.SellPrice.Invert()
		}

		err := i.builder.Row(map[string]interface{}{
			"history_operation_id": entry.HistoryOperationID,
			"\"order\"":            entry.Order,
			"ledger_closed_at":     entry.LedgerCloseTime,
			"offer_id":             entry.Trade.OfferId,
			"base_offer_id":        baseOfferID,
			"base_account_id":      baseAccountID,
			"base_asset_id":        baseAssetID,
			"base_amount":          baseAmount,
			"counter_offer_id":     counterOfferID,
			"counter_account_id":   counterAccountID,
			"counter_asset_id":     counterAssetID,
			"counter_amount":       counterAmount,
			"base_is_seller":       orderPreserved,
			"price_n":              entry.SellPrice.N,
			"price_d":              entry.SellPrice.D,
		})
		if err != nil {
			return errors.Wrap(err, "failed to add trade")
		}
	}

	return nil
}
