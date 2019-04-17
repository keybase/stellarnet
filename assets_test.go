package stellarnet

import (
	"testing"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stretchr/testify/require"
)

func TestAsset(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "asset")

	t.Logf("If this test fails, test asset might have disappeared.  see assets_test.go.")
	// Note: there is no guarantee that this asset code/issuer combo will
	// continue to work on horizon testnet.  It is recorded, but if you change
	// this test and it fails, find a new asset to check.
	//
	// https://horizon-testnet.stellar.org/assets?asset_code=EUR
	//
	// will show a list of EUR assets.  Pick one.
	//
	summary, err := Asset("EUR", "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS")
	require.NoError(t, err)

	require.Equal(t, "credit_alphanum4", summary.AssetType)
	require.Equal(t, "EUR", summary.AssetCode)
	require.Equal(t, "EUR", summary.AssetCode)
	require.Equal(t, "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS", summary.AssetIssuer)
	require.Empty(t, summary.UnverifiedWellKnownLink)
}

func TestAssetSearch(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "assetSearch")

	var res []AssetSummary
	var expectedMatch AssetSummary
	var err error
	search := func(code, issuer string) ([]AssetSummary, error) {
		arg := AssetSearchArg{
			AssetCode: code,
			IssuerID:  issuer,
		}
		return AssetSearch(arg)
	}

	// finds an assetcode with a bunch of issuers
	res, err = search("BTC", "")
	require.NoError(t, err)
	expectedMatch = AssetSummary{
		UnverifiedWellKnownLink: "",
		AssetType:               "credit_alphanum4",
		AssetCode:               "BTC",
		AssetIssuer:             "GA5FUT6R7MD6CXX6T42HJ6E7NYGNIWAQKWMWRUYR42YLRQG3YWUNRNZU",
		Amount:                  "0.0049880",
		NumAccounts:             2,
	}
	require.Equal(t, res[0], expectedMatch)
	require.Equal(t, len(res), 10)

	// finds an issuer with a bunch of assets
	res, err = search("", "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS")
	require.NoError(t, err)
	expectedMatch = AssetSummary{
		UnverifiedWellKnownLink: "",
		AssetType:               "credit_alphanum4",
		AssetCode:               "GBP",
		AssetIssuer:             "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS",
		Amount:                  "3000.0000000",
		NumAccounts:             3,
	}
	var foundMatch bool
	for _, asset := range res {
		if asset == expectedMatch {
			foundMatch = true
			break
		}
	}
	require.True(t, foundMatch)
	require.Equal(t, len(res), 10)

	// finds an exact match
	res, err = search("BTC", "GA5FUT6R7MD6CXX6T42HJ6E7NYGNIWAQKWMWRUYR42YLRQG3YWUNRNZU")
	require.NoError(t, err)
	expectedMatch = AssetSummary{
		UnverifiedWellKnownLink: "",
		AssetType:               "credit_alphanum4",
		AssetCode:               "BTC",
		AssetIssuer:             "GA5FUT6R7MD6CXX6T42HJ6E7NYGNIWAQKWMWRUYR42YLRQG3YWUNRNZU",
		Amount:                  "0.0049880",
		NumAccounts:             2,
	}
	foundMatch = false
	for _, asset := range res {
		if asset == expectedMatch {
			foundMatch = true
			break
		}
	}
	require.True(t, foundMatch)
	require.Equal(t, len(res), 1)

	// does not find a non-existent asset
	res, err = search("XXXX", "")
	require.NoError(t, err)
	require.Equal(t, len(res), 0)

	// does not find a non-existent but valid issuer
	res, err = search("", "GCS27P7V4IG5V3LJVWMNLPUXA4GWD22Y6BDIYMC2E22LQYHWUJGJSLFW")
	require.NoError(t, err)
	require.Equal(t, len(res), 0)

	// does not find a non-existent exact match
	res, err = search("BTC", "GCS27P7V4IG5V3LJVWMNLPUXA4GWD22Y6BDIYMC2E22LQYHWUJGJSLFW")
	require.NoError(t, err)
	require.Equal(t, len(res), 0)

	// empty search is empty results
	res, err = search("", "")
	require.NoError(t, err)
	require.Equal(t, len(res), 0)

	// errors on a bad issuerID
	res, err = search("", "badissuerID")
	require.Error(t, err)
	require.Equal(t, len(res), 0)
}
