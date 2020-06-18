package history

import (
	"github.com/stellar/go/support/db"
)

// QParticipants defines ingestion participant related queries.
type QParticipants interface {
	QCreateAccountsHistory
	NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) TransactionParticipantsBatchInsertBuilder
	NewOperationParticipantBatchInsertBuilder(maxBatchSize int) OperationParticipantBatchInsertBuilder
}

// TransactionParticipantsBatchInsertBuilder is used to insert transaction participants into the
// history_transaction_participants table
type TransactionParticipantsBatchInsertBuilder interface {
	Add(transactionID, accountID int64) error
	Exec() error
}

type transactionParticipantsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewTransactionParticipantsBatchInsertBuilder constructs a new TransactionParticipantsBatchInsertBuilder instance
func (q *Q) NewTransactionParticipantsBatchInsertBuilder(maxBatchSize int) TransactionParticipantsBatchInsertBuilder {
	return &transactionParticipantsBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_transaction_participants"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds a new transaction participant to the batch
func (i *transactionParticipantsBatchInsertBuilder) Add(transactionID, accountID int64) error {
	return i.builder.Row(map[string]interface{}{
		"history_transaction_id": transactionID,
		"history_account_id":     accountID,
	})
}

// Exec flushes all pending transaction participants to the db
func (i *transactionParticipantsBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
