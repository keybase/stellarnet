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
	{
		// this xdr comes from the example in the spec, but it fails base64 decoding
		URI: "web+stellar:tx?origin_domain=blog.stathat.com&xdr=AAAAAL6Qe0ushP7lzogR2y3vyb8LKiorvD1U2KIlfs1wRBliAAAAZAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAABEz4bSpWmsmrXcIVAkY2hM3VdeCBJse56M18LaGzHQUAAAAAAAAAAACadvgAAAAAAAAAAA&signature=rv3QDVC6IDIrP062Wko%2BIMWVc8pDrMszZcjdgCGNfYdzrcjDpfP%2BSC97S61JJfqqfv%2F4KRqcxJcgYnFezNg%2BDA%3D%3D",
		Err: ErrInvalidParameter{Key: "xdr"},
	},
	{
		// this xdr comes from the change trust example in the spec, but it fails base64 decoding
		URI: "web+stellar:tx?origin_domain=blog.stathat.com&xdr=AAAAAP%2Byw%2BZEuNg533pUmwlYxfrq6%2FBoMJqiJ8vuQhf6rHWmAAAAZAB8NHAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAA%2F7LD5kS42DnfelSbCVjF%2Burr8GgwmqIny%2B5CF%2FqsdaYAAAAAAAAAAACYloAAAAAAAAAAAA&signature=1RD8KbHbCCXqlKkA2oiDyb7wgxZO4%2FnSEa3CFhP4gl4YhNZi9UWTspEtkbc6xRPvZyfJTDi0r6u6oNJuO8dLCQ%3D%3D",
		Err: ErrInvalidParameter{Key: "xdr"},
	},
}

var validTests = []validURITest{
	{
		URI:          "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=JTlGMGzxUv90P2SWxUY9xo%2BLlbXaDloend6gkpyylY8X4bUNf6%2F9mFTMJs7JKqSDPRtejlK1kQvrsJfRZSJeAQ%3D%3D",
		Operation:    "pay",
		OriginDomain: "someDomain.com",
	},
	{
		URI:          "web+stellar:tx?origin_domain=blog.stathat.com&xdr=AAAAAL6Qe0ushP7lzogR2y3vyb8LKiorvD1U2KIlfs1wRBliAAAAZAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAABEz4bSpWmsmrXcIVAkY2hM3VdeCBJse56M18LaGzHQUAAAAAAAAAAACadvgAAAAA&signature=VYODTfrluw38TTwoBEa5o7AMGVXv3NBXHIYIpyPY9bhnN5tMWGNw1yLaynWuGA29PKV%2BcyMfeFdP3wTWmHDwCA%3D%3D",
		Operation:    "tx",
		OriginDomain: "blog.stathat.com",
	},
	{
		URI:          "web+stellar:tx?origin_domain=blog.stathat.com&xdr=AAAAAP%2Byw%2BZEuNg533pUmwlYxfrq6%2FBoMJqiJ8vuQhf6rHWmAAAAZAB8NHAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAA%2F7LD5kS42DnfelSbCVjF%2Burr8GgwmqIny%2B5CF%2FqsdaYAAAAAAAAAAACYloAAAAAA&signature=JJaHhHColjAg%2BC8jIl%2Ba08%2F31tzRRBT9zYaiPJhcZobEt%2FMHZL8856ypRai5UTKHm8FVXfBT0XHE%2BE%2B4PdTFAg%3D%3D",
		Operation:    "tx",
		OriginDomain: "blog.stathat.com",
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
		switch v.Operation {
		case "pay":
			if v.Recipient == "" {
				t.Errorf("%d. valid pay operation but no recipient", i)
			}
		case "tx":
			if v.XDR == "" {
				t.Errorf("%d. valid tx operation but no xdr", i)
			}
			if v.Tx == nil {
				t.Errorf("%d. valid tx operation but no xdr.Transaction", i)
			}
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
	if err != nil {
		t.Fatal(err)
	}

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
	case "https://blog.stathat.com/.well-known/stellar.toml":
		body = `URI_REQUEST_SIGNING_KEY="GD6UAXSACFXDNGT6KXXC74VTECJ3M4R6SENCS2VAFRNLFQG5B2VV5JXZ"`
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
