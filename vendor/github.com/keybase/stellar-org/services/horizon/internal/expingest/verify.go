package expingest

import (
	"database/sql"
	"fmt"
	"time"

	ingesterrors "github.com/stellar/go/exp/ingest/errors"
	"github.com/stellar/go/exp/ingest/verify"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const verifyBatchSize = 50000
const assetStatsBatchSize = 500

// stateVerifierExpectedIngestionVersion defines a version of ingestion system
// required by state verifier. This is done to prevent situations where
// ingestion has been updated with new features but state verifier does not
// check them.
// There is a test that checks it, to fix it: update the actual `verifyState`
// method instead of just updating this value!
const stateVerifierExpectedIngestionVersion = 10

// verifyState is called as a go routine from pipeline post hook every 64
// ledgers. It checks if the state is correct. If another go routine is already
// running it exits.
func (s *System) verifyState(verifyAgainstLatestCheckpoint bool) error {
	s.stateVerificationMutex.Lock()
	if s.stateVerificationRunning {
		log.Warn("State verification is already running...")
		s.stateVerificationMutex.Unlock()
		return nil
	}
	s.stateVerificationRunning = true
	s.stateVerificationMutex.Unlock()

	updateMetrics := false

	if stateVerifierExpectedIngestionVersion != CurrentVersion {
		log.Errorf(
			"State verification expected version is %d but actual is: %d",
			stateVerifierExpectedIngestionVersion,
			CurrentVersion,
		)
		return nil
	}

	startTime := time.Now()
	historyQ := s.historyQ.CloneIngestionQ()

	defer func() {
		duration := time.Since(startTime)
		if updateMetrics {
			s.Metrics.StateVerifyTimer.Update(duration)
		}
		log.WithField("duration", duration.Seconds()).Info("State verification finished")
		historyQ.Rollback()
		s.stateVerificationMutex.Lock()
		s.stateVerificationRunning = false
		s.stateVerificationMutex.Unlock()
	}()

	err := historyQ.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return errors.Wrap(err, "Error starting transaction")
	}

	// Ensure the ledger is a checkpoint ledger
	ledgerSequence, err := historyQ.GetLastLedgerExpIngestNonBlocking()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.GetLastLedgerExpIngestNonBlocking")
	}

	localLog := log.WithFields(logpkg.F{
		"subservice": "state_verify",
		"ledger":     ledgerSequence,
	})

	if !historyarchive.IsCheckpoint(ledgerSequence) {
		localLog.Info("Current ledger is not a checkpoint ledger. Cancelling...")
		return nil
	}

	if verifyAgainstLatestCheckpoint {
		// Get root HAS to check if we're checking one of the latest ledgers or
		// Horizon is catching up. It doesn't make sense to verify old ledgers as
		// we want to check the latest state.
		var historyLatestSequence uint32
		historyLatestSequence, err = s.historyAdapter.GetLatestLedgerSequence()
		if err != nil {
			return errors.Wrap(err, "Error getting the latest ledger sequence")
		}

		if ledgerSequence < historyLatestSequence {
			localLog.Info("Current ledger is old. Cancelling...")
			return nil
		}

		localLog.Info("Starting state verification. Waiting 40 seconds for stellar-core to publish HAS...")
		select {
		case <-s.ctx.Done():
			localLog.Info("State verifier shut down...")
			return nil
		case <-time.After(40 * time.Second):
			// Wait for stellar-core to publish HAS
		}
	}

	localLog.Info("Creating state reader...")

	stateReader, err := s.historyAdapter.GetState(
		s.ctx,
		ledgerSequence,
		s.config.MaxStreamRetries,
	)
	if err != nil {
		return errors.Wrap(err, "Error running GetState")
	}
	defer stateReader.Close()

	verifier := &verify.StateVerifier{
		StateReader:       stateReader,
		TransformFunction: transformEntry,
	}

	assetStats := processors.AssetStatSet{}
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
		data := make([]xdr.LedgerKeyData, 0, verifyBatchSize)
		offers := make([]int64, 0, verifyBatchSize)
		trustLines := make([]xdr.LedgerKeyTrustLine, 0, verifyBatchSize)
		for _, key := range keys {
			switch key.Type {
			case xdr.LedgerEntryTypeAccount:
				accounts = append(accounts, key.Account.AccountId.Address())
			case xdr.LedgerEntryTypeData:
				data = append(data, *key.Data)
			case xdr.LedgerEntryTypeOffer:
				offers = append(offers, int64(key.Offer.OfferId))
			case xdr.LedgerEntryTypeTrustline:
				trustLines = append(trustLines, *key.TrustLine)
			default:
				return errors.New("GetLedgerKeys return unexpected type")
			}
		}

		err = addAccountsToStateVerifier(verifier, historyQ, accounts)
		if err != nil {
			return errors.Wrap(err, "addAccountsToStateVerifier failed")
		}

		err = addDataToStateVerifier(verifier, historyQ, data)
		if err != nil {
			return errors.Wrap(err, "addDataToStateVerifier failed")
		}

		err = addOffersToStateVerifier(verifier, historyQ, offers)
		if err != nil {
			return errors.Wrap(err, "addOffersToStateVerifier failed")
		}

		err = addTrustLinesToStateVerifier(verifier, assetStats, historyQ, trustLines)
		if err != nil {
			return errors.Wrap(err, "addTrustLinesToStateVerifier failed")
		}

		total += len(keys)
		localLog.WithField("total", total).Info("Batch added to StateVerifier")
	}

	localLog.WithField("total", total).Info("Finished writing to StateVerifier")

	countAccounts, err := historyQ.CountAccounts()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountAccounts")
	}

	countData, err := historyQ.CountAccountsData()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountData")
	}

	countOffers, err := historyQ.CountOffers()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountOffers")
	}

	countTrustLines, err := historyQ.CountTrustLines()
	if err != nil {
		return errors.Wrap(err, "Error running historyQ.CountTrustLines")
	}

	err = verifier.Verify(countAccounts + countData + countOffers + countTrustLines)
	if err != nil {
		return errors.Wrap(err, "verifier.Verify failed")
	}

	err = checkAssetStats(assetStats, historyQ)
	if err != nil {
		return errors.Wrap(err, "checkAssetStats failed")
	}

	localLog.Info("State correct")
	updateMetrics = true
	return nil
}

