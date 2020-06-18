package expingest

import (
	"bytes"
	"context"
	"fmt"

	"github.com/stellar/go/exp/ingest/adapters"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest/processors"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type ingestionSource int

const (
	_                    = iota
	historyArchiveSource = ingestionSource(iota)
	ledgerSource         = ingestionSource(iota)
	logFrequency         = 100000
)

type horizonTransactionProcessor interface {
	io.LedgerTransactionProcessor
	// TODO maybe rename to Flush()
	Commit() error
}

type statsChangeProcessor struct {
	*io.StatsChangeProcessor
}

func (statsChangeProcessor) Commit() error {
	return nil
}

type statsLedgerTransactionProcessor struct {
	*io.StatsLedgerTransactionProcessor
}

func (statsLedgerTransactionProcessor) Commit() error {
	return nil
}

type ProcessorRunnerInterface interface {
	SetLedgerBackend(ledgerBackend ledgerbackend.LedgerBackend)
	SetHistoryAdapter(historyAdapter adapters.HistoryArchiveAdapterInterface)
	EnableMemoryStatsLogging()
	DisableMemoryStatsLogging()
	RunHistoryArchiveIngestion(checkpointLedger uint32) (io.StatsChangeProcessorResults, error)
	RunTransactionProcessorsOnLedger(sequence uint32) (io.StatsLedgerTransactionProcessorResults, error)
	RunAllProcessorsOnLedger(sequence uint32) (
		io.StatsChangeProcessorResults,
		io.StatsLedgerTransactionProcessorResults,
		error,
	)
}

var _ ProcessorRunnerInterface = (*ProcessorRunner)(nil)

type ProcessorRunner struct {
	config Config

	ctx            context.Context
	historyQ       history.IngestionQ
	historyAdapter adapters.HistoryArchiveAdapterInterface
	ledgerBackend  ledgerbackend.LedgerBackend
	logMemoryStats bool
}

func (s *ProcessorRunner) SetLedgerBackend(ledgerBackend ledgerbackend.LedgerBackend) {
	s.ledgerBackend = ledgerBackend
}

func (s *ProcessorRunner) SetHistoryAdapter(historyAdapter adapters.HistoryArchiveAdapterInterface) {
	s.historyAdapter = historyAdapter
}

func (s *ProcessorRunner) EnableMemoryStatsLogging() {
	s.logMemoryStats = true
}

func (s *ProcessorRunner) DisableMemoryStatsLogging() {
	s.logMemoryStats = false
}

func (s *ProcessorRunner) buildChangeProcessor(
	changeStats *io.StatsChangeProcessor,
	source ingestionSource,
	sequence uint32,
) horizonChangeProcessor {
	statsChangeProcessor := &statsChangeProcessor{
		StatsChangeProcessor: changeStats,
	}

	useLedgerCache := source == ledgerSource
	return groupChangeProcessors{
		statsChangeProcessor,
		processors.NewAccountDataProcessor(s.historyQ),
		processors.NewAccountsProcessor(s.historyQ),
		processors.NewOffersProcessor(s.historyQ, sequence),
		processors.NewAssetStatsProcessor(s.historyQ, useLedgerCache),
		processors.NewSignersProcessor(s.historyQ, useLedgerCache),
		processors.NewTrustLinesProcessor(s.historyQ),
	}
}

type skipFailedTransactions struct {
	horizonTransactionProcessor
}

func (p skipFailedTransactions) ProcessTransaction(tx io.LedgerTransaction) error {
	if !tx.Result.Successful() {
		return nil
	}
	return p.horizonTransactionProcessor.ProcessTransaction(tx)
}

func (s *ProcessorRunner) buildTransactionProcessor(
	ledgerTransactionStats *io.StatsLedgerTransactionProcessor,
	ledger xdr.LedgerHeaderHistoryEntry,
) horizonTransactionProcessor {
	statsLedgerTransactionProcessor := &statsLedgerTransactionProcessor{
		StatsLedgerTransactionProcessor: ledgerTransactionStats,
	}

	sequence := uint32(ledger.Header.LedgerSeq)
	group := groupTransactionProcessors{
		statsLedgerTransactionProcessor,
		processors.NewEffectProcessor(s.historyQ, sequence),
		processors.NewLedgerProcessor(s.historyQ, ledger, CurrentVersion),
		processors.NewOperationProcessor(s.historyQ, sequence),
		processors.NewTradeProcessor(s.historyQ, ledger),
		processors.NewParticipantsProcessor(s.historyQ, sequence),
		processors.NewTransactionProcessor(s.historyQ, sequence),
	}

	if s.config.IngestFailedTransactions {
		return group
	}

	return skipFailedTransactions{group}
}

// validateBucketList validates if the bucket list hash in history archive
// matches the one in corresponding ledger header in stellar-core backend.
// This gives you full security if data in stellar-core backend can be trusted
// (ex. you run it in your infrastructure).
// The hashes of actual buckets of this HAS file are checked using
// historyarchive.XdrStream.SetExpectedHash (this is done in
// SingleLedgerStateReader).
func (s *ProcessorRunner) validateBucketList(ledgerSequence uint32) error {
	historyBucketListHash, err := s.historyAdapter.BucketListHash(ledgerSequence)
	if err != nil {
		return errors.Wrap(err, "Error getting bucket list hash")
	}

	ledgerReader, err := io.NewDBLedgerReader(s.ctx, ledgerSequence, s.ledgerBackend)
	if err != nil {
		if err == io.ErrNotFound {
			return fmt.Errorf(
				"cannot validate bucket hash list. Checkpoint ledger (%d) must exist in Stellar-Core database.",
				ledgerSequence,
			)
		} else {
			return errors.Wrap(err, "Error getting ledger")
		}
	}

	ledgerHeader := ledgerReader.GetHeader()
	ledgerBucketHashList := ledgerHeader.Header.BucketListHash

	if !bytes.Equal(historyBucketListHash[:], ledgerBucketHashList[:]) {
		return fmt.Errorf(
			"Bucket list hash of history archive and ledger header does not match: %#x %#x",
			historyBucketListHash,
			ledgerBucketHashList,
		)
	}

	return nil
}

func (s *ProcessorRunner) RunHistoryArchiveIngestion(checkpointLedger uint32) (io.StatsChangeProcessorResults, error) {
	changeStats := io.StatsChangeProcessor{}
	changeProcessor := s.buildChangeProcessor(&changeStats, historyArchiveSource, checkpointLedger)

	var changeReader io.ChangeReader
	var err error

	if checkpointLedger == 1 {
		changeReader = &io.GenesisLedgerStateReader{
			NetworkPassphrase: s.config.NetworkPassphrase,
		}
	} else {
		if err = s.validateBucketList(checkpointLedger); err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error validating bucket list from HAS")
		}

		changeReader, err = s.historyAdapter.GetState(
			s.ctx,
			checkpointLedger,
			s.config.MaxStreamRetries,
		)
		if err != nil {
			return changeStats.GetResults(), errors.Wrap(err, "Error creating HAS reader")
		}
	}
	defer changeReader.Close()

	log.WithField("ledger", checkpointLedger).
		Info("Processing entries from History Archive Snapshot")

	err = io.StreamChanges(changeProcessor, newloggingChangeReader(
		changeReader,
		"historyArchive",
		checkpointLedger,
		logFrequency,
		s.logMemoryStats,
	))
	if err != nil {
		return changeStats.GetResults(), errors.Wrap(err, "Error streaming changes from HAS")
	}

	err = changeProcessor.Commit()
	if err != nil {
		return changeStats.GetResults(), errors.Wrap(err, "Error commiting changes from processor")
	}

	return changeStats.GetResults(), nil
}

