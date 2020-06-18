package processors

import (
	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type SignersProcessor struct {
	signersQ history.QSigners

	cache *io.LedgerEntryChangeCache
	batch history.AccountSignersBatchInsertBuilder
	// insertOnlyMode is a mode in which we don't use ledger cache and we just
	// add signers to a batch, then we Exec all signers in one insert query.
	// This is done to make history buckets processing faster (batch inserting).
	useLedgerEntryCache bool
}

func NewSignersProcessor(
	signersQ history.QSigners, useLedgerEntryCache bool,
) *SignersProcessor {
	p := &SignersProcessor{signersQ: signersQ, useLedgerEntryCache: useLedgerEntryCache}
	p.reset()
	return p
}

func (p *SignersProcessor) reset() {
	p.batch = p.signersQ.NewAccountSignersBatchInsertBuilder(maxBatchSize)
	p.cache = io.NewLedgerEntryChangeCache()
}

func (p *SignersProcessor) ProcessChange(change io.Change) error {
	if change.Type != xdr.LedgerEntryTypeAccount {
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
		return errors.New("AssetStatsProSignersProcessorcessor is in insert only mode")
	}

	accountEntry := change.Post.Data.MustAccount()
	account := accountEntry.AccountId.Address()

	for signer, weight := range accountEntry.SignerSummary() {
		err := p.batch.Add(history.AccountSigner{
			Account: account,
			Signer:  signer,
			Weight:  weight,
		})
		if err != nil {
			return errors.Wrap(err, "Error adding row to accountSignerBatch")
		}
	}

	return nil
}

func (p *SignersProcessor) Commit() error {
	if !p.useLedgerEntryCache {
		return p.batch.Exec()
	}

	changes := p.cache.GetChanges()
	for _, change := range changes {
		if !change.AccountSignersChanged() {
			continue
		}

		// The code below removes all Pre signers adds Post signers but
		// can be improved by finding a diff (check performance first).
		if change.Pre != nil {
			preAccountEntry := change.Pre.Data.MustAccount()
			for signer := range preAccountEntry.SignerSummary() {
				rowsAffected, err := p.signersQ.RemoveAccountSigner(preAccountEntry.AccountId.Address(), signer)
				if err != nil {
					return errors.Wrap(err, "Error removing a signer")
				}

				if rowsAffected != 1 {
					return ingesterrors.NewStateError(errors.Errorf(
						"Expected account=%s signer=%s in database but not found when removing (rows affected = %d)",
						preAccountEntry.AccountId.Address(),
						signer,
						rowsAffected,
					))
				}
			}
		}

		if change.Post != nil {
			postAccountEntry := change.Post.Data.MustAccount()
			for signer, weight := range postAccountEntry.SignerSummary() {
				rowsAffected, err := p.signersQ.CreateAccountSigner(postAccountEntry.AccountId.Address(), signer, weight)
				if err != nil {
					return errors.Wrap(err, "Error inserting a signer")
				}

				if rowsAffected != 1 {
					return ingesterrors.NewStateError(errors.Errorf(
						"%d rows affected when inserting account=%s signer=%s to database",
						rowsAffected,
						postAccountEntry.AccountId.Address(),
						signer,
					))
				}
			}
		}
	}

	return nil
}
