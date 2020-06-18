package db

import (
	"database/sql"

	"github.com/Masterminds/squirrel"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/mock"
)

var _ SessionInterface = (*MockSession)(nil)

type MockSession struct {
	mock.Mock
}

func (m *MockSession) Begin() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSession) BeginTx(opts *sql.TxOptions) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *MockSession) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSession) TruncateTables(tables []string) error {
	args := m.Called(tables)
	return args.Error(0)
}

func (m *MockSession) Clone() *Session {
	args := m.Called()
	return args.Get(0).(*Session)
}

func (m *MockSession) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSession) Get(dest interface{}, query sq.Sqlizer) error {
	args := m.Called(dest, query)
	return args.Error(0)
}

func (m *MockSession) GetRaw(dest interface{}, query string, args ...interface{}) error {
	argss := m.Called(dest, query, args)
	return argss.Error(0)
}

func (m *MockSession) Select(dest interface{}, query squirrel.Sqlizer) error {
	argss := m.Called(dest, query)
	return argss.Error(0)
}

func (m *MockSession) SelectRaw(
	dest interface{},
	query string,
	args ...interface{},
) error {
	argss := m.Called(dest, query, args)
	return argss.Error(0)
}

func (m *MockSession) GetTable(name string) *Table {
	args := m.Called(name)
	return args.Get(0).(*Table)
}

func (m *MockSession) Exec(query squirrel.Sqlizer) (sql.Result, error) {
	args := m.Called(query)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockSession) ExecRaw(query string, args ...interface{}) (sql.Result, error) {
	argss := m.Called(query, args)
	return argss.Get(0).(sql.Result), argss.Error(1)
}

func (m *MockSession) NoRows(err error) bool {
	args := m.Called(err)
	return args.Get(0).(bool)
}
