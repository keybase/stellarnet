package horizon

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/stellar/go/services/horizon/internal/actions"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/errors"
	"github.com/stellar/go/services/horizon/internal/expingest"
	"github.com/stellar/go/services/horizon/internal/hchi"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/support/db"
	supportErrors "github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/problem"
)

// appContextMiddleware adds the "app" context into every request, so that subsequence appContextMiddleware
// or handlers can retrieve a horizon.App instance
func appContextMiddleware(app *App) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := withAppContext(r.Context(), app)
			h.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requestCacheHeadersMiddleware adds caching headers to each response.
func requestCacheHeadersMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Before changing this read Stack Overflow answer about staled request
		// in older versions of Chrome:
		// https://stackoverflow.com/questions/27513994/chrome-stalls-when-making-multiple-requests-to-same-resource
		w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0")
		h.ServeHTTP(w, r)
	})
}

func contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = hchi.WithChiRequestID(ctx)
		ctx = httpx.RequestContext(ctx, w, r)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const (
	clientNameHeader    = "X-Client-Name"
	clientVersionHeader = "X-Client-Version"
	appNameHeader       = "X-App-Name"
	appVersionHeader    = "X-App-Version"
)

func newWrapResponseWriter(w http.ResponseWriter, r *http.Request) middleware.WrapResponseWriter {
	mw, ok := w.(middleware.WrapResponseWriter)
	if !ok {
		mw = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	}

	return mw
}

// loggerMiddleware logs http requests and resposnes to the logging subsytem of horizon.
func loggerMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		mw := newWrapResponseWriter(w, r)

		logger := log.WithField("req", middleware.GetReqID(ctx))
		ctx = log.Set(ctx, logger)

		// Checking `Accept` header from user request because if the streaming connection
		// is reset before sending the first event no Content-Type header is sent in a response.
		acceptHeader := r.Header.Get("Accept")
		streaming := strings.Contains(acceptHeader, render.MimeEventStream)

		logStartOfRequest(ctx, r, streaming)
		then := time.Now()

		h.ServeHTTP(mw, r.WithContext(ctx))

		duration := time.Since(then)
		logEndOfRequest(ctx, r, duration, mw, streaming)
	})
}

// timeoutMiddleware ensures the request is terminated after the given timeout
func timeoutMiddleware(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			mw := newWrapResponseWriter(w, r)
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer func() {
				cancel()
				if ctx.Err() == context.DeadlineExceeded {
					if mw.Status() == 0 {
						// only write the header if it hasn't been written yet
						mw.WriteHeader(http.StatusGatewayTimeout)
					}
				}
			}()

			r = r.WithContext(ctx)
			next.ServeHTTP(mw, r)
		}
		return http.HandlerFunc(fn)
	}
}

// getClientData gets client data (name or version) from header or GET parameter
// (useful when not possible to set headers, like in EventStream).
func getClientData(r *http.Request, headerName string) string {
	value := r.Header.Get(headerName)
	if value != "" {
		return value
	}

	value = r.URL.Query().Get(headerName)
	if value == "" {
		value = "undefined"
	}

	return value
}

func logStartOfRequest(ctx context.Context, r *http.Request, streaming bool) {
	referer := r.Referer()
	if referer == "" {
		referer = "undefined"
	}

	log.Ctx(ctx).WithFields(log.F{
		"client_name":    getClientData(r, clientNameHeader),
		"client_version": getClientData(r, clientVersionHeader),
		"app_name":       getClientData(r, appNameHeader),
		"app_version":    getClientData(r, appVersionHeader),
		"forwarded_ip":   firstXForwardedFor(r),
		"host":           r.Host,
		"ip":             remoteAddrIP(r),
		"ip_port":        r.RemoteAddr,
		"method":         r.Method,
		"path":           r.URL.String(),
		"streaming":      streaming,
		"referer":        referer,
	}).Info("Starting request")
}

func logEndOfRequest(ctx context.Context, r *http.Request, duration time.Duration, mw middleware.WrapResponseWriter, streaming bool) {
	routePattern := chi.RouteContext(r.Context()).RoutePattern()
	// Can be empty when request did not reached the final route (ex. blocked by
	// a middleware). More info: https://github.com/go-chi/chi/issues/270
	if routePattern == "" {
		routePattern = "undefined"
	}

	referer := r.Referer()
	if referer == "" {
		referer = "undefined"
	}

	log.Ctx(ctx).WithFields(log.F{
		"bytes":          mw.BytesWritten(),
		"client_name":    getClientData(r, clientNameHeader),
		"client_version": getClientData(r, clientVersionHeader),
		"app_name":       getClientData(r, appNameHeader),
		"app_version":    getClientData(r, appVersionHeader),
		"duration":       duration.Seconds(),
		"forwarded_ip":   firstXForwardedFor(r),
		"host":           r.Host,
		"ip":             remoteAddrIP(r),
		"ip_port":        r.RemoteAddr,
		"method":         r.Method,
		"path":           r.URL.String(),
		"route":          routePattern,
		"status":         mw.Status(),
		"streaming":      streaming,
		"referer":        referer,
	}).Info("Finished request")
}

func firstXForwardedFor(r *http.Request) string {
	return strings.TrimSpace(strings.SplitN(r.Header.Get("X-Forwarded-For"), ",", 2)[0])
}

func (w *web) RateLimitMiddleware(next http.Handler) http.Handler {
	if w.rateLimiter == nil {
		return next
	}
	return w.rateLimiter.RateLimit(next)
}

