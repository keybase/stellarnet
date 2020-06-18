package main

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const maxStreamRetries = 3

// csvMap maintains a mapping from ledger entry type to csv file
type csvMap struct {
	files   map[xdr.LedgerEntryType]*os.File
	writers map[xdr.LedgerEntryType]*csv.Writer
}

// newCSVMap constructs an empty csvMap instance
func newCSVMap() csvMap {
	return csvMap{
		files:   map[xdr.LedgerEntryType]*os.File{},
		writers: map[xdr.LedgerEntryType]*csv.Writer{},
	}
}

// put creates a new file with the given file name and links that file to the
// given ledger entry type
func (c csvMap) put(entryType xdr.LedgerEntryType, fileName string) error {
	if _, ok := c.files[entryType]; ok {
		return errors.Errorf("entry type %s is already present in the file set", fileName)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrapf(err, "could not open file %s", fileName)
	}

	c.files[entryType] = file
	c.writers[entryType] = csv.NewWriter(file)

	return nil
}

// get returns a csv writer for the given ledger entry type if it exists in the mapping
func (c csvMap) get(entryType xdr.LedgerEntryType) (*csv.Writer, bool) {
	writer, ok := c.writers[entryType]
	return writer, ok
}

// close will close all files contained in the mapping
func (c csvMap) close() {
	for entryType, file := range c.files {
		if err := file.Close(); err != nil {
			log.WithField("type", entryType.String()).Warn("could not close csv file")
		}
		delete(c.files, entryType)
		delete(c.writers, entryType)
	}
}

type csvProcessor struct {
	files       csvMap
	changeStats *io.StatsChangeProcessor
}

func (processor csvProcessor) ProcessChange(change io.Change) error {
	csvWriter, ok := processor.files.get(change.Type)
	if !ok {
		return nil
	}
	if err := processor.changeStats.ProcessChange(change); err != nil {
		return err
	}

	switch change.Type {
	case xdr.LedgerEntryTypeAccount:
		account := change.Post.Data.MustAccount()

		inflationDest := ""
		if account.InflationDest != nil {
			inflationDest = account.InflationDest.Address()
		}

		var signers string
		if len(account.Signers) > 0 {
			var err error
			signers, err = xdr.MarshalBase64(account.Signers)
			if err != nil {
				return err
			}
		}

		var buyingLiabilities, sellingLiabilities int64

		if account.Ext.V1 != nil {
			buyingLiabilities = int64(account.Ext.V1.Liabilities.Buying)
			sellingLiabilities = int64(account.Ext.V1.Liabilities.Selling)
		}

		csvWriter.Write([]string{
			account.AccountId.Address(),
			strconv.FormatInt(int64(account.Balance), 10),
			strconv.FormatInt(int64(account.SeqNum), 10),
			strconv.FormatInt(int64(account.NumSubEntries), 10),
			inflationDest,
			base64.StdEncoding.EncodeToString([]byte(account.HomeDomain)),
			base64.StdEncoding.EncodeToString(account.Thresholds[:]),
			strconv.FormatInt(int64(account.Flags), 10),
			strconv.FormatInt(buyingLiabilities, 10),
			strconv.FormatInt(sellingLiabilities, 10),
			signers,
		})
	case xdr.LedgerEntryTypeTrustline:
		trustline := change.Post.Data.MustTrustLine()

		var assetType, assetCode, assetIssuer string
		trustline.Asset.MustExtract(&assetType, &assetCode, &assetIssuer)

		var buyingLiabilities, sellingLiabilities int64

		if trustline.Ext.V1 != nil {
			buyingLiabilities = int64(trustline.Ext.V1.Liabilities.Buying)
			sellingLiabilities = int64(trustline.Ext.V1.Liabilities.Selling)
		}

		csvWriter.Write([]string{
			trustline.AccountId.Address(),
			strconv.FormatInt(int64(trustline.Asset.Type), 10),
			assetIssuer,
			assetCode,
			strconv.FormatInt(int64(trustline.Limit), 10),
			strconv.FormatInt(int64(trustline.Balance), 10),
			strconv.FormatInt(int64(trustline.Flags), 10),
			strconv.FormatInt(buyingLiabilities, 10),
			strconv.FormatInt(sellingLiabilities, 10),
		})
	case xdr.LedgerEntryTypeOffer:
		offer := change.Post.Data.MustOffer()

		selling, err := xdr.MarshalBase64(offer.Selling)
		if err != nil {
			return err
		}

		buying, err := xdr.MarshalBase64(offer.Buying)
		if err != nil {
			return err
		}

		csvWriter.Write([]string{
			offer.SellerId.Address(),
			strconv.FormatInt(int64(offer.OfferId), 10),
			selling,
			buying,
			strconv.FormatInt(int64(offer.Amount), 10),
			strconv.FormatInt(int64(offer.Price.N), 10),
			strconv.FormatInt(int64(offer.Price.D), 10),
			strconv.FormatInt(int64(offer.Flags), 10),
		})
	case xdr.LedgerEntryTypeData:
		accountData := change.Post.Data.MustData()
		csvWriter.Write([]string{
			accountData.AccountId.Address(),
			base64.StdEncoding.EncodeToString([]byte(accountData.DataName)),
			base64.StdEncoding.EncodeToString(accountData.DataValue),
		})
	default:
		return errors.Errorf("Invalid LedgerEntryType: %d", change.Type)
	}

	if err := csvWriter.Error(); err != nil {
		return errors.Wrap(err, "Error during csv.Writer.Write")
	}

	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		return errors.Wrap(err, "Error during csv.Writer.Flush")
	}
	return nil
}

