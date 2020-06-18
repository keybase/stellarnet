package history

import (
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

// OperationBatchInsertBuilder is used to insert a transaction's operations into the
// history_operations table
type OperationBatchInsertBuilder interface {
	Add(
		id int64,
		transactionID int64,
		applicationOrder uint32,
		operationType xdr.OperationType,
		details []byte,
		sourceAccount string,
	) error
	Exec() error
}

// operationBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type operationBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewOperationBatchInsertBuilder constructs a new TransactionBatchInsertBuilder instance
func (q *Q) NewOperationBatchInsertBuilder(maxBatchSize int) OperationBatchInsertBuilder {
	return &operationBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_operations"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds a transaction's operations to the batch
func (i *operationBatchInsertBuilder) Add(
	id int64,
	transactionID int64,
	applicationOrder uint32,
	operationType xdr.OperationType,
	details []byte,
	sourceAccount string,
) error {
	return i.builder.Row(map[string]interface{}{
		"id":                id,
		"transaction_id":    transactionID,
		"application_order": applicationOrder,
		"type":              operationType,
		"details":           details,
		"source_account":    sourceAccount,
	})

}

func (i *operationBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