// recoverMiddleware helps the server recover from panics. It ensures that
// no request can fully bring down the horizon server, and it also logs the
// panics to the logging subsystem.
func recoverMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer func() {
			if rec := recover(); rec != nil {
				err := errors.FromPanic(rec)
				errors.ReportToSentry(err, r)
				problem.Render(ctx, w, err)
			}
		}()

		h.ServeHTTP(w, r)
	})
}

// requestMetricsMiddleware records success and failures using a meter, and times every request
func requestMetricsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := AppFromContext(r.Context())
		mw := newWrapResponseWriter(w, r)

		app.web.requestTimer.Time(func() {
			h.ServeHTTP(mw.(http.ResponseWriter), r)
		})

		if 200 <= mw.Status() && mw.Status() < 400 {
			// a success is in [200, 400)
			app.web.successMeter.Mark(1)
		} else if 400 <= mw.Status() && mw.Status() < 600 {
			// a success is in [400, 600)
			app.web.failureMeter.Mark(1)
		}
	})
}

// NewHistoryMiddleware adds session to the request context and ensures Horizon
// is not in a stale state, which is when the difference between latest core
// ledger and latest history ledger is higher than the given threshold
func NewHistoryMiddleware(staleThreshold int32, session *db.Session) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if staleThreshold > 0 {
				ls := ledger.CurrentState()
				isStale := (ls.CoreLatest - ls.HistoryLatest) > int32(staleThreshold)
				if isStale {
					err := hProblem.StaleHistory
					err.Extras = map[string]interface{}{
						"history_latest_ledger": ls.HistoryLatest,
						"core_latest_ledger":    ls.CoreLatest,
					}
					problem.Render(r.Context(), w, err)
					return
				}
			}

			requestSession := session.Clone()
			requestSession.Ctx = r.Context()
			h.ServeHTTP(w, r.WithContext(
				context.WithValue(
					r.Context(),
					&horizonContext.SessionContextKey,
					requestSession,
				),
			))
		})
	}
}

// StateMiddleware is a middleware which enables a state handler if the state
// has been initialized.
// Unless NoStateVerification is set, it ensures that the state (ledger entries)
// has been verified and is correct (Otherwise returns `500 Internal Server Error` to prevent
// returning invalid data to the user)
type StateMiddleware struct {
	HorizonSession      *db.Session
	NoStateVerification bool
}

func ingestionStatus(q *history.Q) (uint32, bool, error) {
	version, err := q.GetExpIngestVersion()
	if err != nil {
		return 0, false, supportErrors.Wrap(
			err, "Error running GetExpIngestVersion",
		)
	}

	lastIngestedLedger, err := q.GetLastLedgerExpIngestNonBlocking()
	if err != nil {
		return 0, false, supportErrors.Wrap(
			err, "Error running GetLastLedgerExpIngestNonBlocking",
		)
	}

	var lastHistoryLedger uint32
	err = q.LatestLedger(&lastHistoryLedger)
	if err != nil {
		return 0, false, supportErrors.Wrap(err, "Error running LatestLedger")
	}

	ready := version == expingest.CurrentVersion &&
		lastIngestedLedger > 0 &&
		lastIngestedLedger == lastHistoryLedger

	return lastIngestedLedger, ready, nil
}

// WrapFunc executes the middleware on a given HTTP handler function
func (m *StateMiddleware) WrapFunc(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := m.HorizonSession.Clone()
		q := &history.Q{session}
		sseRequest := render.Negotiate(r) == render.MimeEventStream

		// We want to start a repeatable read session to ensure that the data we
		// fetch from the db belong to the same ledger.
		// Otherwise, because the ingestion system is running concurrently with this request,
		// it is possible to have one read fetch data from ledger N and another read
		// fetch data from ledger N+1 .
		session.Ctx = r.Context()
		err := session.BeginTx(&sql.TxOptions{
			Isolation: sql.LevelRepeatableRead,
			ReadOnly:  true,
		})
		if err != nil {
			err = supportErrors.Wrap(err, "Error starting exp ingestion read transaction")
			problem.Render(r.Context(), w, err)
			return
		}
		defer session.Rollback()

		if !m.NoStateVerification {
			stateInvalid, invalidErr := q.GetExpStateInvalid()
			if invalidErr != nil {
				invalidErr = supportErrors.Wrap(invalidErr, "Error running GetExpStateInvalid")
				problem.Render(r.Context(), w, invalidErr)
				return
			}
			if stateInvalid {
				problem.Render(r.Context(), w, problem.ServerError)
				return
			}
		}

		lastIngestedLedger, ready, err := ingestionStatus(q)
		if err != nil {
			problem.Render(r.Context(), w, err)
			return
		}
		if !m.NoStateVerification && !ready {
			problem.Render(r.Context(), w, hProblem.StillIngesting)
			return
		}

		// for SSE requests we need to discard the repeatable read transaction
		// otherwise, the stream will not pick up updates occurring in future
		// ledgers
		if sseRequest {
			if err = session.Rollback(); err != nil {
				problem.Render(
					r.Context(),
					w,
					supportErrors.Wrap(
						err,
						"Could not roll back repeatable read session for SSE request",
					),
				)
				return
			}
		} else {
			actions.SetLastLedgerHeader(w, lastIngestedLedger)
		}

		h.ServeHTTP(w, r.WithContext(
			context.WithValue(
				r.Context(),
				&horizonContext.SessionContextKey,
				session,
			),
		))
	}
}

// WrapFunc executes the middleware on a given HTTP handler function
func (m *StateMiddleware) Wrap(h http.Handler) http.Handler {
	return m.WrapFunc(h.ServeHTTP)
}
