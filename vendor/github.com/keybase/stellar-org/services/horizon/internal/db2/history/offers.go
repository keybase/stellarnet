package history

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (q *Q) CountOffers() (int, error) {
	sql := sq.Select("count(*)").From("offers")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

// GetOfferByID loads a row from the `offers` table, selected by offerid.
func (q *Q) GetOfferByID(id int64) (Offer, error) {
	var offer Offer
	sql := selectOffers.Where("offers.offerid = ?", id)
	err := q.Get(&offer, sql)
	return offer, err
}

// GetOffersByIDs loads a row from the `offers` table, selected by multiple offerid.
func (q *Q) GetOffersByIDs(ids []int64) ([]Offer, error) {
	var offers []Offer
	sql := selectOffers.Where(map[string]interface{}{"offers.offerid": ids})
	err := q.Select(&offers, sql)
	return offers, err
}

// GetOffers loads rows from `offers` by paging query.
func (q *Q) GetOffers(query OffersQuery) ([]Offer, error) {
	sql := selectOffers
	sql, err := query.PageQuery.ApplyTo(sql, "offers.offerid")

	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	if query.SellerID != "" {
		sql = sql.Where("offers.sellerid = ?", query.SellerID)
	}

	if query.Selling != nil {
		sellingAsset, err := xdr.MarshalBase64(*query.Selling)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal selling asset")
		}
		sql = sql.Where("offers.sellingasset = ?", sellingAsset)
	}

	if query.Buying != nil {
		buyingAsset, err := xdr.MarshalBase64(*query.Buying)
		if err != nil {
			return nil, errors.Wrap(err, "cannot marshal Buying asset")
		}
		sql = sql.Where("offers.buyingasset = ?", buyingAsset)
	}

	var offers []Offer
	if err := q.Select(&offers, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return offers, nil
}

// GetAllOffers loads a row from `history_accounts`, by address
func (q *Q) GetAllOffers() ([]Offer, error) {
	var offers []Offer
	err := q.Select(&offers, selectOffers)
	return offers, err
}

func offerToMap(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (map[string]interface{}, error) {
	var price float64
	if offer.Price.N > 0 {
		price = float64(offer.Price.N) / float64(offer.Price.D)
	} else if offer.Price.D == 0 {
		return nil, errors.New("offer price denominator is zero")
	}
	buyingAsset, err := xdr.MarshalBase64(offer.Buying)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal buying asset in offer")
	}
	sellingAsset, err := xdr.MarshalBase64(offer.Selling)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal selling asset in offer")
	}

	return map[string]interface{}{
		"sellerid":             offer.SellerId.Address(),
		"offerid":              offer.OfferId,
		"sellingasset":         sellingAsset,
		"buyingasset":          buyingAsset,
		"amount":               offer.Amount,
		"pricen":               offer.Price.N,
		"priced":               offer.Price.D,
		"price":                price,
		"flags":                offer.Flags,
		"last_modified_ledger": lastModifiedLedger,
	}, nil
}

// InsertOffer creates a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) InsertOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m, err := offerToMap(offer, lastModifiedLedger)
	if err != nil {
		return 0, err
	}

	sql := sq.Insert("offers").SetMap(m)
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// UpdateOffer updates a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) UpdateOffer(offer xdr.OfferEntry, lastModifiedLedger xdr.Uint32) (int64, error) {
	m, err := offerToMap(offer, lastModifiedLedger)
	if err != nil {
		return 0, err
	}

	offerID := m["offerid"]
	delete(m, "offerid")

	sql := sq.Update("offers").SetMap(m).Where(sq.Eq{"offerid": offerID})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// RemoveOffer deletes a row in the offers table.
// Returns number of rows affected and error.
func (q *Q) RemoveOffer(offerID xdr.Int64) (int64, error) {
	sql := sq.Delete("offers").Where(sq.Eq{"offerid": offerID})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

var selectOffers = sq.Select(`
	sellerid,
	offerid,
	sellingasset,
	buyingasset,
	amount,
	pricen,
	priced,
	price,
	flags,
	last_modified_ledger
`).From("offers")
