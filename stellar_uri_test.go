package stellarnet

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type invalidURITest struct {
	URI string
	Err error
}

type validURITest struct {
	URI          string
	Operation    string
	OriginDomain string
}

var invalidTests = []invalidURITest{
	{
		URI: "http://keybase.io",
		Err: ErrInvalidScheme,
	},
	{
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrMissingParameter{Key: "origin_domain"},
	},
	{
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=blah.com",
		Err: ErrMissingParameter{Key: "signature"},
	},
	{
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com:8000&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrInvalidParameter{Key: "origin_domain"},
	},
	{
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=http://someDomain.com&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrInvalidParameter{Key: "origin_domain"},
	},
	{
		URI: "web+stellar:sign?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrInvalidOperation,
	},
	{
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrBadSignature,
	},
	{
		URI: "web+stellar:tx?msg=signthis&origin_domain=someDomain.com&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrMissingParameter{Key: "xdr"},
	},
	{
		// this is the example in the spec, but it actually fails to validate
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Err: ErrBadSignature,
	},
	{
		// changed amount
		URI: "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=12.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=JTlGMGzxUv90P2SWxUY9xo%2BLlbXaDloend6gkpyylY8X4bUNf6%2F9mFTMJs7JKqSDPRtejlK1kQvrsJfRZSJeAQ%3D%3D",
		Err: ErrBadSignature,
	},
}

var validTests = []validURITest{
	{
		URI:          "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=JTlGMGzxUv90P2SWxUY9xo%2BLlbXaDloend6gkpyylY8X4bUNf6%2F9mFTMJs7JKqSDPRtejlK1kQvrsJfRZSJeAQ%3D%3D",
		Operation:    "pay",
		OriginDomain: "someDomain.com",
	},
}

func TestInvalidStellarURIs(t *testing.T) {
	for i, test := range invalidTests {
		v, err := ValidateStellarURI(test.URI, &httpClient{})
		if err != test.Err {
			if err == nil || (err.Error() != test.Err.Error()) {
				t.Errorf("%d. expected err %s, got %v", i, test.Err, err)
			}
		}
		if v != nil {
			t.Errorf("%d. expected nil result, got %+v", i, v)
		}
	}
}

func TestValidStellarURIs(t *testing.T) {
	for i, test := range validTests {
		v, err := ValidateStellarURI(test.URI, &httpClient{})
		if err != nil {
			t.Errorf("%d. expected no err, got %s", i, err)
			continue
		}
		if v.Operation != test.Operation {
			t.Errorf("%d. operation: %q, expected %q", i, v.Operation, test.Operation)
		}
		if v.OriginDomain != test.OriginDomain {
			t.Errorf("%d. origin domain: %q, expected %q", i, v.OriginDomain, test.OriginDomain)
		}
	}
}

func TestSignStellarURI(t *testing.T) {
	key := "SBPOVRVKTTV7W3IOX2FJPSMPCJ5L2WU2YKTP3HCLYPXNI5MDIGREVNYC"
	seed, err := NewSeedStr(key)
	if err != nil {
		t.Fatal(err)
	}

	signedURI, signatureB64, err := SignStellarURI("web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com", seed)

	if signedURI != "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=JTlGMGzxUv90P2SWxUY9xo%2BLlbXaDloend6gkpyylY8X4bUNf6%2F9mFTMJs7JKqSDPRtejlK1kQvrsJfRZSJeAQ%3D%3D" {
		t.Error("signedURI mismatch")
	}
	if signatureB64 != "JTlGMGzxUv90P2SWxUY9xo+LlbXaDloend6gkpyylY8X4bUNf6/9mFTMJs7JKqSDPRtejlK1kQvrsJfRZSJeAQ==" {
		t.Error("signature b64 mismatch")
	}
}

type httpClient struct{}

func (h *httpClient) Get(url string) (resp *http.Response, err error) {
	var body string
	switch url {
	case "https://someDomain.com/.well-known/stellar.toml":
		body = `URI_REQUEST_SIGNING_KEY="GD7ACHBPHSC5OJMJZZBXA7Z5IAUFTH6E6XVLNBPASDQYJ7LO5UIYBDQW"`
	default:
		fmt.Println(url)
	}

	r := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(body)),
	}
	if body == "" {
		r.StatusCode = http.StatusNotFound
	}

	return r, nil
}
