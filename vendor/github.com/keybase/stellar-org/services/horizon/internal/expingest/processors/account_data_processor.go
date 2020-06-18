package processors

import (
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type AccountDataProcessor struct {
	dataQ history.QData

	cache *io.LedgerEntryChangeCache
}

func NewAccountDataProcessor(dataQ history.QData) *AccountDataProcessor {
	p := &AccountDataProcessor{dataQ: dataQ}
	p.reset()
	return p
}

func (p *AccountDataProcessor) reset() {
	p.cache = io.NewLedgerEntryChangeCache()
}

func (p *AccountDataProcessor) ProcessChange(change io.Change) error {
	// We're interested in data only
	if change.Type != xdr.LedgerEntryTypeData {
		return nil
	}

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

func (p *AccountDataProcessor) Commit() error {
	batch := p.dataQ.NewAccountDataBatchInsertBuilder(maxBatchSize)

	changes := p.cache.GetChanges()
	for _, change := range changes {
		var err error
		var rowsAffected int64
		var action string
		var ledgerKey xdr.LedgerKey

		switch {
		case change.Pre == nil && change.Post != nil:
			// Created
			action = "inserting"
			err = batch.Add(
				change.Post.Data.MustData(),
				change.Post.LastModifiedLedgerSeq,
			)
			rowsAffected = 1 // We don't track this when batch inserting
		case change.Pre != nil && change.Post == nil:
			// Removed
			action = "removing"
			data := change.Pre.Data.MustData()
			err = ledgerKey.SetData(data.AccountId, string(data.DataName))
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.dataQ.RemoveAccountData(*ledgerKey.Data)
		default:
			// Updated
			action = "updating"
			data := change.Post.Data.MustData()
			err = ledgerKey.SetData(data.AccountId, string(data.DataName))
			if err != nil {
				return errors.Wrap(err, "Error creating ledger key")
			}
			rowsAffected, err = p.dataQ.UpdateAccountData(data, change.Post.LastModifiedLedgerSeq)
		}

		if err != nil {
			return err
		}

		if rowsAffected != 1 {
			return ingesterrors.NewStateError(errors.Errorf(
				"%d rows affected when %s data: %s %s",
				rowsAffected,
				action,
				ledgerKey.Data.AccountId.Address(),
				ledgerKey.Data.DataName,
			))
		}
	}

	err := batch.Exec()
	if err != nil {
		return errors.Wrap(err, "error executing batch")
	}

	return nil
}
