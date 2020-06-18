package processors

import (
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type AccountsProcessor struct {
	accountsQ history.QAccounts

	cache *io.LedgerEntryChangeCache
}

func NewAccountsProcessor(accountsQ history.QAccounts) *AccountsProcessor {
	p := &AccountsProcessor{accountsQ: accountsQ}
	p.reset()
	return p
}

func (p *AccountsProcessor) reset() {
	p.cache = io.NewLedgerEntryChangeCache()
}

func (p *AccountsProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeAccount {
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

func (p *AccountsProcessor) Commit() error {
	batchUpsertAccounts := []xdr.LedgerEntry{}

	changes := p.cache.GetChanges()
	for _, change := range changes {
		changed, err := change.AccountChangedExceptSigners()
		if err != nil {
			return errors.Wrap(err, "Error running change.AccountChangedExceptSigners")
		}

		if !changed {
			continue
		}

		switch {
		case change.Post != nil:
			// Created and updated
			batchUpsertAccounts = append(batchUpsertAccounts, *change.Post)
		case change.Pre != nil && change.Post == nil:
			// Removed
			account := change.Pre.Data.MustAccount()
			accountID := account.AccountId.Address()
			rowsAffected, err := p.accountsQ.RemoveAccount(accountID)

			if err != nil {
				return err
			}

			if rowsAffected != 1 {
				return ingesterrors.NewStateError(errors.Errorf(
					"%d No rows affected when removing account %s",
					rowsAffected,
					accountID,
				))
			}
		default:
			return errors.New("Invalid io.Change: change.Pre == nil && change.Post == nil")
		}
	}

	// Upsert accounts
	if len(batchUpsertAccounts) > 0 {
		err := p.accountsQ.UpsertAccounts(batchUpsertAccounts)
		if err != nil {
			return errors.Wrap(err, "errors in UpsertAccounts")
		}
	}

	return nil
}
