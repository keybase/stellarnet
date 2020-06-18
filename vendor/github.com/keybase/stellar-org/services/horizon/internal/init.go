package horizon

import (
	"context"
	"net/http"

	"github.com/getsentry/raven-go"
	"github.com/rcrowley/go-metrics"

	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/services/horizon/internal/txsub"
	results "github.com/stellar/go/services/horizon/internal/txsub/results/db"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

func mustNewDBSession(databaseURL string, maxIdle, maxOpen int) *db.Session {
	session, err := db.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("cannot open Horizon DB: %v", err)
	}

	session.DB.SetMaxIdleConns(maxIdle)
	session.DB.SetMaxOpenConns(maxOpen)
	return session
}

func mustInitHorizonDB(app *App) {
	maxIdle := app.config.HorizonDBMaxIdleConnections
	maxOpen := app.config.HorizonDBMaxOpenConnections
	if app.config.Ingest {
		maxIdle -= expingest.MaxDBConnections
		maxOpen -= expingest.MaxDBConnections
		if maxIdle <= 0 {
			log.Fatalf("max idle connections to horizon db must be greater than %d", expingest.MaxDBConnections)
		}
		if maxOpen <= 0 {
			log.Fatalf("max open connections to horizon db must be greater than %d", expingest.MaxDBConnections)
		}
	}

	app.historyQ = &history.Q{mustNewDBSession(
		app.config.DatabaseURL,
		maxIdle,
		maxOpen,
	)}
}

func mustInitCoreDB(app *App) {
	maxIdle := app.config.CoreDBMaxIdleConnections
	maxOpen := app.config.CoreDBMaxOpenConnections
	if app.config.Ingest {
		maxIdle -= expingest.MaxDBConnections
		maxOpen -= expingest.MaxDBConnections
		if maxIdle <= 0 {
			log.Fatalf("max idle connections to stellar-core db must be greater than %d", expingest.MaxDBConnections)
		}
		if maxOpen <= 0 {
			log.Fatalf("max open connections to stellar-core db must be greater than %d", expingest.MaxDBConnections)
		}
	}

	app.coreQ = &core.Q{mustNewDBSession(
		app.config.StellarCoreDatabaseURL,
		maxIdle,
		maxOpen,
	)}
}