func (s *ProcessorRunner) runChangeProcessorOnLedger(
	changeProcessor horizonChangeProcessor, ledger uint32,
) error {
	var changeReader io.ChangeReader
	var err error
	changeReader, err = io.NewLedgerChangeReader(s.ctx, ledger, s.ledgerBackend)
	if err != nil {
		return errors.Wrap(err, "Error creating ledger change reader")
	}
	changeReader = newloggingChangeReader(
		changeReader,
		"ledger",
		ledger,
		logFrequency,
		s.logMemoryStats,
	)
	if err = io.StreamChanges(changeProcessor, changeReader); err != nil {
		return errors.Wrap(err, "Error streaming changes from ledger")
	}

	err = changeProcessor.Commit()
	if err != nil {
		return errors.Wrap(err, "Error commiting changes from processor")
	}

	return nil
}

func (s *ProcessorRunner) RunTransactionProcessorsOnLedger(ledger uint32) (io.StatsLedgerTransactionProcessorResults, error) {
	ledgerTransactionStats := io.StatsLedgerTransactionProcessor{}

	ledgerReader, err := io.NewDBLedgerReader(s.ctx, ledger, s.ledgerBackend)
	if err != nil {
		return ledgerTransactionStats.GetResults(), errors.Wrap(err, "Error creating ledger reader")
	}

	txProcessor := s.buildTransactionProcessor(&ledgerTransactionStats, ledgerReader.GetHeader())
	err = io.StreamLedgerTransactions(txProcessor, ledgerReader)
	if err != nil {
		return ledgerTransactionStats.GetResults(), errors.Wrap(err, "Error streaming changes from ledger")
	}

	err = txProcessor.Commit()
	if err != nil {
		return ledgerTransactionStats.GetResults(), errors.Wrap(err, "Error commiting changes from processor")
	}

	return ledgerTransactionStats.GetResults(), nil
}

func (s *ProcessorRunner) RunAllProcessorsOnLedger(sequence uint32) (io.StatsChangeProcessorResults, io.StatsLedgerTransactionProcessorResults, error) {
	changeStats := io.StatsChangeProcessor{}
	var statsLedgerTransactionProcessorResults io.StatsLedgerTransactionProcessorResults

	err := s.runChangeProcessorOnLedger(
		s.buildChangeProcessor(&changeStats, ledgerSource, sequence),
		sequence,
	)
	if err != nil {
		return changeStats.GetResults(), statsLedgerTransactionProcessorResults, err
	}

	statsLedgerTransactionProcessorResults, err = s.RunTransactionProcessorsOnLedger(sequence)
	if err != nil {
		return changeStats.GetResults(), statsLedgerTransactionProcessorResults, err
	}

	return changeStats.GetResults(), statsLedgerTransactionProcessorResults, nil
}
