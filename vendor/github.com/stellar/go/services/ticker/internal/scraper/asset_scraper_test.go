package scraper

import (
	"net/url"
	"testing"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stretchr/testify/assert"
)

func TestShouldDiscardAsset(t *testing.T) {
	testAsset := hProtocol.AssetStat{
		Amount: "",
	}

	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount: "0.0",
	}
	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount: "0",
	}
	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount:      "123901.0129310",
		NumAccounts: 8,
	}
	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount:      "123901.0129310",
		NumAccounts: 12,
	}
	testAsset.Code = "REMOVE"
	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount:      "123901.0129310",
		NumAccounts: 100,
	}
	testAsset.Code = "SOMETHINGVALID"
	testAsset.Links.Toml.Href = ""
	assert.Equal(t, shouldDiscardAsset(testAsset, true), false)

	testAsset = hProtocol.AssetStat{
		Amount:      "123901.0129310",
		NumAccounts: 40,
	}
	testAsset.Code = "SOMETHINGVALID"
	testAsset.Links.Toml.Href = "http://www.stellar.org/.well-known/stellar.toml"
	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount:      "123901.0129310",
		NumAccounts: 40,
	}
	testAsset.Code = "SOMETHINGVALID"
	testAsset.Links.Toml.Href = ""
	assert.Equal(t, shouldDiscardAsset(testAsset, true), true)

	testAsset = hProtocol.AssetStat{
		Amount:      "123901.0129310",
		NumAccounts: 40,
	}
	testAsset.Code = "SOMETHINGVALID"
	testAsset.Links.Toml.Href = "https://www.stellar.org/.well-known/stellar.toml"
	assert.Equal(t, shouldDiscardAsset(testAsset, true), false)
}

func TestDomainsMatch(t *testing.T) {
	tomlURL, _ := url.Parse("https://stellar.org/stellar.toml")
	orgURL, _ := url.Parse("https://stellar.org/")
	assert.True(t, domainsMatch(tomlURL, orgURL))

	tomlURL, _ = url.Parse("https://assets.stellar.org/stellar.toml")
	orgURL, _ = url.Parse("https://stellar.org/")
	assert.False(t, domainsMatch(tomlURL, orgURL))

	tomlURL, _ = url.Parse("https://stellar.org/stellar.toml")
	orgURL, _ = url.Parse("https://home.stellar.org/")
	assert.True(t, domainsMatch(tomlURL, orgURL))

	tomlURL, _ = url.Parse("https://stellar.org/stellar.toml")
	orgURL, _ = url.Parse("https://home.stellar.com/")
	assert.False(t, domainsMatch(tomlURL, orgURL))

	tomlURL, _ = url.Parse("https://stellar.org/stellar.toml")
	orgURL, _ = url.Parse("https://stellar.com/")
	assert.False(t, domainsMatch(tomlURL, orgURL))
}

func TestIsDomainVerified(t *testing.T) {
	tomlURL := "https://stellar.org/stellar.toml"
	orgURL := "https://stellar.org/"
	hasCurrency := true
	assert.True(t, isDomainVerified(orgURL, tomlURL, hasCurrency))

	tomlURL = "https://stellar.org/stellar.toml"
	orgURL = ""
	hasCurrency = true
	assert.True(t, isDomainVerified(orgURL, tomlURL, hasCurrency))

	tomlURL = ""
	orgURL = ""
	hasCurrency = true
	assert.False(t, isDomainVerified(orgURL, tomlURL, hasCurrency))

	tomlURL = "https://stellar.org/stellar.toml"
	orgURL = "https://stellar.org/"
	hasCurrency = false
	assert.False(t, isDomainVerified(orgURL, tomlURL, hasCurrency))

	tomlURL = "http://stellar.org/stellar.toml"
	orgURL = "https://stellar.org/"
	hasCurrency = true
	assert.False(t, isDomainVerified(orgURL, tomlURL, hasCurrency))

	tomlURL = "https://stellar.org/stellar.toml"
	orgURL = "http://stellar.org/"
	hasCurrency = true
	assert.False(t, isDomainVerified(orgURL, tomlURL, hasCurrency))

	tomlURL = "https://stellar.org/stellar.toml"
	orgURL = "https://stellar.com/"
	hasCurrency = true
	assert.False(t, isDomainVerified(orgURL, tomlURL, hasCurrency))
}

func TestIgnoreInvalidTOMLUrls(t *testing.T) {
	invalidURL := "https:// there is something wrong here.com/stellar.toml"
	assetStat := hProtocol.AssetStat{}
	assetStat.Links.Toml = hal.Link{Href: invalidURL}

	_, err := fetchTOMLData(assetStat)

	urlErr, ok := errors.Cause(err).(*url.Error)
	if !ok {
		t.Fatalf("err expected to be a url.Error but was %#v", err)
	}
	assert.Equal(t, "parse", urlErr.Op)
	assert.Equal(t, "https:// there is something wrong here.com/stellar.toml", urlErr.URL)
	assert.EqualError(t, urlErr.Err, `invalid character " " in host name`)
}
