package history

import (
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

// accountsBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type accountsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func (i *accountsBatchInsertBuilder) Add(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) error {
	return i.builder.Row(accountToMap(account, lastModifiedLedger))
}

func (i *accountsBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
