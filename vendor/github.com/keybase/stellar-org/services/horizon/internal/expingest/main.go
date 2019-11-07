// Package expingest contains the new ingestion system for horizon.
// It currently runs completely independent of the old one, that means
// that the new system can be ledgers behind/ahead the old system.
package expingest

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/stellar/go/clients/stellarcore"
	"github.com/stellar/go/exp/ingest"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/ledgerbackend"
	"github.com/stellar/go/exp/orderbook"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/historyarchive"
	ilog "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

const (
	// CurrentVersion reflects the latest version of the ingestion
	// algorithm. This value is stored in KV store and is used to decide
	// if there's a need to reprocess the ledger state or reingest data.
	//
	// Version history:
	// - 1: Initial version
	// - 2: We added the orderbook, offers processors and distributed
	//      ingestion.
	// - 3: Fixes a bug that could potentialy result in invalid state
	//      (#1722). Update the version to clear the state.
	// - 4: Fixes a bug in AccountSignersChanged method.
	CurrentVersion = 4
)

var log = ilog.DefaultLogger.WithField("service", "expingest")

type Config struct {
	CoreSession    *db.Session
	StellarCoreURL string

	HistorySession           *db.Session
	HistoryArchiveURL        string
	TempSet                  io.TempSet
	DisableStateVerification bool

	OrderBookGraph *orderbook.OrderBookGraph
}

type dbQ interface {
	Begin() error
	Rollback() error
	GetLastLedgerExpIngest() (uint32, error)
	GetExpIngestVersion() (int, error)
	UpdateLastLedgerExpIngest(uint32) error
	UpdateExpStateInvalid(bool) error
	GetExpStateInvalid() (bool, error)
	GetAllOffers() ([]history.Offer, error)
}

type dbSession interface {
	TruncateTables([]string) error
	Clone() *db.Session
}

type liveSession interface {
	Run() error
	GetArchive() historyarchive.ArchiveInterface
	Resume(ledgerSequence uint32) error
	GetLatestSuccessfullyProcessedLedger() (ledgerSequence uint32, processed bool)
	Shutdown()
}

type retry interface {
	onError(func() error)
}

type System struct {
	session        liveSession
	historyQ       dbQ
	historySession dbSession
	graph          *orderbook.OrderBookGraph
	retry          retry

	// stateVerificationRunning is true when verification routine is currently
	// running.
	stateVerificationMutex   sync.Mutex
	stateVerificationRunning bool
	disableStateVerification bool
}

type alwaysRetry struct {
	backOff time.Duration
}

func (r alwaysRetry) onError(f func() error) {
	for {
		err := f()
		if err != nil {
			log.Error(err)
			time.Sleep(r.backOff)
			continue
		}
		break
	}
}

func NewSystem(config Config) (*System, error) {
	archive, err := createArchive(config.HistoryArchiveURL)
	if err != nil {
		return nil, errors.Wrap(err, "error creating history archive")
	}

	ledgerBackend, err := ledgerbackend.NewDatabaseBackendFromSession(config.CoreSession)
	if err != nil {
		return nil, errors.Wrap(err, "error creating ledger backend")
	}

	historyQ := &history.Q{config.HistorySession}

	session := &ingest.LiveSession{
		Archive:        archive,
		LedgerBackend:  ledgerBackend,
		StatePipeline:  buildStatePipeline(historyQ, config.OrderBookGraph),
		LedgerPipeline: buildLedgerPipeline(historyQ, config.OrderBookGraph),
		StellarCoreClient: &stellarcore.Client{
			URL: config.StellarCoreURL,
		},

		StateReporter:  &LoggingStateReporter{Log: log, Interval: 100000},
		LedgerReporter: &LoggingLedgerReporter{Log: log},

		TempSet: config.TempSet,
	}

	system := &System{
		session:                  session,
		historySession:           config.HistorySession,
		historyQ:                 historyQ,
		graph:                    config.OrderBookGraph,
		retry:                    alwaysRetry{time.Second},
		disableStateVerification: config.DisableStateVerification,
	}

	addPipelineHooks(
		system,
		session.StatePipeline,
		config.HistorySession,
		session,
		config.OrderBookGraph,
	)
	addPipelineHooks(
		system,
		session.LedgerPipeline,
		config.HistorySession,
		session,
		config.OrderBookGraph,
	)

	return system, nil
}

