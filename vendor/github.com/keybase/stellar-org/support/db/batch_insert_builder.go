package db

import (
	"fmt"
	"sort"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
)

// Row adds a new row to the batch. All rows must have exactly the same columns
// (map keys). Otherwise, error will be returned. Please note that rows are not
// added one by one but in batches when `Exec` is called (or `MaxBatchSize` is
// reached).
func (b *BatchInsertBuilder) Row(row map[string]interface{}) error {
	if b.columns == nil {
		b.columns = make([]string, 0, len(row))
		b.rows = make([][]interface{}, 0)

		for column := range row {
			b.columns = append(b.columns, column)
		}

		sort.Strings(b.columns)
	}

	if len(b.columns) != len(row) {
		return errors.Errorf("invalid number of columns (expected=%d, actual=%d)", len(b.columns), len(row))
	}

	rowSlice := make([]interface{}, 0, len(b.columns))
	for _, column := range b.columns {
		val, ok := row[column]
		if !ok {
			return errors.Errorf(`column "%s" does not exist`, column)
		}
		rowSlice = append(rowSlice, val)
	}

	b.rows = append(b.rows, rowSlice)

	// Call Exec when MaxBatchSize is reached.
	if len(b.rows) == b.MaxBatchSize {
		return b.Exec()
	}

	return nil
}

// Exec inserts rows in batches. In case of errors it's possible that some batches
// were added so this should be run in a DB transaction for easy rollbacks.
func (b *BatchInsertBuilder) Exec() error {
	b.sql = sq.Insert(b.Table.Name).Columns(b.columns...)
	paramsCount := 0

	for _, row := range b.rows {
		b.sql = b.sql.Values(row...)
		paramsCount += len(row)

		if paramsCount > postgresQueryMaxParams-2*len(b.columns) {
			_, err := b.Table.Session.Exec(b.sql)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error adding values while inserting to %s", b.Table.Name))
			}
			paramsCount = 0
			b.sql = sq.Insert(b.Table.Name).Columns(b.columns...)
		}
	}

	// Insert last batch
	if paramsCount > 0 {
		_, err := b.Table.Session.Exec(b.sql)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error adding values while inserting to %s", b.Table.Name))
		}
	}

	// Clear the rows so user can reuse it for batch inserting to a single table
	b.rows = make([][]interface{}, 0)
	return nil
}
