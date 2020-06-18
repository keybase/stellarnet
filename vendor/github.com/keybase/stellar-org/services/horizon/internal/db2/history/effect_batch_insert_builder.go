package history

import (
	"github.com/stellar/go/support/db"
)

// EffectBatchInsertBuilder is used to insert effects into the
// history_effects table
type EffectBatchInsertBuilder interface {
	Add(
		accountID int64,
		operationID int64,
		order uint32,
		effectType EffectType,
		details []byte,
	) error
	Exec() error
}

// effectBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type effectBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewEffectBatchInsertBuilder constructs a new EffectBatchInsertBuilder instance
func (q *Q) NewEffectBatchInsertBuilder(maxBatchSize int) EffectBatchInsertBuilder {
	return &effectBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_effects"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds a effect to the batch
func (i *effectBatchInsertBuilder) Add(
	accountID int64,
	operationID int64,
	order uint32,
	effectType EffectType,
	details []byte,
) error {
	return i.builder.Row(map[string]interface{}{
		"history_account_id":   accountID,
		"history_operation_id": operationID,
		"\"order\"":            order,
		"type":                 effectType,
		"details":              details,
	})
}

func (i *effectBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