func main() {
	testnet := flag.Bool("testnet", false, "connect to the Stellar test network")
	flag.Parse()

	archive, err := archive(*testnet)
	if err != nil {
		panic(err)
	}
	log.SetLevel(log.InfoLevel)

	files := newCSVMap()
	defer files.close()

	for entryType, fileName := range map[xdr.LedgerEntryType]string{
		xdr.LedgerEntryTypeAccount:   "./accounts.csv",
		xdr.LedgerEntryTypeData:      "./accountdata.csv",
		xdr.LedgerEntryTypeOffer:     "./offers.csv",
		xdr.LedgerEntryTypeTrustline: "./trustlines.csv",
	} {
		if err = files.put(entryType, fileName); err != nil {
			log.WithField("err", err).
				WithField("file", fileName).
				Fatal("cannot create csv file")
		}
	}

	ledgerSequenceString := os.Getenv("LATEST_LEDGER")
	ledgerSequence, err := strconv.Atoi(ledgerSequenceString)
	if err != nil {
		log.WithField("ledger", ledgerSequenceString).
			WithField("err", err).
			Fatal("cannot parse latest ledger")
	}
	log.WithField("ledger", ledgerSequence).
		Info("Processing entries from History Archive Snapshot")

	changeReader, err := io.MakeSingleLedgerStateReader(
		context.Background(),
		archive,
		uint32(ledgerSequence),
		maxStreamRetries,
	)
	if err != nil {
		log.WithField("err", err).Fatal("cannot construct change reader")
	}
	defer changeReader.Close()

	changeStats := &io.StatsChangeProcessor{}
	doneStats := printPipelineStats(changeStats)
	err = io.StreamChanges(
		csvProcessor{files: files, changeStats: changeStats},
		changeReader,
	)
	if err != nil {
		log.WithField("err", err).Fatal("could not process all changes from HAS")
	}

	// Remove sorted files
	sortedFiles := []string{
		"./accounts_sorted.csv",
		"./accountdata_sorted.csv",
		"./offers_sorted.csv",
		"./trustlines_sorted.csv",
	}
	for _, file := range sortedFiles {
		err := os.Remove(file)
		// Ignore not exist errors
		if err != nil && !os.IsNotExist(err) {
			panic(err)
		}
	}

	doneStats <- true
}

func archive(testnet bool) (*historyarchive.Archive, error) {
	if testnet {
		return historyarchive.Connect(
			"https://history.stellar.org/prd/core-testnet/core_testnet_001",
			historyarchive.ConnectOptions{},
		)
	}

	return historyarchive.Connect(
		fmt.Sprintf("https://history.stellar.org/prd/core-live/core_live_001/"),
		historyarchive.ConnectOptions{},
	)
}

func printPipelineStats(reporter *io.StatsChangeProcessor) chan<- bool {
	startTime := time.Now()
	done := make(chan bool)
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()

		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			results := reporter.GetResults()
			stats := log.F(results.Map())
			stats["Alloc"] = bToMb(m.Alloc)
			stats["HeapAlloc"] = bToMb(m.HeapAlloc)
			stats["Sys"] = bToMb(m.Sys)
			stats["NumGC"] = m.NumGC
			stats["Goroutines"] = runtime.NumGoroutine()
			stats["NumCPU"] = runtime.NumCPU()
			stats["Duration"] = time.Since(startTime)

			log.WithFields(stats).Info("Current Job Status")

			select {
			case <-ticker.C:
				continue
			case <-done:
				// Pipeline done
				return
			}
		}
	}()

	return done
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