func initExpIngester(app *App) {
	var err error
	app.expingester, err = expingest.NewSystem(expingest.Config{
		CoreSession: mustNewDBSession(
			app.config.StellarCoreDatabaseURL, expingest.MaxDBConnections, expingest.MaxDBConnections,
		),
		HistorySession: mustNewDBSession(
			app.config.DatabaseURL, expingest.MaxDBConnections, expingest.MaxDBConnections,
		),
		NetworkPassphrase: app.config.NetworkPassphrase,
		// TODO:
		// Use the first archive for now. We don't have a mechanism to
		// use multiple archives at the same time currently.
		HistoryArchiveURL:        app.config.HistoryArchiveURLs[0],
		StellarCoreURL:           app.config.StellarCoreURL,
		StellarCoreCursor:        app.config.CursorName,
		MaxStreamRetries:         3,
		DisableStateVerification: app.config.IngestDisableStateVerification,
		IngestFailedTransactions: app.config.IngestFailedTransactions,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func initPathFinder(app *App) {
	orderBookGraph := orderbook.NewOrderBookGraph()
	app.orderBookStream = expingest.NewOrderBookStream(
		&history.Q{app.HorizonSession(app.ctx)},
		orderBookGraph,
	)

	app.paths = simplepath.NewInMemoryFinder(orderBookGraph)
}

// initSentry initialized the default sentry client with the configured DSN
func initSentry(app *App) {
	if app.config.SentryDSN == "" {
		return
	}

	log.WithField("dsn", app.config.SentryDSN).Info("Initializing sentry")
	err := raven.SetDSN(app.config.SentryDSN)
	if err != nil {
		log.Fatal(err)
	}
}

// initLogglyLog attaches a loggly hook to our logging system.
func initLogglyLog(app *App) {
	if app.config.LogglyToken == "" {
		return
	}

	log.WithFields(log.F{
		"token": app.config.LogglyToken,
		"tag":   app.config.LogglyTag,
	}).Info("Initializing loggly hook")

	hook := log.NewLogglyHook(app.config.LogglyToken, app.config.LogglyTag)
	log.DefaultLogger.Logger.Hooks.Add(hook)

	go func() {
		<-app.ctx.Done()
		hook.Flush()
	}()
}

func initDbMetrics(app *App) {
	app.historyLatestLedgerGauge = metrics.NewGauge()
	app.historyElderLedgerGauge = metrics.NewGauge()
	app.coreLatestLedgerGauge = metrics.NewGauge()
	app.horizonConnGauge = metrics.NewGauge()
	app.coreConnGauge = metrics.NewGauge()
	app.goroutineGauge = metrics.NewGauge()
	app.metrics.Register("history.latest_ledger", app.historyLatestLedgerGauge)
	app.metrics.Register("history.elder_ledger", app.historyElderLedgerGauge)
	app.metrics.Register("stellar_core.latest_ledger", app.coreLatestLedgerGauge)
	app.metrics.Register("order_book_stream.latest_ledger", app.orderBookStream.LatestLedgerGauge)
	app.metrics.Register("history.open_connections", app.horizonConnGauge)
	app.metrics.Register("stellar_core.open_connections", app.coreConnGauge)
	app.metrics.Register("goroutines", app.goroutineGauge)
}

// initIngestMetrics registers the metrics for the ingestion into the provided
// app's metrics registry.
func initIngestMetrics(app *App) {
	if app.expingester == nil {
		return
	}
	app.metrics.Register("ingest.ledger_ingestion", app.expingester.Metrics.LedgerIngestionTimer)
	app.metrics.Register("ingest.ledger_in_memory_ingestion", app.expingester.Metrics.LedgerInMemoryIngestionTimer)
	app.metrics.Register("ingest.state_verify", app.expingester.Metrics.StateVerifyTimer)
}

func initTxSubMetrics(app *App) {
	app.submitter.Init()
	app.metrics.Register("txsub.buffered", app.submitter.Metrics.BufferedSubmissionsGauge)
	app.metrics.Register("txsub.open", app.submitter.Metrics.OpenSubmissionsGauge)
	app.metrics.Register("txsub.succeeded", app.submitter.Metrics.SuccessfulSubmissionsMeter)
	app.metrics.Register("txsub.failed", app.submitter.Metrics.FailedSubmissionsMeter)
	app.metrics.Register("txsub.v0", app.submitter.Metrics.V0TransactionsMeter)
	app.metrics.Register("txsub.v1", app.submitter.Metrics.V1TransactionsMeter)
	app.metrics.Register("txsub.feebump", app.submitter.Metrics.FeeBumpTransactionsMeter)
	app.metrics.Register("txsub.total", app.submitter.Metrics.SubmissionTimer)
}

// initWebMetrics registers the metrics for the web server into the provided
// app's metrics registry.
func initWebMetrics(app *App) {
	app.metrics.Register("requests.total", app.web.requestTimer)
	app.metrics.Register("requests.succeeded", app.web.successMeter)
	app.metrics.Register("requests.failed", app.web.failureMeter)
}

func initSubmissionSystem(app *App) {
	// Due to a delay between Stellar-Core closing a ledger and Horizon
	// ingesting it, it's possible that results of transaction submitted to
	// Stellar-Core via Horizon may not be immediately visible. This is
	// happening because `txsub` package checks two sources when checking a
	// transaction result: Stellar-Core and Horizon DB.
	//
	// The extreme case is https://github.com/stellar/go/issues/2272 where
	// results of transaction creating an account are not visible: requesting
	// account details in Horizon returns `404 Not Found`:
	//
	// ```
	// 	 Horizon DB                  Core DB                  User
	// 		 |                          |                      |
	// 		 |                          |                      |
	// 		 |                          | <------- Submit create_account op
	// 		 |                          |                      |
	// 		 |                  Insert transaction             |
	// 		 |                          |                      |
	// 		 |                     Tx found -----------------> |
	// 		 |                          |                      |
	// 		 |                          |                      |
	// 		 | <--------------------------------------- Get account info
	// 		 |                          |                      |
	// 		 |                          |                      |
	//         Account NOT found ------------------------------------> |
	// 		 |                          |                      |
	//         Insert account                   |                      |
	// ```
	//
	// To fix this skip checking Stellar-Core DB for transaction results if
	// Horizon is ingesting failed transactions.

	app.submitter = &txsub.System{
		Pending:         txsub.NewDefaultSubmissionList(),
		Submitter:       txsub.NewDefaultSubmitter(http.DefaultClient, app.config.StellarCoreURL),
		SubmissionQueue: sequence.NewManager(),
		Results: &results.DB{
			Core:           &core.Q{Session: app.CoreSession(context.Background())},
			History:        &history.Q{Session: app.HorizonSession(context.Background())},
			SkipCoreChecks: app.config.IngestFailedTransactions,
		},
		Sequences: &history.Q{Session: app.HorizonSession(context.Background())},
	}
}
