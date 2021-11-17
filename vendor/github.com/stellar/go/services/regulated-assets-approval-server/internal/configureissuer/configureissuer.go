package configureissuer

import (
	"net/http"
	"net/url"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
)

type Options struct {
	AssetCode           string
	BaseURL             string
	HorizonURL          string
	IssuerAccountSecret string
	NetworkPassphrase   string
}

func Setup(opts Options) {
	hClient := &horizonclient.Client{
		HorizonURL: opts.HorizonURL,
		HTTP:       &http.Client{Timeout: 30 * time.Second},
	}
	if opts.HorizonURL == horizonclient.DefaultTestNetClient.HorizonURL && opts.NetworkPassphrase == network.TestNetworkPassphrase {
		hClient = horizonclient.DefaultTestNetClient
	}

	issuerKP := keypair.MustParse(opts.IssuerAccountSecret)

	err := setup(opts, hClient)
	if err != nil {
		log.Error(errors.Wrap(err, "setting up issuer account"))
		log.Fatal("Couldn't complete setup!")
	}

	log.Infof("🎉🎉🎉 Successfully configured asset issuer for %s:%s", opts.AssetCode, issuerKP.Address())
}

func setup(opts Options, hClient horizonclient.ClientInterface) error {
	issuerKP, err := keypair.ParseFull(opts.IssuerAccountSecret)
	if err != nil {
		log.Fatal(errors.Wrap(err, "parsing secret"))
	}

	issuerAcc, err := getOrFundIssuerAccount(issuerKP.Address(), hClient)
	if err != nil {
		return errors.Wrap(err, "getting or funding issuer account")
	}

	asset := txnbuild.CreditAsset{
		Code:   opts.AssetCode,
		Issuer: issuerKP.Address(),
	}
	assetResults, err := hClient.Assets(horizonclient.AssetRequest{
		ForAssetCode:   asset.Code,
		ForAssetIssuer: asset.Issuer,
		Limit:          1,
	})
	if err != nil {
		return errors.Wrap(err, "getting list of assets")
	}

	u, err := url.Parse(opts.BaseURL)
	if err != nil {
		return errors.Wrap(err, "parsing base url")
	}
	homeDomain := u.Hostname()

	if issuerAcc.Flags.AuthRequired && issuerAcc.Flags.AuthRevocable && issuerAcc.HomeDomain == homeDomain && len(assetResults.Embedded.Records) > 0 {
		log.Warn("Account already configured. Aborting without performing any action.")
		return nil
	}

	trustorKP, err := keypair.Random()
	if err != nil {
		return errors.Wrap(err, "generating keypair")
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        issuerAcc,
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.SetOptions{
				SetFlags: []txnbuild.AccountFlag{
					txnbuild.AuthRequired,
					txnbuild.AuthRevocable,
				},
				HomeDomain: &homeDomain,
			},
			&txnbuild.BeginSponsoringFutureReserves{
				SponsoredID:   trustorKP.Address(),
				SourceAccount: issuerKP.Address(),
			},
			&txnbuild.CreateAccount{
				Destination:   trustorKP.Address(),
				Amount:        "0",
				SourceAccount: asset.Issuer,
			},
			// a trustline is generated to the desired so horizon creates entry at `{horizon-url}/assets`. This was added as many Wallets reach that endpoint to check if a given asset exists.
			&txnbuild.ChangeTrust{
				Line:          asset.MustToChangeTrustAsset(),
				SourceAccount: trustorKP.Address(),
			},
			&txnbuild.SetOptions{
				MasterWeight:    txnbuild.NewThreshold(0),
				LowThreshold:    txnbuild.NewThreshold(1),
				MediumThreshold: txnbuild.NewThreshold(1),
				HighThreshold:   txnbuild.NewThreshold(1),
				Signer:          &txnbuild.Signer{Address: issuerKP.Address(), Weight: txnbuild.Threshold(10)},
				SourceAccount:   trustorKP.Address(),
			},
			&txnbuild.EndSponsoringFutureReserves{
				SourceAccount: trustorKP.Address(),
			},
		},
		BaseFee:    300,
		Timebounds: txnbuild.NewTimeout(300),
	})
	if err != nil {
		return errors.Wrap(err, "building transaction")
	}

	tx, err = tx.Sign(opts.NetworkPassphrase, issuerKP, trustorKP)
	if err != nil {
		return errors.Wrap(err, "signing transaction")
	}

	_, err = hClient.SubmitTransaction(tx)
	if err != nil {
		return errors.Wrap(err, "submitting transaction")
	}

	return nil
}

func getOrFundIssuerAccount(issuerAddress string, hClient horizonclient.ClientInterface) (*horizon.Account, error) {
	issuerAcc, err := hClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: issuerAddress,
	})
	if err != nil {
		if !horizonclient.IsNotFoundError(err) || hClient != horizonclient.DefaultTestNetClient {
			return nil, errors.Wrapf(err, "getting detail for account %s", issuerAddress)
		}

		log.Info("Issuer account not found 👀 on network, will fund it using friendbot.")
		_, err = hClient.Fund(issuerAddress)
		if err != nil {
			return nil, errors.Wrap(err, "funding account with friendbot")
		}
		log.Info("🎉  Successfully funded account using friendbot.")
	}

	// now the account should be funded by the friendbot already
	issuerAcc, err = hClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: issuerAddress,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "getting detail for account %s", issuerAddress)
	}

	return &issuerAcc, nil
}
