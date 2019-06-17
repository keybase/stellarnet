package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	horizonclient "github.com/stellar/go/clients/horizonclient"
	hlog "github.com/stellar/go/support/log"
)

var DatabaseURL string
var Logger = hlog.New()
var Client = horizonclient.DefaultPublicNetClient // TODO: make this configurable

var rootCmd = &cobra.Command{
	Use:   "ticker",
	Short: "Stellar Development Foundation Ticker.",
	Long:  `A tool to provide Stellar Asset and Market data.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&DatabaseURL,
		"db-url",
		"d",
		"postgres://localhost:5432/stellarticker01?sslmode=disable",
		"database URL, such as: postgres://user:pass@localhost:5432/ticker",
	)

	Logger.SetLevel(logrus.DebugLevel)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
