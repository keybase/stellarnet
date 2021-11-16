package main

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stellar/go/exp/services/captivecore/internal"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	supporthttp "github.com/stellar/go/support/http"
	supportlog "github.com/stellar/go/support/log"
)

func main() {
	var port int
	var networkPassphrase, binaryPath, configPath, dbURL string
	var captiveCoreTomlParams ledgerbackend.CaptiveCoreTomlParams
	var historyArchiveURLs []string
	var checkpointFrequency uint32
	var logLevel logrus.Level
	logger := supportlog.New()

	configOpts := config.ConfigOptions{
		{
			Name:        "port",
			Usage:       "Port to listen and serve on",
			OptType:     types.Int,
			ConfigKey:   &port,
			FlagDefault: 8000,
			Required:    true,
		},
		{
			Name:        "network-passphrase",
			Usage:       "Network passphrase of the Stellar network transactions should be signed for",
			OptType:     types.String,
			ConfigKey:   &networkPassphrase,
			FlagDefault: network.TestNetworkPassphrase,
			Required:    true,
		},
		&config.ConfigOption{
			Name:        "stellar-core-binary-path",
			OptType:     types.String,
			FlagDefault: "",
			Required:    true,
			Usage:       "path to stellar core binary",
			ConfigKey:   &binaryPath,
		},
		&config.ConfigOption{
			Name:        "captive-core-config-path",
			OptType:     types.String,
			FlagDefault: "",
			Required:    true,
			Usage:       "path to additional configuration for the Stellar Core configuration file used by captive core. It must, at least, include enough details to define a quorum set",
			ConfigKey:   &configPath,
		},
		&config.ConfigOption{
			Name:        "history-archive-urls",
			ConfigKey:   &historyArchiveURLs,
			OptType:     types.String,
			Required:    true,
			FlagDefault: "",
			CustomSetValue: func(co *config.ConfigOption) error {
				stringOfUrls := viper.GetString(co.Name)
				urlStrings := strings.Split(stringOfUrls, ",")

				*(co.ConfigKey.(*[]string)) = urlStrings
				return nil
			},
			Usage: "comma-separated list of stellar history archives to connect with",
		},
		&config.ConfigOption{
			Name:        "log-level",
			ConfigKey:   &logLevel,
			OptType:     types.String,
			FlagDefault: "info",
			CustomSetValue: func(co *config.ConfigOption) error {
				ll, err := logrus.ParseLevel(viper.GetString(co.Name))
				if err != nil {
					return fmt.Errorf("Could not parse log-level: %v", viper.GetString(co.Name))
				}
				*(co.ConfigKey.(*logrus.Level)) = ll
				return nil
			},
			Usage: "minimum log severity (debug, info, warn, error) to log",
		},
		&config.ConfigOption{
			Name:      "db-url",
			EnvVar:    "DATABASE_URL",
			ConfigKey: &dbURL,
			OptType:   types.String,
			Required:  false,
			Usage:     "horizon postgres database to connect with",
		},
		&config.ConfigOption{
			Name:           "stellar-captive-core-http-port",
			ConfigKey:      &captiveCoreTomlParams.HTTPPort,
			OptType:        types.Uint,
			CustomSetValue: config.SetOptionalUint,
			Required:       false,
			FlagDefault:    uint(11626),
			Usage:          "HTTP port for Captive Core to listen on (0 disables the HTTP server)",
		},
		&config.ConfigOption{
			Name:        "checkpoint-frequency",
			ConfigKey:   &checkpointFrequency,
			OptType:     types.Uint32,
			FlagDefault: uint32(64),
			Required:    false,
			Usage:       "establishes how many ledgers exist between checkpoints, do NOT change this unless you really know what you are doing",
		},
	}
	cmd := &cobra.Command{
		Use:   "captivecore",
		Short: "Run the remote captive core server",
		Run: func(_ *cobra.Command, _ []string) {
			configOpts.Require()
			configOpts.SetValues()
			logger.SetLevel(logLevel)

			captiveCoreTomlParams.HistoryArchiveURLs = historyArchiveURLs
			captiveCoreTomlParams.NetworkPassphrase = networkPassphrase
			captiveCoreTomlParams.Strict = true
			captiveCoreToml, err := ledgerbackend.NewCaptiveCoreTomlFromFile(configPath, captiveCoreTomlParams)
			if err != nil {
				logger.WithError(err).Fatal("Invalid captive core toml")
			}

			captiveConfig := ledgerbackend.CaptiveCoreConfig{
				BinaryPath:          binaryPath,
				NetworkPassphrase:   networkPassphrase,
				HistoryArchiveURLs:  historyArchiveURLs,
				CheckpointFrequency: checkpointFrequency,
				Log:                 logger.WithField("subservice", "stellar-core"),
				Toml:                captiveCoreToml,
			}

			var dbConn *db.Session
			if len(dbURL) > 0 {
				dbConn, err = db.Open("postgres", dbURL)
				if err != nil {
					logger.WithError(err).Fatal("Could not create db connection instance")
				}
				captiveConfig.LedgerHashStore = ledgerbackend.NewHorizonDBLedgerHashStore(dbConn)
			}

			core, err := ledgerbackend.NewCaptive(captiveConfig)
			if err != nil {
				logger.WithError(err).Fatal("Could not create captive core instance")
			}
			api := internal.NewCaptiveCoreAPI(core, logger.WithField("subservice", "api"))

			supporthttp.Run(supporthttp.Config{
				ListenAddr: fmt.Sprintf(":%d", port),
				Handler:    internal.Handler(api),
				OnStarting: func() {
					logger.Infof("Starting Captive Core server on %v", port)
				},
				OnStopping: func() {
					// TODO: Check this aborts in-progress requests instead of letting
					// them finish, to preserve existing behaviour.
					api.Shutdown()
					if dbConn != nil {
						dbConn.Close()
					}
				},
			})
		},
	}

	if err := configOpts.Init(cmd); err != nil {
		logger.WithError(err).Fatal("could not parse config options")
	}

	if err := cmd.Execute(); err != nil {
		logger.WithError(err).Fatal("could not run")
	}
}
