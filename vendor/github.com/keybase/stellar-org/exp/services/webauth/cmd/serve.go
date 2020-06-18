package cmd

import (
	"go/types"

	"github.com/spf13/cobra"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/services/webauth/internal/serve"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/config"
	supportlog "github.com/stellar/go/support/log"
)

type ServeCommand struct {
	Logger *supportlog.Entry
}

func (c *ServeCommand) Command() *cobra.Command {
	opts := serve.Options{
		Logger: c.Logger,
	}
	configOpts := config.ConfigOptions{
		{
			Name:        "port",
			Usage:       "Port to listen and serve on",
			OptType:     types.Int,
			ConfigKey:   &opts.Port,
			FlagDefault: 8000,
			Required:    true,
		},
		{
			Name:        "horizon-url",
			Usage:       "Horizon URL used for looking up account details",
			OptType:     types.String,
			ConfigKey:   &opts.HorizonURL,
			FlagDefault: horizonclient.DefaultTestNetClient.HorizonURL,
			Required:    true,
		},
		{
			Name:        "network-passphrase",
			Usage:       "Network passphrase of the Stellar network transactions should be signed for",
			OptType:     types.String,
			ConfigKey:   &opts.NetworkPassphrase,
			FlagDefault: network.TestNetworkPassphrase,
			Required:    true,
		},
		{
			Name:      "signing-key",
			Usage:     "Stellar signing key(s) used for signing transactions comma separated (first key is used for signing, others used for verifying challenges)",
			OptType:   types.String,
			ConfigKey: &opts.SigningKeys,
			Required:  true,
		},
		{
			Name:           "challenge-expires-in",
			Usage:          "The time period in seconds after which the challenge transaction expires",
			OptType:        types.Int,
			CustomSetValue: config.SetDuration,
			ConfigKey:      &opts.ChallengeExpiresIn,
			FlagDefault:    300,
			Required:       true,
		},
		{
			Name:      "jwk",
			Usage:     "JSON Web Key (JWK) used for signing JWTs (if the key is an asymmetric key that has separate public and private key, the JWK must contain the private key)",
			OptType:   types.String,
			ConfigKey: &opts.JWK,
			Required:  true,
		},
		{
			Name:      "jwt-issuer",
			Usage:     "The issuer to set in the JWT iss claim",
			OptType:   types.String,
			ConfigKey: &opts.JWTIssuer,
			Required:  true,
		},
		{
			Name:           "jwt-expires-in",
			Usage:          "The time period in seconds after which the JWT expires",
			OptType:        types.Int,
			CustomSetValue: config.SetDuration,
			ConfigKey:      &opts.JWTExpiresIn,
			FlagDefault:    300,
			Required:       true,
		},
		{
			Name:        "allow-accounts-that-do-not-exist",
			Usage:       "Allow accounts that do not exist",
			OptType:     types.Bool,
			ConfigKey:   &opts.AllowAccountsThatDoNotExist,
			FlagDefault: false,
		},
	}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the SEP-10 Web Authentication server",
		Run: func(_ *cobra.Command, _ []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Run(opts)
		},
	}
	configOpts.Init(cmd)
	return cmd
}

func (c *ServeCommand) Run(opts serve.Options) {
	serve.Serve(opts)
}
