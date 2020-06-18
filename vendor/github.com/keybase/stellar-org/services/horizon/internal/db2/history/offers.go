package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// QOffers defines offer related queries.
type QOffers interface {
	GetAllOffers() ([]Offer, error)
	GetOffersByIDs(ids []int64) ([]Offer, error)
	CountOffers() (int, error)
	GetUpdatedOffers(newerThanSequence uint32) ([]Offer, error)
	NewOffersBatchInsertBuilder(maxBatchSize int) OffersBatchInsertBuilder
	UpdateOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (int64, error)
	RemoveOffer(offerID xdr.Int64, lastModifiedLedger uint32) (int64, error)
	CompactOffers(cutOffSequence uint32) (int64, error)
}

func (q *Q) CountOffers() (int, error) {
	sql := sq.Select("count(*)").Where("deleted = ?", false).From("offers")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetOfferByID loads a row from the `offers` table, selected by offerid.
func (q *Q) GetOfferByID(id int64) (Offer, error) {
	var offer Offer
	sql := selectOffers.Where("deleted = ?", false).
		Where("offers.offer_id = ?", id)
	err := q.Get(&offer, sql)
	return offer, err
}

// GetOffersByIDs loads a row from the `offers` table, selected by multiple offerid.
func (q *Q) GetOffersByIDs(ids []int64) ([]Offer, error) {
	var offers []Offer
	sql := selectOffers.Where("deleted = ?", false).
		Where(map[string]interface{}{"offers.offer_id": ids})
	err := q.Select(&offers, sql)
	return offers, err
}

// GetOffers loads rows from `offers` by paging query.
func (q *Q) GetOffers(query OffersQuery) ([]Offer, error) {
	sql := selectOffers.Where("deleted = ?", false)
	sql, err := query.PageQuery.ApplyTo(sql, "offers.offer_id")

	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	if query.SellerID != "" {
		sql = sql.Where("offers.seller_id = ?", query.SellerID)
	}

	if query.Selling != nil {
		sellingAsset, err := xdr.MarshalBase64(*query.Selling)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal selling asset")
		}
		sql = sql.Where("offers.selling_asset = ?", sellingAsset)
	}

	if query.Buying != nil {
		buyingAsset, err := xdr.MarshalBase64(*query.Buying)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal Buying asset")
		}
		sql = sql.Where("offers.buying_asset = ?", buyingAsset)
	}

	var offers []Offer
	if err := q.Select(&offers, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return offers, nil
}

// GetAllOffers loads all non deleted offers
func (q *Q) GetAllOffers() ([]Offer, error) {
	var offers []Offer
	err := q.Select(&offers, selectOffers.Where("deleted = ?", false))
	return offers, err
}

// GetUpdatedOffers returns all offers created, updated, or deleted after the given ledger sequence.
func (q *Q) GetUpdatedOffers(newerThanSequence uint32) ([]Offer, error) {
	var offers []Offer
	err := q.Select(&offers, selectOffers.Where("offers.last_modified_ledger > ?", newerThanSequence))
	return offers, err
}

// UpdateOffer updates a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) UpdateOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	var price float64
	if offer.Price.N > 0 {
		price = float64(offer.Price.N) / float64(offer.Price.D)
	} else if offer.Price.D == 0 {
		return 0, errors.New("offer price denominator is zero")
	}

	offerMap := map[string]interface{}{
		"seller_id":            offer.SellerId.Address(),
		"selling_asset":        offer.Selling,
		"buying_asset":         offer.Buying,
		"amount":               offer.Amount,
		"pricen":               offer.Price.N,
		"priced":               offer.Price.D,
		"price":                price,
		"flags":                offer.Flags,
		"last_modified_ledger": lastModifiedLedger,
	}

	sql := sq.Update("offers").SetMap(offerMap).Where("offer_id = ?", offer.OfferId)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveOffer marks a row in the offers table as deleted.
// Returns number of rows affected and error.
func (q *Q) RemoveOffer(offerID xdr.Int64, lastModifiedLedger uint32) (int64, error) {
	sql := sq.Update("offers").
		Set("deleted", true).
		Set("last_modified_ledger", lastModifiedLedger).
		Where("offer_id = ?", offerID)

	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// CompactOffers removes rows from the offers table which are marked for deletion.
func (q *Q) CompactOffers(cutOffSequence uint32) (int64, error) {
	sql := sq.Delete("offers").
		Where("deleted = ?", true).
		Where("last_modified_ledger <= ?", cutOffSequence)

	result, err := q.Exec(sql)
	if err != nil {
		return 0, errors.Wrap(err, "cannot delete offer rows")
	}

	if err = q.UpdateOfferCompactionSequence(cutOffSequence); err != nil {
		return 0, errors.Wrap(err, "cannot update offer compaction sequence")
	}

	return result.RowsAffected()
}

var selectOffers = sq.Select(`
	seller_id,
	offer_id,
	selling_asset,
	buying_asset,
	amount,
	pricen,
	priced,
	price,
	flags,
	deleted,
	last_modified_ledger
`).From("offers")
