package horizon

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stellar/go/clients/stellarcore"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/logmetrics"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/services/horizon/internal/paths"
	"github.com/stellar/go/services/horizon/internal/reap"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/app"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/throttled"
	graceful "gopkg.in/tylerb/graceful.v1"
)

// App represents the root of the state of a horizon instance.
type App struct {
	config                       Config
	web                          *web
	historyQ                     *history.Q
	coreQ                        *core.Q
	ctx                          context.Context
	cancel                       func()
	coreVersion                  string
	horizonVersion               string
	currentProtocolVersion       int32
	coreSupportedProtocolVersion int32
	orderBookStream              *expingest.OrderBookStream
	submitter                    *txsub.System
	paths                        paths.Finder
	expingester                  *expingest.System
	reaper                       *reap.System
	ticks                        *time.Ticker

	// metrics
	metrics                  metrics.Registry
	historyLatestLedgerGauge metrics.Gauge
	historyElderLedgerGauge  metrics.Gauge
	horizonConnGauge         metrics.Gauge
	coreLatestLedgerGauge    metrics.Gauge
	coreConnGauge            metrics.Gauge
	goroutineGauge           metrics.Gauge
}

// NewApp constructs an new App instance from the provided config.
func NewApp(config Config) *App {
	a := &App{
		config:         config,
		horizonVersion: app.Version(),
		ticks:          time.NewTicker(1 * time.Second),
	}

	a.init()
	return a
}

// Serve starts the horizon web server, binding it to a socket, setting up
// the shutdown signals.
func (a *App) Serve() {
	addr := fmt.Sprintf(":%d", a.config.Port)

	srv := &graceful.Server{
		Timeout: 10 * time.Second,

		Server: &http.Server{
			Addr:        addr,
			Handler:     a.web.router,
			ReadTimeout: 5 * time.Second,
		},

		ShutdownInitiated: func() {
			log.Info("received signal, gracefully stopping")
			a.Close()
		},
	}

	log.Infof("Starting horizon on %s (ingest: %v)", addr, a.config.Ingest)

	if a.config.AdminPort != 0 {
		go func() {
			adminAddr := fmt.Sprintf(":%d", a.config.AdminPort)
			log.Infof("Starting internal server on %s", adminAddr)

			internalSrv := &http.Server{
				Addr:        adminAddr,
				Handler:     a.web.internalRouter,
				ReadTimeout: 5 * time.Second,
			}

			err := internalSrv.ListenAndServe()
			if err != nil {
				log.Warn(errors.Wrap(err, "error in internalSrv.ListenAndServe()"))
			}
		}()
	}

	go a.run()
	go a.orderBookStream.Run(a.ctx)

	// WaitGroup for all go routines. Makes sure that DB is closed when
	// all services gracefully shutdown.
	var wg sync.WaitGroup

	if a.expingester != nil {
		wg.Add(1)
		go func() {
			a.expingester.Run()
			wg.Done()
		}()
	}

	var err error
	if a.config.TLSCert != "" {
		err = srv.ListenAndServeTLS(a.config.TLSCert, a.config.TLSKey)
	} else {
		err = srv.ListenAndServe()
	}

	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	a.CloseDB()

	log.Info("stopped")
}

// Close cancels the app. It does not close DB connections - use App.CloseDB().
func (a *App) Close() {
	a.cancel()
	if a.expingester != nil {
		a.expingester.Shutdown()
	}
	a.ticks.Stop()
}

// CloseDB closes DB connections. When using during web server shut down make
// sure all requests are first properly finished to avoid "sql: database is
// closed" errors.
func (a *App) CloseDB() {
	a.historyQ.Session.DB.Close()
	a.coreQ.Session.DB.Close()
}

// HistoryQ returns a helper object for performing sql queries against the
// history portion of horizon's database.
func (a *App) HistoryQ() *history.Q {
	return a.historyQ
}

// HorizonSession returns a new session that loads data from the horizon
// database. The returned session is bound to `ctx`.
func (a *App) HorizonSession(ctx context.Context) *db.Session {
	return &db.Session{DB: a.historyQ.Session.DB, Ctx: ctx}
}

// CoreSession returns a new session that loads data from the stellar core
// database. The returned session is bound to `ctx`.
func (a *App) CoreSession(ctx context.Context) *db.Session {
	return &db.Session{DB: a.coreQ.Session.DB, Ctx: ctx}
}

