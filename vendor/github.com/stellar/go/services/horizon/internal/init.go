package horizon

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/services/horizon/internal/txsub/sequence"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

func mustNewDBSession(subservice db.Subservice, databaseURL string, maxIdle, maxOpen int, registry *prometheus.Registry) db.SessionInterface {
	session, err := db.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("cannot open %v DB: %v", subservice, err)
	}

	session.DB.SetMaxIdleConns(maxIdle)
	session.DB.SetMaxOpenConns(maxOpen)
	return db.RegisterMetrics(session, "horizon", subservice, registry)
}

func mustInitHorizonDB(app *App) {
	maxIdle := app.config.HorizonDBMaxIdleConnections
	maxOpen := app.config.HorizonDBMaxOpenConnections
	if app.config.Ingest {
		maxIdle -= ingest.MaxDBConnections
		maxOpen -= ingest.MaxDBConnections
		if maxIdle <= 0 {
			log.Fatalf("max idle connections to horizon db must be greater than %d", ingest.MaxDBConnections)
		}
		if maxOpen <= 0 {
			log.Fatalf("max open connections to horizon db must be greater than %d", ingest.MaxDBConnections)
		}
	}

	if app.config.RoDatabaseURL == "" {
		app.historyQ = &history.Q{mustNewDBSession(
			db.HistorySubservice,
			app.config.DatabaseURL,
			maxIdle,
			maxOpen,
			app.prometheusRegistry,
		)}
	} else {
		// If RO set, use it for all DB queries
		app.historyQ = &history.Q{mustNewDBSession(
			db.HistorySubservice,
			app.config.RoDatabaseURL,
			maxIdle,
			maxOpen,
			app.prometheusRegistry,
		)}

		app.primaryHistoryQ = &history.Q{mustNewDBSession(
			db.HistoryPrimarySubservice,
			app.config.DatabaseURL,
			maxIdle,
			maxOpen,
			app.prometheusRegistry,
		)}
	}
}

func initIngester(app *App) {
	var err error
	var coreSession db.SessionInterface
	if !app.config.EnableCaptiveCoreIngestion {
		coreSession = mustNewDBSession(
			db.CoreSubservice, app.config.StellarCoreDatabaseURL, ingest.MaxDBConnections, ingest.MaxDBConnections, app.prometheusRegistry)
	}
	app.ingester, err = ingest.NewSystem(ingest.Config{
		CoreSession: coreSession,
		HistorySession: mustNewDBSession(
			db.IngestSubservice, app.config.DatabaseURL, ingest.MaxDBConnections, ingest.MaxDBConnections, app.prometheusRegistry,
		),
		NetworkPassphrase: app.config.NetworkPassphrase,
		// TODO:
		// Use the first archive for now. We don't have a mechanism to
		// use multiple archives at the same time currently.
		HistoryArchiveURL:            app.config.HistoryArchiveURLs[0],
		CheckpointFrequency:          app.config.CheckpointFrequency,
		StellarCoreURL:               app.config.StellarCoreURL,
		StellarCoreCursor:            app.config.CursorName,
		CaptiveCoreBinaryPath:        app.config.CaptiveCoreBinaryPath,
		CaptiveCoreStoragePath:       app.config.CaptiveCoreStoragePath,
		CaptiveCoreToml:              app.config.CaptiveCoreToml,
		RemoteCaptiveCoreURL:         app.config.RemoteCaptiveCoreURL,
		EnableCaptiveCore:            app.config.EnableCaptiveCoreIngestion,
		DisableStateVerification:     app.config.IngestDisableStateVerification,
		EnableExtendedLogLedgerStats: app.config.IngestEnableExtendedLogLedgerStats,
	})

	if err != nil {
		log.Fatal(err)
	}
}

func initPathFinder(app *App) {
	orderBookGraph := orderbook.NewOrderBookGraph()
	app.orderBookStream = ingest.NewOrderBookStream(
		&history.Q{app.HorizonSession()},
		orderBookGraph,
	)

	app.paths = simplepath.NewInMemoryFinder(orderBookGraph, !app.config.DisablePoolPathFinding)
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
	log.DefaultLogger.AddHook(hook)

	go func() {
		<-app.ctx.Done()
		hook.Flush()
	}()
}

