// Package ingest contains the ingestion system for horizon.  This system takes
// data produced by the connected stellar-core database, transforms it and
// inserts it into the horizon database.
package ingest

import (
	"sync"

	sq "github.com/Masterminds/squirrel"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/support/db"
	ilog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

var log = ilog.DefaultLogger.WithField("service", "ingest")

const (
	// CurrentVersion reflects the latest version of the ingestion
	// algorithm. As rows are ingested into the horizon database, this version is
	// used to tag them.  In the future, any breaking changes introduced by a
	// developer should be accompanied by an increase in this value.
	//
	// Scripts, that have yet to be ported to this codebase can then be leveraged
	// to re-ingest old data with the new algorithm, providing a seamless
	// transition when the ingested data's structure changes.
	CurrentVersion = 16
)

// Address is a type of a param provided to BatchInsertBuilder that gets exchanged
// to record ID in a DB.
type Address string

type TableName string

const (
	AssetStatsTableName              TableName = "asset_stats"
	EffectsTableName                 TableName = "history_effects"
	LedgersTableName                 TableName = "history_ledgers"
	OperationParticipantsTableName   TableName = "history_operation_participants"
	OperationsTableName              TableName = "history_operations"
	TradesTableName                  TableName = "history_trades"
	TransactionParticipantsTableName TableName = "history_transaction_participants"
	TransactionsTableName            TableName = "history_transactions"
)

// Cursor iterates through a stellar core database's ledgers
type Cursor struct {
	// FirstLedger is the beginning of the range of ledgers (inclusive) that will
	// attempt to be ingested in this session.
	FirstLedger int32
	// LastLedger is the end of the range of ledgers (inclusive) that will
	// attempt to be ingested in this session.
	LastLedger int32

	// CoreDB is the stellar-core db that data is ingested from.
	CoreDB *db.Session

	Metrics    *IngesterMetrics
	AssetStats *AssetStats

	// Err is the error that caused this iteration to fail, if any.
	Err error

	// Name is a unique identifier tracking the latest ingested ledger on stellar-core
	Name string

	lg   int32
	tx   int
	op   int
	data *LedgerBundle
}

// Config allows passing some configuration values to System and Session.
type Config struct {
	// EnableAssetStats is a feature flag that determines whether to calculate
	// asset stats in this ingestion system.
	EnableAssetStats bool
	// IngestFailedTransactions is a feature flag that determines if system
	// should ingest failed transactions.
	IngestFailedTransactions bool
	// CursorName is the cursor used for ingesting from stellar-core.
	// Setting multiple cursors in different Horizon instances allows multiple
	// Horizons to ingest from the same stellar-core instance without cursor
	// collisions.
	CursorName string
}

// EffectIngestion is a helper struct to smooth the ingestion of effects.  this
// struct will track what the correct operation to use and order to use when
// adding effects into an ingestion.
type EffectIngestion struct {
	Dest        *Ingestion
	OperationID int64
	err         error
	added       int
	parent      *Ingestion
}

// LedgerBundle represents a single ledger's worth of novelty created by one
// ledger close
type LedgerBundle struct {
	Sequence        int32
	Header          core.LedgerHeader
	TransactionFees []core.TransactionFee
	Transactions    []core.Transaction
}

// System represents the data ingestion subsystem of horizon.
type System struct {
	// Config allows passing some configuration values to System.
	Config Config
	// HorizonDB is the connection to the horizon database that ingested data will
	// be written to.
	HorizonDB *db.Session
	// CoreDB is the stellar-core db that data is ingested from.
	CoreDB  *db.Session
	Metrics IngesterMetrics
	// Network is the passphrase for the network being imported
	Network string
	// StellarCoreURL is the http endpoint of the stellar-core that data is being
	// ingested from.
	StellarCoreURL string
	// SkipCursorUpdate causes the ingestor to skip
	// reporting the "last imported ledger" cursor to
	// stellar-core
	SkipCursorUpdate bool
	// HistoryRetentionCount is the desired minimum number of ledgers to
	// keep in the history database, working backwards from the latest core
	// ledger.  0 represents "all ledgers".
	HistoryRetentionCount uint
	// IngestFailedTransactions toggles whether to ingest failed transactions
	IngestFailedTransactions bool

	lock    sync.Mutex
	current *Session
}

// IngesterMetrics tracks all the metrics for the ingestion subsystem
type IngesterMetrics struct {
	ClearLedgerTimer  metrics.Timer
	IngestLedgerTimer metrics.Timer
	LoadLedgerTimer   metrics.Timer
}

// BatchInsertBuilder works like sq.InsertBuilder but has a better support for batching
// large number of rows.
type BatchInsertBuilder struct {
	TableName TableName
	Columns   []string

	initOnce      sync.Once
	rows          [][]interface{}
	insertBuilder sq.InsertBuilder
}

// AssetStats tracks and updates all the assets modified during a cycle of ingestion.
type AssetStats struct {
	CoreSession    *db.Session
	HistorySession *db.Session

	batchInsertBuilder *BatchInsertBuilder
	toUpdate           map[string]xdr.Asset
	initOnce           sync.Once
}

// Ingestion receives write requests from a Session
type Ingestion struct {
	// DB is the sql connection to be used for writing any rows into the horizon
	// database.
	DB       *db.Session
	builders map[TableName]*BatchInsertBuilder
}

// Session represents a single attempt at ingesting data into the history
// database.
type Session struct {
	// Config allows passing some configuration values to System.
	Config    Config
	Cursor    *Cursor
	Ingestion *Ingestion
	// Network is the passphrase for the network being imported
	Network string
	// StellarCoreURL is the http endpoint of the stellar-core that data is being
	// ingested from.
	StellarCoreURL string
	// ClearExisting causes the session to clear existing data from the horizon db
	// when the session is run.
	ClearExisting bool
	// SkipCursorUpdate causes the session to skip
	// reporting the "last imported ledger" cursor to
	// stellar-core
	SkipCursorUpdate bool
	// Metrics is a reference to where the session should record its metric information
	Metrics *IngesterMetrics
	// AssetStats calculates asset stats
	AssetStats *AssetStats

	//
	// Results fields
	//

	// Err is the error that caused this session to fail, if any.
	Err error
	// Ingested is the number of ledgers that were successfully ingested during
	// this session.
	Ingested int
}

// New initializes the ingester, causing it to begin polling the stellar-core
// database for now ledgers and ingesting data into the horizon database.
func New(network string, coreURL string, core, horizon *db.Session, config Config) *System {
	i := &System{
		Config:         config,
		Network:        network,
		StellarCoreURL: coreURL,
		HorizonDB:      horizon,
		CoreDB:         core,
	}

	i.Metrics.ClearLedgerTimer = metrics.NewTimer()
	i.Metrics.IngestLedgerTimer = metrics.NewTimer()
	i.Metrics.LoadLedgerTimer = metrics.NewTimer()
	return i
}

// NewCursor initializes a new ingestion cursor
func NewCursor(first, last int32, i *System) *Cursor {
	return &Cursor{
		FirstLedger: first,
		LastLedger:  last,
		CoreDB:      i.CoreDB,
		Name:        i.Config.CursorName,
		Metrics:     &i.Metrics,
	}
}

// NewSession initialize a new ingestion session
func NewSession(i *System) *Session {
	cdb := i.CoreDB.Clone()
	hdb := i.HorizonDB.Clone()

	return &Session{
		Config: i.Config,
		Ingestion: &Ingestion{
			DB: hdb,
		},
		Network:          i.Network,
		StellarCoreURL:   i.StellarCoreURL,
		SkipCursorUpdate: i.SkipCursorUpdate,
		Metrics:          &i.Metrics,
		AssetStats: &AssetStats{
			CoreSession:    cdb,
			HistorySession: hdb,
		},
	}
}