// Run starts ingestion system. Ingestion system supports distributed ingestion
// that means that Horizon ingestion can be running on multiple machines and
// only one, random node will lead the ingestion.
//
// It needs to support cartesian product of the following run scenarios cases:
// - Init from empty state (1a) and resuming from existing state (1b).
// - Ingestion system version has been upgraded (2a) or not (2b).
// - Current node is leading ingestion (3a) or not (3b).
//
// We always clear state when ingestion system is upgraded so 2a and 2b are
// included in 1a.
//
// We ensure that only one instance is a leader because in each round instances
// try to acquire a lock on `LastLedgerExpIngest value in key value store and only
// one instance will be able to acquire it. This happens in both initial processing
// and ledger processing. So this solves 3a and 3b in both 1a and 1b.
//
// Finally, 1a and 1b are tricky because we need to keep the latest version
// of order book graph in memory of each Horizon instance. To solve this:
// * For state init:
//   * If instance is a leader, we update the order book graph by running state
//     pipeline normally.
//   * If instance is NOT a leader, we build a graph from offers present in a
//     database. We completely omit state pipeline in this case.
// * For resuming:
//   * If instances is a leader, it runs full ledger pipeline, including updating
//     a database.
//   * If instances is a NOT leader, it runs ledger pipeline without updating a
//     a database so order book graph is updated but database is not overwritten.
func (s *System) Run() {
	// Expingest is an experimental package so we don't want entire Horizon app
	// to crash in case of unexpected errors.
	// TODO: This should be removed when expingest is no longer experimental.
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(ilog.F{
				"err":   r,
				"stack": string(debug.Stack()),
			}).Error("expingest panic")
		}
	}()

	// retryOnError loop is needed only in case of initial state sync errors.
	// If the state is successfully ingested `resumeFromLedger` method continues
	// processing ledgers.
	s.retry.onError(func() error {
		// Transaction will be commited or rolled back in pipelines post hooks.
		err := s.historyQ.Begin()
		if err != nil {
			return errors.Wrap(err, "Error starting a transaction")
		}
		// We rollback in pipelines post-hooks but the error can happen before
		// pipeline starts processing.
		defer s.historyQ.Rollback()

		// This will get the value `FOR UPDATE`, blocking it for other nodes.
		lastIngestedLedger, err := s.historyQ.GetLastLedgerExpIngest()
		if err != nil {
			return errors.Wrap(err, "Error getting last ingested ledger")
		}

		ingestVersion, err := s.historyQ.GetExpIngestVersion()
		if err != nil {
			return errors.Wrap(err, "Error getting exp ingest version")
		}

		if ingestVersion != CurrentVersion || lastIngestedLedger == 0 {
			// This block is either starting from empty state or ingestion
			// version upgrade.
			// This will always run on a single instance due to the fact that
			// `LastLedgerExpIngest` value is blocked for update and will always
			// be updated when leading instance finishes processing state.
			// In case of errors it will start `Run` from the beginning.
			log.Info("Starting ingestion system from empty state...")

			// Clear last_ingested_ledger in key value store
			if err = s.historyQ.UpdateLastLedgerExpIngest(0); err != nil {
				return errors.Wrap(err, "Error updating last ingested ledger")
			}

			// Clear invalid state in key value store. It's possible that upgraded
			// ingestion is fixing it.
			if err = s.historyQ.UpdateExpStateInvalid(false); err != nil {
				return errors.Wrap(err, "Error updating state invalid value")
			}

			err = s.historySession.TruncateTables(
				history.ExperimentalIngestionTables,
			)
			if err != nil {
				return errors.Wrap(err, "Error clearing ingest tables")
			}

			err = s.session.Run()
			if err != nil {
				// Check if session processed a state, if so, continue since the
				// last processed ledger, otherwise start over.
				var processed bool
				lastIngestedLedger, processed = s.session.GetLatestSuccessfullyProcessedLedger()
				if !processed {
					return err
				}

				log.WithFields(ilog.F{
					"err":                  err,
					"last_ingested_ledger": lastIngestedLedger,
				}).Error("Error running session, resuming from the last ingested ledger")
			}
		} else {
			// The other node already ingested a state (just now or in the past)
			// so we need to get offers from a DB, then resume session normally.
			// State pipeline is NOT processed.
			log.WithField("last_ledger", lastIngestedLedger).
				Info("Resuming ingestion system from last processed ledger...")

			err = loadOrderBookGraphFromDB(s.historyQ, s.graph)
			if err != nil {
				return errors.Wrap(err, "Error loading order book graph from db")
			}
		}

		s.resumeFromLedger(lastIngestedLedger)
		return nil
	})
}

func loadOrderBookGraphFromDB(historyQ dbQ, graph *orderbook.OrderBookGraph) error {
	defer graph.Discard()

	log.Info("Loading offers from a database into memory store...")
	start := time.Now()

	offers, err := historyQ.GetAllOffers()
	if err != nil {
		return err
	}

	for _, offer := range offers {
		sellerID := xdr.MustAddress(offer.SellerID)
		graph.AddOffer(xdr.OfferEntry{
			SellerId: sellerID,
			OfferId:  offer.OfferID,
			Selling:  offer.SellingAsset,
			Buying:   offer.BuyingAsset,
			Amount:   offer.Amount,
			Price: xdr.Price{
				N: xdr.Int32(offer.Pricen),
				D: xdr.Int32(offer.Priced),
			},
			Flags: xdr.Uint32(offer.Flags),
		})
	}

	err = graph.Apply()
	if err == nil {
		log.WithField(
			"duration",
			time.Since(start).Seconds(),
		).Info("Finished loading offers from a database into memory store")
	}

	return err
}

func (s *System) resumeFromLedger(lastIngestedLedger uint32) {
	s.retry.onError(func() error {
		err := s.session.Resume(lastIngestedLedger + 1)
		if err != nil {
			// If no ledgers processed so far, try again with the
			// lastIngestedLedger+1 (do nothing).
			// Otherwise, set lastIngestedLedger to the last successfully
			// ingested ledger in the session.
			sessionLastLedger, processed := s.session.GetLatestSuccessfullyProcessedLedger()
			if processed {
				lastIngestedLedger = sessionLastLedger
			}
			return errors.Wrap(err, "Error returned from ingest.LiveSession")
		}

		log.Info("Session shut down")
		return nil
	})
}

func (s *System) Shutdown() {
	log.Info("Shutting down ingestion system...")
	s.session.Shutdown()
}

func createArchive(archiveURL string) (*historyarchive.Archive, error) {
	return historyarchive.Connect(
		archiveURL,
		historyarchive.ConnectOptions{},
	)
}
