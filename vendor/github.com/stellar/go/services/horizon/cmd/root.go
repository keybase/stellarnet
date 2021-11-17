package cmd

import (
	"fmt"
	stdLog "log"

	"github.com/spf13/cobra"
	horizon "github.com/stellar/go/services/horizon/internal"
)

var (
	config, flags = horizon.Flags()

	RootCmd = &cobra.Command{
		Use:           "horizon",
		Short:         "client-facing api server for the Stellar network",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long:          "Client-facing API server for the Stellar network. It acts as the interface between Stellar Core and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := horizon.NewAppFromFlags(config, flags)
			if err != nil {
				return err
			}
			return app.Serve()
		},
	}
)

// ErrUsage indicates we should print the usage string and exit with code 1
type ErrUsage struct {
	cmd *cobra.Command
}

func (e ErrUsage) Error() string {
	return e.cmd.UsageString()
}

// Indicates we want to exit with a specific error code without printing an error.
type ErrExitCode int

func (e ErrExitCode) Error() string {
	return fmt.Sprintf("exit code: %d", e)
}

func init() {
	err := flags.Init(RootCmd)
	if err != nil {
		stdLog.Fatal(err.Error())
	}
}

func Execute() error {
	return RootCmd.Execute()
}
