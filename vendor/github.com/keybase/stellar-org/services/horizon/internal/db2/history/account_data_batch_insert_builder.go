package history

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (i *accountDataBatchInsertBuilder) Add(data xdr.DataEntry, lastModifiedLedger xdr.Uint32) error {
	// Add ledger_key only when inserting rows
	key, err := dataEntryToLedgerKeyString(data)
	if err != nil {
		return errors.Wrap(err, "Error running dataEntryToLedgerKeyString")
	}

	return i.builder.Row(map[string]interface{}{
		"ledger_key":           key,
		"account_id":           data.AccountId.Address(),
		"name":                 data.DataName,
		"value":                AccountDataValue(data.DataValue),
		"last_modified_ledger": lastModifiedLedger,
	})
}

func (i *accountDataBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