func checkAssetStats(set processors.AssetStatSet, q history.IngestionQ) error {
	page := db2.PageQuery{
		Order: "asc",
		Limit: assetStatsBatchSize,
	}

	for {
		assetStats, err := q.GetAssetStats("", "", page)
		if err != nil {
			return errors.Wrap(err, "could not fetch asset stats from db")
		}
		if len(assetStats) == 0 {
			break
		}

		for _, assetStat := range assetStats {
			fromSet, removed := set.Remove(assetStat.AssetType, assetStat.AssetCode, assetStat.AssetIssuer)
			if !removed {
				return ingesterrors.NewStateError(
					fmt.Errorf(
						"db contains asset stat with code %s issuer %s which is missing from HAS",
						assetStat.AssetCode, assetStat.AssetIssuer,
					),
				)
			}

			if fromSet != assetStat {
				return ingesterrors.NewStateError(
					fmt.Errorf(
						"db asset stat with code %s issuer %s does not match asset stat from HAS",
						assetStat.AssetCode, assetStat.AssetIssuer,
					),
				)
			}
		}

		page.Cursor = assetStats[len(assetStats)-1].PagingToken()
	}

	if len(set) > 0 {
		return ingesterrors.NewStateError(
			fmt.Errorf(
				"HAS contains %d more asset stats than db",
				len(set),
			),
		)
	}
	return nil
}

