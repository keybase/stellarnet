package processors

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type LedgersProcessor struct {
	ledgersQ       history.QLedgers
	ledger         xdr.LedgerHeaderHistoryEntry
	ingestVersion  int
	successTxCount int
	failedTxCount  int
	opCount        int
}

func NewLedgerProcessor(
	ledgerQ history.QLedgers,
	ledger xdr.LedgerHeaderHistoryEntry,
	ingestVersion int,
) *LedgersProcessor {
	return &LedgersProcessor{
		ledger:        ledger,
		ledgersQ:      ledgerQ,
		ingestVersion: ingestVersion,
	}
}

func (p *LedgersProcessor) ProcessTransaction(transaction io.LedgerTransaction) (err error) {
	if transaction.Result.Successful() {
		p.successTxCount++
		p.opCount += len(transaction.Envelope.Operations())
	} else {
		p.failedTxCount++
	}

	return nil
}

func (p *LedgersProcessor) Commit() error {
	rowsAffected, err := p.ledgersQ.InsertLedger(
		p.ledger,
		p.successTxCount,
		p.failedTxCount,
		p.opCount,
		p.ingestVersion,
	)

	if err != nil {
		return errors.Wrap(err, "Could not insert ledger")
	}

	sequence := uint32(p.ledger.Header.LedgerSeq)

	if rowsAffected != 1 {
		log.WithField("rowsAffected", rowsAffected).
			WithField("sequence", sequence).
			Error("Invalid number of rows affected when ingesting new ledger")
		return errors.Errorf(
			"0 rows affected when ingesting new ledger: %v",
			sequence,
		)
	}

	return nil
}
