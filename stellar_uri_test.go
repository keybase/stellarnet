package stellarnet

import "testing"

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
}

var validTests = []validURITest{
	{
		URI:          "web+stellar:pay?destination=GCALNQQBXAPZ2WIRSDDBMSTAKCUH5SG6U76YBFLQLIXJTF7FE5AX7AOO&amount=120.1234567&memo=skdjfasf&msg=pay%20me%20with%20lumens&origin_domain=someDomain.com&signature=x%2BiZA4v8kkDj%2BiwoD1wEr%2BeFUcY2J8SgxCaYcNz4WEOuDJ4Sq0ps0rJpHfIKKzhrP4Gi1M58sTzlizpcVNX3DQ%3D%3D",
		Operation:    "pay",
		OriginDomain: "someDomain.com",
	},
}

func TestInvalidStellarURIs(t *testing.T) {
	for i, test := range invalidTests {
		v, err := ValidateStellarURI(test.URI)
		if err != test.Err {
			t.Errorf("%d. expected err %s, got %s", i, test.Err, err)
		}
		if v != nil {
			t.Errorf("%d. expected nil result, got %+v", i, v)
		}
	}
}

func TestValidStellarURIs(t *testing.T) {
	for i, test := range validTests {
		v, err := ValidateStellarURI(test.URI)
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
