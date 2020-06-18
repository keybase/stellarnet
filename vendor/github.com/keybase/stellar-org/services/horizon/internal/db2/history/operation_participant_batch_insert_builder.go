package history

import (
	"github.com/stellar/go/support/db"
)

// OperationParticipantBatchInsertBuilder is used to insert a transaction's operations into the
// history_operations table
type OperationParticipantBatchInsertBuilder interface {
	Add(
		operationID int64,
		accountID int64,
	) error
	Exec() error
}

// operationParticipantBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type operationParticipantBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewOperationParticipantBatchInsertBuilder constructs a new TransactionBatchInsertBuilder instance
func (q *Q) NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder {
	return &operationParticipantBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_operation_participants"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds an operation participant to the batch
func (i *operationParticipantBatchInsertBuilder) Add(
	operationID int64,
	accountID int64,
) error {
	return i.builder.Row(map[string]interface{}{
		"history_operation_id": operationID,
		"history_account_id":   accountID,
	})
}

func (i *operationParticipantBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
