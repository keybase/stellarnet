package expingest

import (
	"database/sql"
	"time"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/verify"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	ilog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const verifyBatchSize = 50000

// stateVerifierExpectedIngestionVersion defines a version of ingestion system
// required by state verifier. This is done to prevent situations where
// ingestion has been updated with new features but state verifier does not
// check them.
// There is a test that checks it, to fix it: update the actual `verifyState`
// method instead of just updating this value!
const stateVerifierExpectedIngestionVersion = 4

// verifyState is called as a go routine from pipeline post hook every 64
// ledgers. It checks if the state is correct. If another go routine is already
// running it exists.
func (s *System) verifyState() error {
	s.stateVerificationMutex.Lock()
	if s.stateVerificationRunning {
		log.Warn("State verification is already running...")
		s.stateVerificationMutex.Unlock()
		return nil
	}
	s.stateVerificationRunning = true
	s.stateVerificationMutex.Unlock()

	if stateVerifierExpectedIngestionVersion != CurrentVersion {
		log.Errorf(
			"State verification expected version is %d but actual is: %d",
			stateVerifierExpectedIngestionVersion,
			CurrentVersion,
		)
		return nil
	}

	startTime := time.Now()
	session := s.historySession.Clone()

	defer func() {
		log.WithField("duration", time.Since(startTime).Seconds()).Info("State verification finished")
		session.Rollback()

		s.stateVerificationMutex.Lock()
		s.stateVerificationRunning = false
		s.stateVerificationMutex.Unlock()
	}()

	err := session.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return errors.Wrap(err, "Error starting transaction")
	}

	historyQ := &history.Q{session}

	// Ensure the ledger is a checkpoint ledger
	ledgerSequence, err := historyQ.GetLastLedgerExpIngestNonBlocking()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.GetLastLedgerExpIngestNonBlocking")
	}

	localLog := log.WithFields(ilog.F{
		"subservice": "state_verify",
		"ledger":     ledgerSequence,
	})

	if !historyarchive.IsCheckpoint(ledgerSequence) {
		localLog.Info("Current ledger is not a checkpoint ledger. Cancelling...")
		return nil
	}

	// Get root HAS to check if we're checking one of the latest ledgers or
	// Horizon is catching up. It doesn't make sense to verify old ledgers as
	// we want to check the latest state.
	archive := s.session.GetArchive()
	historyAdapter := adapters.MakeHistoryArchiveAdapter(archive)
	historyLatestSequence, err := historyAdapter.GetLatestLedgerSequence()
	if err != nil {
		return errors.Wrap(err, "Error getting the latest ledger sequence")
	}

	if ledgerSequence < historyLatestSequence {
		localLog.Info("Current ledger is old. Cancelling...")
		return nil
	}

	localLog.Info("Starting state verification. Waiting 40 seconds for stellar-core to publish HAS...")

	// Wait for stellar-core to publish HAS
	time.Sleep(40 * time.Second)

	localLog.Info("Creating state reader...")

	stateReader, err := io.MakeSingleLedgerStateReader(
		s.session.GetArchive(),
		&io.MemoryTempSet{},
		ledgerSequence,
	)
	if err != nil {
		return errors.Wrap(err, "Error running io.MakeSingleLedgerStateReader")
	}
	defer stateReader.Close()

	verifier := &verify.StateVerifier{
		StateReader:       stateReader,
		TransformFunction: transformEntry,
	}

	total := 0
	for {
		var keys []xdr.LedgerKey
		keys, err = verifier.GetLedgerKeys(verifyBatchSize)
		if err != nil {
			return errors.Wrap(err, "verifier.GetLedgerKeys")
		}

		if len(keys) == 0 {
			break
		}

		accounts := make([]string, 0, verifyBatchSize)
		offers := make([]int64, 0, verifyBatchSize)
		for _, key := range keys {
			switch key.Type {
			case xdr.LedgerEntryTypeAccount:
				accounts = append(accounts, key.Account.AccountId.Address())
			case xdr.LedgerEntryTypeOffer:
				offers = append(offers, int64(key.Offer.OfferId))
			default:
				return errors.New("GetLedgerKeys return unexpected type")
			}
		}

		err = addAccountsToStateVerifier(verifier, historyQ, accounts)
		if err != nil {
			return errors.Wrap(err, "addAccountsToStateVerifier failed")
		}

		err = addOffersToStateVerifier(verifier, historyQ, offers)
		if err != nil {
			return errors.Wrap(err, "addOffersToStateVerifier failed")
		}

		total += len(keys)
		localLog.WithField("total", total).Info("Batch added to StateVerifier")
	}

	localLog.WithField("total", total).Info("Finished writing to StateVerifier")

	countAccounts, err := historyQ.CountAccounts()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountAccounts")
	}

	countOffers, err := historyQ.CountOffers()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountOffers")
	}

	err = verifier.Verify(countAccounts + countOffers)
	if err != nil {
		return errors.Wrap(err, "verifier.Verify failed")
	}

	localLog.Info("State correct")
	return nil
}