func initDbMetrics(app *App) {
	app.buildInfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "build", Name: "info"},
		[]string{"version", "goversion"},
	)
	app.prometheusRegistry.MustRegister(app.buildInfoGauge)
	app.buildInfoGauge.With(prometheus.Labels{
		"version":   app.horizonVersion,
		"goversion": runtime.Version(),
	}).Inc()

	app.ingestingGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{Namespace: "horizon", Subsystem: "ingest", Name: "enabled"},
	)
	app.prometheusRegistry.MustRegister(app.ingestingGauge)

	app.historyLatestLedgerCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{Namespace: "horizon", Subsystem: "history", Name: "latest_ledger"},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return float64(ls.HistoryLatest)
		},
	)
	app.prometheusRegistry.MustRegister(app.historyLatestLedgerCounter)

	app.historyLatestLedgerClosedAgoGauge = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "history", Name: "latest_ledger_closed_ago_seconds",
			Help: "seconds since the close of the last ingested ledger",
		},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return time.Since(ls.HistoryLatestClosedAt).Seconds()
		},
	)
	app.prometheusRegistry.MustRegister(app.historyLatestLedgerClosedAgoGauge)

	app.historyElderLedgerCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{Namespace: "horizon", Subsystem: "history", Name: "elder_ledger"},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return float64(ls.HistoryElder)
		},
	)
	app.prometheusRegistry.MustRegister(app.historyElderLedgerCounter)

	app.coreLatestLedgerCounter = prometheus.NewCounterFunc(
		prometheus.CounterOpts{Namespace: "horizon", Subsystem: "stellar_core", Name: "latest_ledger"},
		func() float64 {
			ls := app.ledgerState.CurrentStatus()
			return float64(ls.CoreLatest)
		},
	)
	app.prometheusRegistry.MustRegister(app.coreLatestLedgerCounter)

	app.coreSynced = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "stellar_core", Name: "synced",
			Help: "determines if Stellar-Core defined by --stellar-core-url is synced with the network",
		},
		func() float64 {
			if app.coreState.Get().Synced {
				return 1
			} else {
				return 0
			}
		},
	)
	app.prometheusRegistry.MustRegister(app.coreSynced)

	app.coreSupportedProtocolVersion = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "horizon", Subsystem: "stellar_core", Name: "supported_protocol_version",
			Help: "determines the supported version of the protocol by Stellar-Core defined by --stellar-core-url",
		},
		func() float64 {
			return float64(app.coreState.Get().CoreSupportedProtocolVersion)
		},
	)
	app.prometheusRegistry.MustRegister(app.coreSupportedProtocolVersion)

	app.prometheusRegistry.MustRegister(app.orderBookStream.LatestLedgerGauge)
}

// initGoMetrics registers the Go collector provided by prometheus package which
// includes Go-related metrics.
func initGoMetrics(app *App) {
	app.prometheusRegistry.MustRegister(prometheus.NewGoCollector())
}

// initProcessMetrics registers the process collector provided by prometheus
// package. This is only available on operating systems with a Linux-style proc
// filesystem and on Microsoft Windows.
func initProcessMetrics(app *App) {
	app.prometheusRegistry.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
}

// initIngestMetrics registers the metrics for the ingestion into the provided
// app's metrics registry.
func initIngestMetrics(app *App) {
	if app.ingester == nil {
		return
	}

	app.ingestingGauge.Inc()
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().MaxSupportedProtocolVersion)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LocalLatestLedger)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LedgerIngestionDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LedgerIngestionTradeAggregationDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().StateVerifyDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().StateInvalidGauge)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LedgerStatsCounter)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().ProcessorsRunDuration)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().ProcessorsRunDurationSummary)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().CaptiveStellarCoreSynced)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().CaptiveCoreSupportedProtocolVersion)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().LedgerFetchDurationSummary)
	app.prometheusRegistry.MustRegister(app.ingester.Metrics().StateVerifyLedgerEntriesCount)
}

func initTxSubMetrics(app *App) {
	app.submitter.Init()
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.SubmissionDuration)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.BufferedSubmissionsGauge)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.OpenSubmissionsGauge)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.FailedSubmissionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.SuccessfulSubmissionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.V0TransactionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.V1TransactionsCounter)
	app.prometheusRegistry.MustRegister(app.submitter.Metrics.FeeBumpTransactionsCounter)
}

func initWebMetrics(app *App) {
	app.prometheusRegistry.MustRegister(app.webServer.Metrics.RequestDurationSummary)
	app.prometheusRegistry.MustRegister(app.webServer.Metrics.ReplicaLagErrorsCounter)
}

func initSubmissionSystem(app *App) {
	app.submitter = &txsub.System{
		Pending:         txsub.NewDefaultSubmissionList(),
		Submitter:       txsub.NewDefaultSubmitter(http.DefaultClient, app.config.StellarCoreURL),
		SubmissionQueue: sequence.NewManager(),
		DB: func(ctx context.Context) txsub.HorizonDB {
			return &history.Q{SessionInterface: app.HorizonSession()}
		},
	}
}