// CoreQ returns a helper object for performing sql queries aginst the
// stellar core database.
func (a *App) CoreQ() *core.Q {
	return a.coreQ
}

// IsHistoryStale returns true if the latest history ledger is more than
// `StaleThreshold` ledgers behind the latest core ledger
func (a *App) IsHistoryStale() bool {
	return isHistoryStale(a.config.StaleThreshold)
}

func isHistoryStale(staleThreshold uint) bool {
	if staleThreshold == 0 {
		return false
	}

	ls := ledger.CurrentState()
	return (ls.CoreLatest - ls.HistoryLatest) > int32(staleThreshold)
}

// UpdateLedgerState triggers a refresh of several metrics gauges, such as open
// db connections and ledger state
func (a *App) UpdateLedgerState() {
	var next ledger.State

	logErr := func(err error, msg string) {
		log.WithStack(err).WithField("err", err.Error()).Error(msg)
	}

	err := a.CoreQ().LatestLedger(&next.CoreLatest)
	if err != nil {
		logErr(err, "failed to load the latest known ledger state from core DB")
		return
	}

	err = a.HistoryQ().LatestLedger(&next.HistoryLatest)
	if err != nil {
		logErr(err, "failed to load the latest known ledger state from history DB")
		return
	}

	err = a.HistoryQ().ElderLedger(&next.HistoryElder)
	if err != nil {
		logErr(err, "failed to load the oldest known ledger state from history DB")
		return
	}

	next.ExpHistoryLatest, err = a.HistoryQ().GetLastLedgerExpIngestNonBlocking()
	if err != nil {
		logErr(err, "failed to load the oldest known exp ledger state from history DB")
		return
	}

	ledger.SetState(next)
}

