package stellarnet

import (
	"testing"

	"github.com/keybase/stellarnet/testclient"
	"github.com/stellar/go/xdr"
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
	summary, err := Asset("EUR", "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA")
	require.NoError(t, err)

	require.Equal(t, "credit_alphanum4", summary.AssetType)
	require.Equal(t, "EUR", summary.AssetCode)
	require.Equal(t, "EUR", summary.AssetCode)
	require.Equal(t, "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA", summary.AssetIssuer)
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
		AssetIssuer:             "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA",
		Amount:                  "10000.0000000",
		NumAccounts:             10,
	}
	require.Equal(t, expectedMatch, res[0])
	require.Equal(t, len(res), 10)

	// finds an issuer with a bunch of assets
	res, err = search("", "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA")
	require.NoError(t, err)
	expectedMatch = AssetSummary{
		UnverifiedWellKnownLink: "",
		AssetType:               "credit_alphanum4",
		AssetCode:               "MAD",
		AssetIssuer:             "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA",
		Amount:                  "0.0000000",
		NumAccounts:             10,
	}
	require.Contains(t, res, expectedMatch)
	require.Len(t, res, 6)

	// finds an exact match
	res, err = search("BTC", "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA")
	require.NoError(t, err)
	expectedMatch = AssetSummary{
		UnverifiedWellKnownLink: "",
		AssetType:               "credit_alphanum4",
		AssetCode:               "BTC",
		AssetIssuer:             "GA2NDBWMAXXMNM746EGSBQFIUDLOS6Q2EBVG6XPB7E4WNCVRK4CLJLRA",
		Amount:                  "10000.0000000",
		NumAccounts:             10,
	}
	require.Contains(t, res, expectedMatch)
	require.Equal(t, len(res), 1)

	// does not find a non-existent asset
	res, err = search("XAAX", "")
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

	// works with credit_alphanum12 asset codes
	res, err = search("006064", "")
	require.NoError(t, err)
	expectedMatch = AssetSummary{
		UnverifiedWellKnownLink: "",
		AssetType:               "credit_alphanum12",
		AssetCode:               "006064",
		AssetIssuer:             "GD4SAUKGB6GE2Q25H2CZMZ3BSP5CVYIY2LQYJDCFNNICR473AVL7IYH5",
		Amount:                  "53441.0000000",
		NumAccounts:             834,
	}
	require.Contains(t, res, expectedMatch)
}

func TestAssetList(t *testing.T) {
	helper, client, network := testclient.Setup(t)
	SetClientAndNetwork(client, network)
	helper.SetState(t, "assetList")

	limit := 10
	order := "asc"
	firstRes, firstCursor, err := AssetList("", limit, order)
	require.NoError(t, err)
	require.Equal(t, len(firstRes), limit)
	// We don't really care what the cursor is, only that it exists and we can pass it
	// along to the next request. But this is what they look like:
	// 0288d1_GD4SAUKGB6GE2Q25H2CZMZ3BSP5CVYIY2LQYJDCFNNICR473AVL7IYH5_credit_alphanum12
	require.True(t, len(firstCursor) > 0)

	secondRes, secondCursor, err := AssetList(firstCursor, limit, order)
	require.NoError(t, err)
	require.Equal(t, len(secondRes), limit)
	require.True(t, len(secondCursor) > 0)
}

func TestMakeXDRAsset(t *testing.T) {
	a, err := makeXDRAsset("", "")
	require.NoError(t, err)
	require.Equal(t, a.Type, xdr.AssetTypeAssetTypeNative)

	a, err = makeXDRAsset("", "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS")
	require.Error(t, err)

	a, err = makeXDRAsset("EUR", "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS")
	require.NoError(t, err)
	require.Equal(t, a.Type, xdr.AssetTypeAssetTypeCreditAlphanum4)

	a, err = makeXDRAsset("EUREUREUREUR", "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS")
	require.NoError(t, err)
	require.Equal(t, a.Type, xdr.AssetTypeAssetTypeCreditAlphanum12)

	a, err = makeXDRAsset("THIRTEENCHARS", "GAJVD2WOS7QXLSGFUQ3VIDEFG5I7S3VWL4X3V5FEFN4N2OC5CQDMHHZS")
	require.Error(t, err)
}
