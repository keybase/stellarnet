// Package db is the base package for database access at stellar.  It primarily
// exposes Session which is a lightweight wrapper around a *sqlx.DB that
// provides utility methods (See the repo tests for examples).
//
// In addition to the query methods, this package provides query logging and
// stateful transaction management.
//
// In addition to the lower-level access facilities, this package exposes a
// system to build queries more dynamically using the help of
// https://github.com/Masterminds/squirrel.  These builder method are access
// through the `Table` type.
package db

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/support/errors"

	// Enable postgres
	_ "github.com/lib/pq"
)

// postgresQueryMaxParams defines the maximum number of parameters in a query.
var postgresQueryMaxParams = 65535

var (
	// ErrCancelled is an error returned by Session methods when request has
	// been cancelled (ex. context cancelled).
	ErrCancelled = errors.New("canceling statement due to user request")
)

// Conn represents a connection to a single database.
type Conn interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Rebind(sql string) string
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// DeleteBuilder is a helper struct used to construct sql queries of the DELETE
// variety.
type DeleteBuilder struct {
	Table *Table
	sql   squirrel.DeleteBuilder
}

// InsertBuilder is a helper struct used to construct sql queries of the INSERT
// variety.
// NOTE: InsertBuilder will use "zero" value of a type in case of nil pointer values.
// If you need to insert `NULL` use sql.Null* or build your own type that implements
// database/sql/driver.Valuer.
type InsertBuilder struct {
	Table *Table

	rows        []interface{}
	ignoredCols map[string]bool
	sql         squirrel.InsertBuilder
}

// GetBuilder is a helper struct used to construct sql queries of the SELECT
// variety.
type GetBuilder struct {
	Table *Table

	dest interface{}
	sql  squirrel.SelectBuilder
}

// SelectBuilder is a helper struct used to construct sql queries of the SELECT
// variety.
type SelectBuilder struct {
	Table *Table

	dest interface{}
	sql  squirrel.SelectBuilder
}

// UpdateBuilder is a helper struct used to construct sql queries of the UPDATE
// variety.
type UpdateBuilder struct {
	Table *Table

	source interface{}
	sql    squirrel.UpdateBuilder
}

// Session provides helper methods for making queries against `DB` and provides
// utilities such as automatic query logging and transaction management.  NOTE:
// A Session is designed to be lightweight and temporarily lived (usually
// request scoped) which is one reason it is acceptable for it to store a
// context.  It is not presently intended to cross goroutine boundaries and is
// not concurrency safe.
type Session struct {
	// DB is the database connection that queries should be executed against.
	DB *sqlx.DB

	// Ctx is the context in which the repo is operating under.
	Ctx context.Context

	tx        *sqlx.Tx
	txOptions *sql.TxOptions
}

type SessionInterface interface {
	BeginTx(opts *sql.TxOptions) error
	Begin() error
	Rollback() error
	TruncateTables(tables []string) error
	Clone() *Session
	Close() error
	Get(dest interface{}, query squirrel.Sqlizer) error
	GetRaw(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query squirrel.Sqlizer) error
	SelectRaw(dest interface{}, query string, args ...interface{}) error
	GetTable(name string) *Table
	Exec(query squirrel.Sqlizer) (sql.Result, error)
	ExecRaw(query string, args ...interface{}) (sql.Result, error)
	NoRows(err error) bool
}

// Table helps to build sql queries against a given table.  It logically
// represents a SQL table on the database that `Session` is connected to.
type Table struct {
	// Name is the name of the table
	Name string

	Session *Session
}

// Open the database at `dsn` and returns a new *Session using it.
func Open(dialect, dsn string) (*Session, error) {
	db, err := sqlx.Connect(dialect, dsn)
	if err != nil {
		return nil, errors.Wrap(err, "connect failed")
	}

	return &Session{DB: db, Ctx: context.Background()}, nil
}

// Wrap wraps a bare *sql.DB (from the database/sql stdlib package) in a
// *db.Session instance.  It is meant to be used in cases where you do not
// control the instantiation of the database connection, but would still like to
// leverage the facilities provided in Session.
func Wrap(base *sql.DB, dialect string) *Session {
	return &Session{DB: sqlx.NewDb(base, dialect), Ctx: context.Background()}
}

// ensure various types conform to Conn interface
var _ Conn = (*sqlx.Tx)(nil)
var _ Conn = (*sqlx.DB)(nil)
