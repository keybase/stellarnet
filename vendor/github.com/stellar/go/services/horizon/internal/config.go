package horizon

import (
	"net/url"
	"time"

	"github.com/stellar/go/ingest/ledgerbackend"

	"github.com/sirupsen/logrus"
	"github.com/stellar/throttled"
)

// Config is the configuration for horizon.  It gets populated by the
// app's main function and is provided to NewApp.
type Config struct {
	DatabaseURL        string
	RoDatabaseURL      string
	HistoryArchiveURLs []string
	Port               uint
	AdminPort          uint

	EnableCaptiveCoreIngestion  bool
	UsingDefaultPubnetConfig    bool
	CaptiveCoreBinaryPath       string
	RemoteCaptiveCoreURL        string
	CaptiveCoreConfigPath       string
	CaptiveCoreTomlParams       ledgerbackend.CaptiveCoreTomlParams
	CaptiveCoreToml             *ledgerbackend.CaptiveCoreToml
	CaptiveCoreStoragePath      string
	CaptiveCoreReuseStoragePath bool

	StellarCoreDatabaseURL string
	StellarCoreURL         string

	// MaxDBConnections has a priority over all 4 values below.
	MaxDBConnections            int
	HorizonDBMaxOpenConnections int
	HorizonDBMaxIdleConnections int

	SSEUpdateFrequency time.Duration
	ConnectionTimeout  time.Duration
	RateQuota          *throttled.RateQuota
	FriendbotURL       *url.URL
	LogLevel           logrus.Level
	LogFile            string

	// MaxPathLength is the maximum length of the path returned by `/paths` endpoint.
	MaxPathLength uint
	// MaxAssetsPerPathRequest is the maximum number of assets considered for `/paths/strict-send` and `/paths/strict-recieve`
	MaxAssetsPerPathRequest int
	DisablePoolPathFinding  bool

	NetworkPassphrase string
	SentryDSN         string
	LogglyToken       string
	LogglyTag         string
	// TLSCert is a path to a certificate file to use for horizon's TLS config
	TLSCert string
	// TLSKey is the path to a private key file to use for horizon's TLS config
	TLSKey string
	// Ingest toggles whether this horizon instance should run the data ingestion subsystem.
	Ingest bool
	// CursorName is the cursor used for ingesting from stellar-core.
	// Setting multiple cursors in different Horizon instances allows multiple
	// Horizons to ingest from the same stellar-core instance without cursor
	// collisions.
	CursorName string
	// HistoryRetentionCount represents the minimum number of ledgers worth of
	// history data to retain in the horizon database. For the purposes of
	// determining a "retention duration", each ledger roughly corresponds to 10
	// seconds of real time.
	HistoryRetentionCount uint
	// StaleThreshold represents the number of ledgers a history database may be
	// out-of-date by before horizon begins to respond with an error to history
	// requests.
	StaleThreshold uint
	// SkipCursorUpdate causes the ingestor to skip reporting the "last imported
	// ledger" state to stellar-core.
	SkipCursorUpdate bool
	// IngestDisableStateVerification disables state verification
	// `System.verifyState()` when set to `true`.
	IngestDisableStateVerification bool
	// IngestEnableExtendedLogLedgerStats enables extended ledger stats in
	// logging.
	IngestEnableExtendedLogLedgerStats bool
	// ApplyMigrations will apply pending migrations to the horizon database
	// before starting the horizon service
	ApplyMigrations bool
	// CheckpointFrequency establishes how many ledgers exist between checkpoints
	CheckpointFrequency uint32
	// BehindCloudflare determines if Horizon instance is behind Cloudflare. In
	// such case http.Request.RemoteAddr will be replaced with Cloudflare header.
	BehindCloudflare bool
	// BehindAWSLoadBalancer determines if Horizon instance is behind AWS load
	// balances like ELB or ALB. In such case http.Request.RemoteAddr will be
	// replaced with the last IP in X-Forwarded-For header.
	BehindAWSLoadBalancer bool
}