func addAccountsToStateVerifier(verifier *verify.StateVerifier, q *history.Q, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	signers, err := q.SignersForAccounts(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.SignersForAccounts")
	}

	var account *xdr.AccountEntry
	for _, row := range signers {
		if account == nil || account.AccountId.Address() != row.Account {
			if account != nil {
				// Sort signers
				account.Signers = xdr.SortSignersByKey(account.Signers)

				entry := xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						Type:    xdr.LedgerEntryTypeAccount,
						Account: account,
					},
				}
				err = verifier.Write(entry)
				if err != nil {
					return err
				}
			}

			account = &xdr.AccountEntry{
				AccountId: xdr.MustAddress(row.Account),
				Signers:   []xdr.Signer{},
			}
		}

		if row.Account == row.Signer {
			// Master key
			account.Thresholds = [4]byte{
				// Store master weight only
				byte(row.Weight), 0, 0, 0,
			}
		} else {
			// Normal signer
			account.Signers = append(account.Signers, xdr.Signer{
				Key:    xdr.MustSigner(row.Signer),
				Weight: xdr.Uint32(row.Weight),
			})
		}
	}

	if account != nil {
		// Sort signers
		account.Signers = xdr.SortSignersByKey(account.Signers)

		// Add last created in a loop account
		entry := xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type:    xdr.LedgerEntryTypeAccount,
				Account: account,
			},
		}
		err = verifier.Write(entry)
		if err != nil {
			return err
		}
	}

	return nil
}

func addOffersToStateVerifier(verifier *verify.StateVerifier, q *history.Q, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	offers, err := q.GetOffersByIDs(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetOfferByIDs")
	}

	for _, row := range offers {
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeOffer,
				Offer: &xdr.OfferEntry{
					SellerId: xdr.MustAddress(row.SellerID),
					OfferId:  row.OfferID,
					Selling:  row.SellingAsset,
					Buying:   row.BuyingAsset,
					Amount:   row.Amount,
					Price: xdr.Price{
						N: xdr.Int32(row.Pricen),
						D: xdr.Int32(row.Priced),
					},
					Flags: xdr.Uint32(row.Flags),
				},
			},
		}
		err := verifier.Write(entry)
		if err != nil {
			return err
		}
	}

	return nil
}

func transformEntry(entry xdr.LedgerEntry) (bool, xdr.LedgerEntry) {
	switch entry.Data.Type {
	case xdr.LedgerEntryTypeAccount:
		accountEntry := entry.Data.Account

		// We don't store accounts with no signers and no master.
		// Ignore such accounts.
		if accountEntry.MasterKeyWeight() == 0 && len(accountEntry.Signers) == 0 {
			return true, xdr.LedgerEntry{}
		}

		// We store account id, master weight and signers only
		return false, xdr.LedgerEntry{
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: accountEntry.AccountId,
					Thresholds: [4]byte{
						// Store master weight only
						accountEntry.Thresholds[0], 0, 0, 0,
					},
					Signers: xdr.SortSignersByKey(accountEntry.Signers),
				},
			},
		}
	case xdr.LedgerEntryTypeOffer:
		// Full check of offer object
		return false, entry
	case xdr.LedgerEntryTypeTrustline:
		// Ignore
		return true, xdr.LedgerEntry{}
	case xdr.LedgerEntryTypeData:
		// Ignore
		return true, xdr.LedgerEntry{}
	default:
		panic("Invalid type")
	}
}