// UpdateFeeStatsState triggers a refresh of several operation fee metrics.
func (a *App) UpdateFeeStatsState() {
	var (
		next          operationfeestats.State
		latest        history.LatestLedger
		feeStats      history.FeeStats
		capacityStats history.LedgerCapacityUsageStats
	)

	logErr := func(err error, msg string) {
		// If DB is empty ignore the error
		if errors.Cause(err) == sql.ErrNoRows {
			return
		}

		log.WithStack(err).WithField("err", err.Error()).Error(msg)
	}

	cur, ok := operationfeestats.CurrentState()

	err := a.HistoryQ().LatestLedgerBaseFeeAndSequence(&latest)
	if err != nil {
		logErr(err, "failed to load the latest known ledger's base fee and sequence number")
		return
	}

	// finish early if no new ledgers
	if ok && cur.LastLedger == uint32(latest.Sequence) {
		return
	}

	next.LastBaseFee = int64(latest.BaseFee)
	next.LastLedger = uint32(latest.Sequence)

	err = a.HistoryQ().FeeStats(latest.Sequence, &feeStats)
	if err != nil {
		logErr(err, "failed to load operation fee stats")
		return
	}

	err = a.HistoryQ().LedgerCapacityUsageStats(latest.Sequence, &capacityStats)
	if err != nil {
		logErr(err, "failed to load ledger capacity usage stats")
		return
	}

	next.LedgerCapacityUsage = capacityStats.CapacityUsage.String

	// if no transactions in last 5 ledgers, return
	// latest ledger's base fee for all
	if !feeStats.MaxFeeMode.Valid && !feeStats.MaxFeeMin.Valid {
		// MaxFee
		next.MaxFeeMax = next.LastBaseFee
		next.MaxFeeMin = next.LastBaseFee
		next.MaxFeeMode = next.LastBaseFee
		next.MaxFeeP10 = next.LastBaseFee
		next.MaxFeeP20 = next.LastBaseFee
		next.MaxFeeP30 = next.LastBaseFee
		next.MaxFeeP40 = next.LastBaseFee
		next.MaxFeeP50 = next.LastBaseFee
		next.MaxFeeP60 = next.LastBaseFee
		next.MaxFeeP70 = next.LastBaseFee
		next.MaxFeeP80 = next.LastBaseFee
		next.MaxFeeP90 = next.LastBaseFee
		next.MaxFeeP95 = next.LastBaseFee
		next.MaxFeeP99 = next.LastBaseFee

		// FeeCharged
		next.FeeChargedMax = next.LastBaseFee
		next.FeeChargedMin = next.LastBaseFee
		next.FeeChargedMode = next.LastBaseFee
		next.FeeChargedP10 = next.LastBaseFee
		next.FeeChargedP20 = next.LastBaseFee
		next.FeeChargedP30 = next.LastBaseFee
		next.FeeChargedP40 = next.LastBaseFee
		next.FeeChargedP50 = next.LastBaseFee
		next.FeeChargedP60 = next.LastBaseFee
		next.FeeChargedP70 = next.LastBaseFee
		next.FeeChargedP80 = next.LastBaseFee
		next.FeeChargedP90 = next.LastBaseFee
		next.FeeChargedP95 = next.LastBaseFee
		next.FeeChargedP99 = next.LastBaseFee

	} else {
		// MaxFee
		next.MaxFeeMax = feeStats.MaxFeeMax.Int64
		next.MaxFeeMin = feeStats.MaxFeeMin.Int64
		next.MaxFeeMode = feeStats.MaxFeeMode.Int64
		next.MaxFeeP10 = feeStats.MaxFeeP10.Int64
		next.MaxFeeP20 = feeStats.MaxFeeP20.Int64
		next.MaxFeeP30 = feeStats.MaxFeeP30.Int64
		next.MaxFeeP40 = feeStats.MaxFeeP40.Int64
		next.MaxFeeP50 = feeStats.MaxFeeP50.Int64
		next.MaxFeeP60 = feeStats.MaxFeeP60.Int64
		next.MaxFeeP70 = feeStats.MaxFeeP70.Int64
		next.MaxFeeP80 = feeStats.MaxFeeP80.Int64
		next.MaxFeeP90 = feeStats.MaxFeeP90.Int64
		next.MaxFeeP95 = feeStats.MaxFeeP95.Int64
		next.MaxFeeP99 = feeStats.MaxFeeP99.Int64

		// FeeCharged
		next.FeeChargedMax = feeStats.FeeChargedMax.Int64
		next.FeeChargedMin = feeStats.FeeChargedMin.Int64
		next.FeeChargedMode = feeStats.FeeChargedMode.Int64
		next.FeeChargedP10 = feeStats.FeeChargedP10.Int64
		next.FeeChargedP20 = feeStats.FeeChargedP20.Int64
		next.FeeChargedP30 = feeStats.FeeChargedP30.Int64
		next.FeeChargedP40 = feeStats.FeeChargedP40.Int64
		next.FeeChargedP50 = feeStats.FeeChargedP50.Int64
		next.FeeChargedP60 = feeStats.FeeChargedP60.Int64
		next.FeeChargedP70 = feeStats.FeeChargedP70.Int64
		next.FeeChargedP80 = feeStats.FeeChargedP80.Int64
		next.FeeChargedP90 = feeStats.FeeChargedP90.Int64
		next.FeeChargedP95 = feeStats.FeeChargedP95.Int64
		next.FeeChargedP99 = feeStats.FeeChargedP99.Int64
	}

	operationfeestats.SetState(next)
}

// UpdateStellarCoreInfo updates the value of coreVersion,
// currentProtocolVersion, and coreSupportedProtocolVersion from the Stellar
// core API.
func (a *App) UpdateStellarCoreInfo() {
	if a.config.StellarCoreURL == "" {
		return
	}

	core := &stellarcore.Client{
		URL: a.config.StellarCoreURL,
	}

	resp, err := core.Info(context.Background())
	if err != nil {
		log.Warnf("could not load stellar-core info: %s", err)
		return
	}

	// Check if NetworkPassphrase is different, if so exit Horizon as it can break the
	// state of the application.
	if resp.Info.Network != a.config.NetworkPassphrase {
		log.Errorf(
			"Network passphrase of stellar-core (%s) does not match Horizon configuration (%s). Exiting...",
			resp.Info.Network,
			a.config.NetworkPassphrase,
		)
		os.Exit(1)
	}

	a.coreVersion = resp.Info.Build
	a.currentProtocolVersion = int32(resp.Info.Ledger.Version)
	a.coreSupportedProtocolVersion = int32(resp.Info.ProtocolVersion)
}

