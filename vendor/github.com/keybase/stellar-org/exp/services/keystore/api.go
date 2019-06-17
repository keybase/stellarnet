package keystore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
)

func init() {
	// register errors
	problem.RegisterError(httpjson.ErrBadRequest, probInvalidRequest)
	problem.RegisterError(sql.ErrNoRows, problem.NotFound)

	// register service host as an empty string
	problem.RegisterHost("")
}

func (s *Service) wrapMiddleware(handler http.Handler) http.Handler {
	handler = authHandler(handler, s.authenticator)
	return recoverHandler(handler)
}

func ServeMux(s *Service) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/keys", s.wrapMiddleware(s.keysHTTPMethodHandler()))
	mux.Handle("/health", s.wrapMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		return
	})))
	return mux
}

func (s *Service) keysHTTPMethodHandler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			jsonHandler(s.getKeys).ServeHTTP(rw, req)

		case http.MethodPut:
			jsonHandler(s.putKeys).ServeHTTP(rw, req)

		case http.MethodDelete:
			jsonHandler(s.deleteKeys).ServeHTTP(rw, req)

		default:
			problem.Render(req.Context(), rw, probMethodNotAllowed)
		}
	})
}

type authResponse struct {
	UserID string `json:"userID"`
}

func authHandler(next http.Handler, authenticator *Authenticator) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if authenticator == nil {
			// to facilitate API testing
			next.ServeHTTP(rw, req.WithContext(withUserID(req.Context(), "test-user")))
			return
		}

		var (
			proxyReq *http.Request
			err      error
			clientIP string
		)
		ctx := req.Context()
		// set a 5-second timeout
		client := http.Client{Timeout: time.Duration(5 * time.Second)}

		switch authenticator.APIType {
		case REST:
			proxyReq, err = http.NewRequest("GET", authenticator.URL, nil)
			if err != nil {
				problem.Render(ctx, rw, err)
				return
			}

		case GraphQL:
			// to be implemented later
		default:
			problem.Render(ctx, rw, probNotAuthorized)
			return
		}

		// shallow copy
		proxyReq.Header = req.Header
		if clientIP, _, err = net.SplitHostPort(req.RemoteAddr); err == nil {
			proxyReq.Header.Set("X-Forwarded-For", clientIP)
		}

		resp, err := client.Do(proxyReq)
		if err != nil {
			problem.Render(ctx, rw, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				problem.Render(ctx, rw, err)
				return
			}

			var authResp authResponse
			err = json.Unmarshal(body, &authResp)
			if err != nil {
				problem.Render(ctx, rw, err)
				return
			}
			if authResp.UserID == "" {
				problem.Render(ctx, rw, probNotAuthorized)
				return
			}
			// assuming the context-type is application/json
			ctx = withUserID(ctx, authResp.UserID)
		} else {
			problem.Render(ctx, rw, probNotAuthorized)
			return
		}

		next.ServeHTTP(rw, req.WithContext(ctx))
	})
}

func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.ReqBodyHandler(f, httpjson.JSON)
	if err != nil {
		panic(err)
	}
	return h
}

func recoverHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", r)
			}
			if errors.Cause(err) == http.ErrAbortHandler {
				panic(err)
			}

			ctx := req.Context()
			log.Ctx(ctx).WithStack(err).Error(err)
			problem.Render(ctx, rw, err)
		}()

		next.ServeHTTP(rw, req)
	})
}