func addAccountsToStateVerifier(verifier *verify.StateVerifier, q history.IngestionQ, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	accounts, err := q.GetAccountsByIDs(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetAccountsByIDs")
	}

	signers, err := q.SignersForAccounts(ids)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.SignersForAccounts")
	}

	masterWeightMap := make(map[string]int32)
	signersMap := make(map[string][]xdr.Signer)
	for _, row := range signers {
		if row.Account == row.Signer {
			masterWeightMap[row.Account] = row.Weight
		} else {
			signersMap[row.Account] = append(
				signersMap[row.Account],
				xdr.Signer{
					Key:    xdr.MustSigner(row.Signer),
					Weight: xdr.Uint32(row.Weight),
				},
			)
		}
	}

	for _, row := range accounts {
		var inflationDest *xdr.AccountId
		if row.InflationDestination != "" {
			t := xdr.MustAddress(row.InflationDestination)
			inflationDest = &t
		}

		// Ensure master weight matches, if not it's a state error!
		if int32(row.MasterWeight) != masterWeightMap[row.AccountID] {
			return ingesterrors.NewStateError(
				fmt.Errorf(
					"Master key weight in account %s does not match (expected=%d, actual=%d)",
					row.AccountID,
					masterWeightMap[row.AccountID],
					int32(row.MasterWeight),
				),
			)
		}

		account := &xdr.AccountEntry{
			AccountId:     xdr.MustAddress(row.AccountID),
			Balance:       xdr.Int64(row.Balance),
			SeqNum:        xdr.SequenceNumber(row.SequenceNumber),
			NumSubEntries: xdr.Uint32(row.NumSubEntries),
			InflationDest: inflationDest,
			Flags:         xdr.Uint32(row.Flags),
			HomeDomain:    xdr.String32(row.HomeDomain),
			Thresholds: xdr.Thresholds{
				row.MasterWeight,
				row.ThresholdLow,
				row.ThresholdMedium,
				row.ThresholdHigh,
			},
			Signers: xdr.SortSignersByKey(signersMap[row.AccountID]),
			Ext: xdr.AccountEntryExt{
				V: 1,
				V1: &xdr.AccountEntryV1{
					Liabilities: xdr.Liabilities{
						Buying:  xdr.Int64(row.BuyingLiabilities),
						Selling: xdr.Int64(row.SellingLiabilities),
					},
				},
			},
		}

		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
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

func addDataToStateVerifier(verifier *verify.StateVerifier, q history.IngestionQ, keys []xdr.LedgerKeyData) error {
	if len(keys) == 0 {
		return nil
	}

	data, err := q.GetAccountDataByKeys(keys)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetAccountDataByKeys")
	}

	for _, row := range data {
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeData,
				Data: &xdr.DataEntry{
					AccountId: xdr.MustAddress(row.AccountID),
					DataName:  xdr.String64(row.Name),
					DataValue: xdr.DataValue(row.Value),
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

func addOffersToStateVerifier(
	verifier *verify.StateVerifier,
	q history.IngestionQ,
	ids []int64,
) error {
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

func addTrustLinesToStateVerifier(
	verifier *verify.StateVerifier,
	assetStats processors.AssetStatSet,
	q history.IngestionQ,
	keys []xdr.LedgerKeyTrustLine,
) error {
	if len(keys) == 0 {
		return nil
	}

	trustLines, err := q.GetTrustLinesByKeys(keys)
	if err != nil {
		return errors.Wrap(err, "Error running history.Q.GetTrustLinesByKeys")
	}

	for _, row := range trustLines {
		asset := xdr.MustNewCreditAsset(row.AssetCode, row.AssetIssuer)
		trustline := xdr.TrustLineEntry{
			AccountId: xdr.MustAddress(row.AccountID),
			Asset:     asset,
			Balance:   xdr.Int64(row.Balance),
			Limit:     xdr.Int64(row.Limit),
			Flags:     xdr.Uint32(row.Flags),
			Ext: xdr.TrustLineEntryExt{
				V: 1,
				V1: &xdr.TrustLineEntryV1{
					Liabilities: xdr.Liabilities{
						Buying:  xdr.Int64(row.BuyingLiabilities),
						Selling: xdr.Int64(row.SellingLiabilities),
					},
				},
			},
		}
		entry := xdr.LedgerEntry{
			LastModifiedLedgerSeq: xdr.Uint32(row.LastModifiedLedger),
			Data: xdr.LedgerEntryData{
				Type:      xdr.LedgerEntryTypeTrustline,
				TrustLine: &trustline,
			},
		}
		if err := verifier.Write(entry); err != nil {
			return err
		}
		if err := assetStats.Add(trustline); err != nil {
			return ingesterrors.NewStateError(
				errors.Wrap(err, "could not add trustline to asset stats"),
			)
		}
	}

	return nil
}

func transformEntry(entry xdr.LedgerEntry) (bool, xdr.LedgerEntry) {
	switch entry.Data.Type {
	case xdr.LedgerEntryTypeAccount:
		accountEntry := entry.Data.Account
		// Sort signers
		accountEntry.Signers = xdr.SortSignersByKey(accountEntry.Signers)
		// Account can have ext=0. For those, create ext=1
		// with 0 liabilities.
		if accountEntry.Ext.V == 0 {
			accountEntry.Ext.V = 1
			accountEntry.Ext.V1 = &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  0,
					Selling: 0,
				},
			}
		}

		return false, entry
	case xdr.LedgerEntryTypeOffer:
		// Full check of offer object
		return false, entry
	case xdr.LedgerEntryTypeTrustline:
		// Trust line can have ext=0. For those, create ext=1
		// with 0 liabilities.
		trustLineEntry := entry.Data.TrustLine
		if trustLineEntry.Ext.V == 0 {
			trustLineEntry.Ext.V = 1
			trustLineEntry.Ext.V1 = &xdr.TrustLineEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  0,
					Selling: 0,
				},
			}
		}

		return false, entry
	case xdr.LedgerEntryTypeData:
		// Full check of data object
		return false, entry
	default:
		panic("Invalid type")
	}
}