// UpdateMetrics triggers a refresh of several metrics gauges, such as open
// db connections and ledger state
func (a *App) UpdateMetrics() {
	a.goroutineGauge.Update(int64(runtime.NumGoroutine()))
	ls := ledger.CurrentState()
	a.historyLatestLedgerGauge.Update(int64(ls.HistoryLatest))
	a.historyElderLedgerGauge.Update(int64(ls.HistoryElder))
	a.coreLatestLedgerGauge.Update(int64(ls.CoreLatest))

	a.horizonConnGauge.Update(int64(a.historyQ.Session.DB.Stats().OpenConnections))
	a.coreConnGauge.Update(int64(a.coreQ.Session.DB.Stats().OpenConnections))
}

// DeleteUnretainedHistory forwards to the app's reaper.  See
// `reap.DeleteUnretainedHistory` for details
func (a *App) DeleteUnretainedHistory() error {
	return a.reaper.DeleteUnretainedHistory()
}

// Tick triggers horizon to update all of it's background processes such as
// transaction submission, metrics, ingestion and reaping.
func (a *App) Tick() {
	var wg sync.WaitGroup
	log.Debug("ticking app")
	// update ledger state, operation fee state, and stellar-core info in parallel
	wg.Add(3)
	go func() { a.UpdateLedgerState(); wg.Done() }()
	go func() { a.UpdateFeeStatsState(); wg.Done() }()
	go func() { a.UpdateStellarCoreInfo(); wg.Done() }()
	wg.Wait()

	wg.Add(2)
	go func() { a.reaper.Tick(); wg.Done() }()
	go func() { a.submitter.Tick(a.ctx); wg.Done() }()
	wg.Wait()

	// finally, update metrics
	a.UpdateMetrics()
	log.Debug("finished ticking app")
}

// Init initializes app, using the config to populate db connections and
// whatnot.
func (a *App) init() {
	// app-context
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// log
	log.DefaultLogger.Logger.Level = a.config.LogLevel
	log.DefaultLogger.Logger.Hooks.Add(logmetrics.DefaultMetrics)

	// sentry
	initSentry(a)

	// loggly
	initLogglyLog(a)

	// stellarCoreInfo
	a.UpdateStellarCoreInfo()

	// horizon-db and core-db
	mustInitHorizonDB(a)
	mustInitCoreDB(a)

	if a.config.Ingest {
		// expingester
		initExpIngester(a)
	}
	initPathFinder(a)

	// txsub
	initSubmissionSystem(a)

	// reaper
	a.reaper = reap.New(a.config.HistoryRetentionCount, a.HorizonSession(context.Background()))

	// web.init
	a.web = mustInitWeb(a.ctx, a.historyQ, a.config.SSEUpdateFrequency, a.config.StaleThreshold, a.config.IngestFailedTransactions)

	// web.rate-limiter
	a.web.rateLimiter = maybeInitWebRateLimiter(a.config.RateQuota)

	// web.middleware
	// Note that we passed in `a` here for putting the whole App in the context.
	// This parameter will be removed soon.
	a.web.mustInstallMiddlewares(a, a.config.ConnectionTimeout)

	// web.actions
	a.web.mustInstallActions(a.config, a.paths, a.historyQ.Session, a.metrics)

	// metrics and log.metrics
	a.metrics = metrics.NewRegistry()
	for level, meter := range *logmetrics.DefaultMetrics {
		a.metrics.Register(fmt.Sprintf("logging.%s", level), meter)
	}

	// db-metrics
	initDbMetrics(a)

	// ingest.metrics
	initIngestMetrics(a)

	// web.metrics
	initWebMetrics(a)

	// txsub.metrics
	initTxSubMetrics(a)
}

// run is the function that runs in the background that triggers Tick each
// second
func (a *App) run() {
	for {
		select {
		case <-a.ticks.C:
			a.Tick()
		case <-a.ctx.Done():
			log.Info("finished background ticker")
			return
		}
	}
}

// withAppContext create a context on from the App type.
func withAppContext(ctx context.Context, a *App) context.Context {
	return context.WithValue(ctx, &horizonContext.AppContextKey, a)
}

// GetRateLimiter returns the HTTPRateLimiter of the App.
func (a *App) GetRateLimiter() *throttled.HTTPRateLimiter {
	return a.web.rateLimiter
}

// AppFromContext returns the set app, if one has been set, from the
// provided context returns nil if no app has been set.
func AppFromContext(ctx context.Context) *App {
	if ctx == nil {
		return nil
	}

	val, _ := ctx.Value(&horizonContext.AppContextKey).(*App)
	return val
}
