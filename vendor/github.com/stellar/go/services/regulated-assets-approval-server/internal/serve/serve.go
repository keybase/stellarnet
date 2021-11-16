package serve

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/stellar/go/amount"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/db"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/kycstatus"
	"github.com/stellar/go/support/errors"
	supporthttp "github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/health"
)

type Options struct {
	AssetCode                         string
	BaseURL                           string
	DatabaseURL                       string
	FriendbotPaymentAmount            int
	HorizonURL                        string
	IssuerAccountSecret               string
	KYCRequiredPaymentAmountThreshold string
	NetworkPassphrase                 string
	Port                              int
}

func Serve(opts Options) {
	listenAddr := fmt.Sprintf(":%d", opts.Port)
	serverConfig := supporthttp.Config{
		ListenAddr:          listenAddr,
		Handler:             handleHTTP(opts),
		TCPKeepAlive:        time.Minute * 3,
		ShutdownGracePeriod: time.Second * 50,
		ReadTimeout:         time.Second * 5,
		WriteTimeout:        time.Second * 35,
		IdleTimeout:         time.Minute * 2,
		OnStarting: func() {
			log.Info("Starting SEP-8 Approval Server")
			log.Infof("Listening on %s", listenAddr)
		},
		OnStopping: func() {
			log.Info("Stopping SEP-8 Approval Server")
		},
	}
	supporthttp.Run(serverConfig)
}

func handleHTTP(opts Options) http.Handler {
	issuerKP, err := keypair.ParseFull(opts.IssuerAccountSecret)
	if err != nil {
		log.Fatal(errors.Wrap(err, "parsing secret"))
	}
	parsedKYCRequiredPaymentThreshold, err := amount.ParseInt64(opts.KYCRequiredPaymentAmountThreshold)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "%s cannot be parsed as a Stellar amount", opts.KYCRequiredPaymentAmountThreshold))
	}
	db, err := db.Open(opts.DatabaseURL)
	if err != nil {
		log.Fatal(errors.Wrap(err, "error parsing database url"))
	}
	db.SetMaxOpenConns(20)
	err = db.Ping()
	if err != nil {
		log.Warn("Error pinging to Database: ", err)
	}
	mux := chi.NewMux()

	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(supporthttp.LoggingMiddleware)
	mux.Use(corsHandler)

	mux.Get("/health", health.PassHandler{}.ServeHTTP)
	mux.Get("/.well-known/stellar.toml", stellarTOMLHandler{
		assetCode:         opts.AssetCode,
		issuerAddress:     issuerKP.Address(),
		networkPassphrase: opts.NetworkPassphrase,
		approvalServer:    buildURLString(opts.BaseURL, "tx-approve"),
		kycThreshold:      parsedKYCRequiredPaymentThreshold,
	}.ServeHTTP)
	mux.Get("/friendbot", friendbotHandler{
		assetCode:           opts.AssetCode,
		issuerAccountSecret: opts.IssuerAccountSecret,
		horizonClient:       opts.horizonClient(),
		horizonURL:          opts.HorizonURL,
		networkPassphrase:   opts.NetworkPassphrase,
		paymentAmount:       opts.FriendbotPaymentAmount,
	}.ServeHTTP)
	mux.Post("/tx-approve", txApproveHandler{
		assetCode:         opts.AssetCode,
		issuerKP:          issuerKP,
		horizonClient:     opts.horizonClient(),
		networkPassphrase: opts.NetworkPassphrase,
		db:                db,
		kycThreshold:      parsedKYCRequiredPaymentThreshold,
		baseURL:           opts.BaseURL,
	}.ServeHTTP)
	mux.Route("/kyc-status", func(mux chi.Router) {
		mux.Post("/{callback_id}", kycstatus.PostHandler{
			DB: db,
		}.ServeHTTP)
		mux.Get("/{stellar_address_or_callback_id}", kycstatus.GetDetailHandler{
			DB: db,
		}.ServeHTTP)
		mux.Delete("/{stellar_address}", kycstatus.DeleteHandler{
			DB: db,
		}.ServeHTTP)
	})

	return mux
}

func (opts Options) horizonClient() horizonclient.ClientInterface {
	return &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP:       &http.Client{Timeout: 30 * time.Second},
	}
}

func buildURLString(baseURL, endpoint string) string {
	URL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "Unable to parse URL: %s", baseURL))
	}
	URL.Path = path.Join(URL.Path, endpoint)
	return URL.String()
}
