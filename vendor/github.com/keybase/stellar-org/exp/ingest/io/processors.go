package io

import (
	"io"

	"github.com/stellar/go/support/errors"
)

type ChangeProcessor interface {
	ProcessChange(change Change) error
}

type LedgerTransactionProcessor interface {
	ProcessTransaction(transaction LedgerTransaction) error
}

func StreamLedgerTransactions(
	txProcessor LedgerTransactionProcessor,
	reader LedgerReader,
) error {
	for {
		tx, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "could not read transaction")
		}
		if err = txProcessor.ProcessTransaction(tx); err != nil {
			return errors.Wrapf(
				err,
				"could not process transaction %v",
				tx.Index,
			)
		}
	}
}

func StreamChanges(
	changeProcessor ChangeProcessor,
	reader ChangeReader,
) error {
	for {
		change, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "could not read transaction")
		}

		if err = changeProcessor.ProcessChange(change); err != nil {
			return errors.Wrap(
				err,
				"could not process change",
			)
		}
	}
}
