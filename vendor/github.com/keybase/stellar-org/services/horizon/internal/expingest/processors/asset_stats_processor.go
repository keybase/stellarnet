package processors

import (
	"database/sql"
	"math/big"

	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type AssetStatsProcessor struct {
	assetStatsQ history.QAssetStats

	cache               *io.LedgerEntryChangeCache
	assetStatSet        AssetStatSet
	useLedgerEntryCache bool
}

// NewAssetStatsProcessor constructs a new AssetStatsProcessor instance.
// If useLedgerEntryCache is false we don't use ledger cache and we just
// add trust lines to assetStatSet, then we insert all the stats in one
// insert query. This is done to make history buckets processing faster
// (batch inserting).
func NewAssetStatsProcessor(
	assetStatsQ history.QAssetStats,
	useLedgerEntryCache bool,
) *AssetStatsProcessor {
	p := &AssetStatsProcessor{
		assetStatsQ:         assetStatsQ,
		useLedgerEntryCache: useLedgerEntryCache,
	}
	p.reset()
	return p
}

func (p *AssetStatsProcessor) reset() {
	p.cache = io.NewLedgerEntryChangeCache()
	p.assetStatSet = AssetStatSet{}
}

func (p *AssetStatsProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeTrustline {
		return nil
	}

	if p.useLedgerEntryCache {
		err := p.cache.AddChange(change)
		if err != nil {
			return errors.Wrap(err, "error adding to ledgerCache")
		}

		if p.cache.Size() > maxBatchSize {
			err = p.Commit()
			if err != nil {
				return errors.Wrap(err, "error in Commit")
			}
			p.reset()
		}
		return nil
	}

	if !(change.Pre == nil && change.Post != nil) {
		return errors.New("AssetStatsProcessor is in insert only mode")
	}

	postTrustLine := change.Post.Data.MustTrustLine()
	err := p.adjustAssetStat(nil, &postTrustLine)
	if err != nil {
		return errors.Wrap(err, "Error adjusting asset stat")
	}

	return nil
}

func (p *AssetStatsProcessor) Commit() error {
	if !p.useLedgerEntryCache {
		return p.assetStatsQ.InsertAssetStats(p.assetStatSet.All(), maxBatchSize)
	}

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var err error

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			postTrustLine := change.Post.Data.MustTrustLine()
			err = p.adjustAssetStat(nil, &postTrustLine)
		case change.Pre != nil && change.Post != nil:
			// Updated
			preTrustLine := change.Pre.Data.MustTrustLine()
			postTrustLine := change.Post.Data.MustTrustLine()
			err = p.adjustAssetStat(&preTrustLine, &postTrustLine)
		case change.Pre != nil && change.Post == nil:
			// Removed
			preTrustLine := change.Pre.Data.MustTrustLine()
			err = p.adjustAssetStat(&preTrustLine, nil)
		default:
			return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
		}

		if err != nil {
			return errors.Wrap(err, "Error adjusting asset stat")
		}
	}

	assetStatsDeltas := p.assetStatSet.All()
	for _, delta := range assetStatsDeltas {
		var rowsAffected int64

		stat, err := p.assetStatsQ.GetAssetStat(
			delta.AssetType,
			delta.AssetCode,
			delta.AssetIssuer,
		)
		assetStatNotFound := err == sql.ErrNoRows
		if !assetStatNotFound && err != nil {
			return errors.Wrap(err, "could not fetch asset stat from db")
		}

		if assetStatNotFound {
			// Insert
			if delta.NumAccounts < 0 {
				return ingesterrors.NewStateError(errors.Errorf(
					"NumAccounts negative but DB entry does not exist for asset: %s %s %s",
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				))
			}

			var errInsert error
			rowsAffected, errInsert = p.assetStatsQ.InsertAssetStat(delta)
			if errInsert != nil {
				return errors.Wrap(errInsert, "could not insert asset stat")
			}
		} else {
			statBalance, ok := new(big.Int).SetString(stat.Amount, 10)
			if !ok {
				return errors.New("Error parsing: " + stat.Amount)
			}

			deltaBalance, ok := new(big.Int).SetString(delta.Amount, 10)
			if !ok {
				return errors.New("Error parsing: " + stat.Amount)
			}

			// statBalance = statBalance + deltaBalance
			statBalance.Add(statBalance, deltaBalance)
			statAccounts := stat.NumAccounts + delta.NumAccounts

			if statAccounts == 0 {
				// Remove stats
				if statBalance.Cmp(big.NewInt(0)) != 0 {
					return ingesterrors.NewStateError(errors.Errorf(
						"Removing asset stat by final amount non-zero for: %s %s %s",
						delta.AssetType,
						delta.AssetCode,
						delta.AssetIssuer,
					))
				}
				rowsAffected, err = p.assetStatsQ.RemoveAssetStat(
					delta.AssetType,
					delta.AssetCode,
					delta.AssetIssuer,
				)
				if err != nil {
					return errors.Wrap(err, "could not remove asset stat")
				}
			} else {
				// Update
				rowsAffected, err = p.assetStatsQ.UpdateAssetStat(history.ExpAssetStat{
					AssetType:   delta.AssetType,
					AssetCode:   delta.AssetCode,
					AssetIssuer: delta.AssetIssuer,
					Amount:      statBalance.String(),
					NumAccounts: statAccounts,
				})
				if err != nil {
					return errors.Wrap(err, "could not update asset stat")
				}
			}
		}

		if rowsAffected != 1 {
			return ingesterrors.NewStateError(errors.Errorf(
				"%d rows affected when adjusting asset stat for asset: %s %s %s",
				rowsAffected,
				delta.AssetType,
				delta.AssetCode,
				delta.AssetIssuer,
			))
		}
	}

	return nil
}

func (p *AssetStatsProcessor) adjustAssetStat(
	preTrustline *xdr.TrustLineEntry,
	postTrustline *xdr.TrustLineEntry,
) error {
	var deltaBalance xdr.Int64
	var deltaAccounts int32
	var trustline xdr.TrustLineEntry

	if preTrustline != nil && postTrustline == nil {
		trustline = *preTrustline
		// removing a trustline
		if xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() {
			deltaAccounts = -1
			deltaBalance = -preTrustline.Balance
		}
	} else if preTrustline == nil && postTrustline != nil {
		trustline = *postTrustline
		// adding a trustline
		if xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			deltaAccounts = 1
			deltaBalance = postTrustline.Balance
		}
	} else if preTrustline != nil && postTrustline != nil {
		trustline = *postTrustline
		// updating a trustline
		if xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() &&
			xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			// trustline remains authorized
			deltaAccounts = 0
			deltaBalance = postTrustline.Balance - preTrustline.Balance
		} else if xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() &&
			!xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			// trustline was authorized and became unauthorized
			deltaAccounts = -1
			deltaBalance = -preTrustline.Balance
		} else if !xdr.TrustLineFlags(preTrustline.Flags).IsAuthorized() &&
			xdr.TrustLineFlags(postTrustline.Flags).IsAuthorized() {
			// trustline was unauthorized and became authorized
			deltaAccounts = 1
			deltaBalance = postTrustline.Balance
		}
		// else, trustline was unauthorized and remains unauthorized
		// so there is no change to accounts or balances
	} else {
		return ingesterrors.NewStateError(errors.New("both pre and post trustlines cannot be nil"))
	}

	err := p.assetStatSet.AddDelta(trustline.Asset, int64(deltaBalance), deltaAccounts)
	if err != nil {
		return errors.Wrap(err, "error running AssetStatSet.AddDelta")
	}
	return nil
}
