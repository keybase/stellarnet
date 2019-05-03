package stellarnet

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/stellar/go/keypair"
)

// HTTPGetter is an interface for making GET http requests.
type HTTPGetter interface {
	Get(url string) (resp *http.Response, err error)
}

// ErrMissingParameter is returned when a required parameter is missing.
type ErrMissingParameter struct {
	Key string
}

// Error implements error for ErrMissingParameter.
func (e ErrMissingParameter) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("request missing required parameter %q", e.Key)
	}
	return "request missing required parameter"
}

// ErrInvalidParameter is returned when a parameter is invalid.
type ErrInvalidParameter struct {
	Key string
}

// Error implements error for ErrInvalidParameter.
func (e ErrInvalidParameter) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("request parameter %q is invalid", e.Key)
	}
	return "request parameter invalid"
}

// ErrNetworkWellKnownOrigin is returned when there is a network error
// fetching the well-known stellar.toml file.
type ErrNetworkWellKnownOrigin struct {
	Wrapped error
}

func (e ErrNetworkWellKnownOrigin) Error() string {
	return "network error getting signing key from origin domain"
}

// ErrInvalidWellKnownOrigin is returned when there the well-known stellar.toml
// file is invalid for the purposes of web+stellar URIs.
type ErrInvalidWellKnownOrigin struct {
	Wrapped error
}

func (e ErrInvalidWellKnownOrigin) Error() string {
	return "invalid origin domain stellar.toml file looking for signing key"
}

// ErrInvalidScheme is returned if the URI scheme is not web+stellar.
var ErrInvalidScheme = errors.New("invalid stellar URI scheme")

// ErrInvalidOperation is returned if the URI operation is not supported.
var ErrInvalidOperation = errors.New("invalid stellar URI operation")

// ErrBadSignature is returned if the signature fails verification.
var ErrBadSignature = errors.New("bad signature")

// ValidatedStellarURI contains the origin domain that ValidateStellarURI
// confirmed
type ValidatedStellarURI struct {
	URI          string
	XDR          string
	Operation    string
	OriginDomain string
}

// ValidateStellarURI will check the validity of a web+stellar SEP7 URI.
//
// It will check that the parameters are valid and that the payload is
// signed with the appropriate key.
func ValidateStellarURI(uri string, getter HTTPGetter) (*ValidatedStellarURI, error) {
	uv, err := newUnvalidatedURI(uri)
	if err != nil {
		return nil, err
	}
	return uv.Validate(getter)
}

type unvalidatedURI struct {
	raw          string
	values       url.Values
	Operation    string
	OriginDomain string
	Signature    string
}

func newUnvalidatedURI(uri string) (*unvalidatedURI, error) {
	res := &unvalidatedURI{raw: uri}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "web+stellar" {
		return nil, ErrInvalidScheme
	}

	res.Operation = u.Opaque

	res.values = u.Query()

	res.OriginDomain = res.value("origin_domain")
	res.Signature = res.value("signature")

	return res, nil

}

func (u *unvalidatedURI) Validate(getter HTTPGetter) (*ValidatedStellarURI, error) {
	// origin_domain and signature are optional in the spec, but
	// it seems like a really bad idea to allow any of these
	// requests without them, so we are going to make them required.
	if u.OriginDomain == "" {
		return nil, ErrMissingParameter{Key: "origin_domain"}
	}
	if u.Signature == "" {
		return nil, ErrMissingParameter{Key: "signature"}
	}

	// make sure there's no port or scheme
	if strings.IndexByte(u.OriginDomain, ':') != -1 {
		return nil, ErrInvalidParameter{Key: "origin_domain"}
	}

	switch u.Operation {
	case "pay":
		return u.validatePay(getter)
	case "tx":
		return u.validateTx(getter)
	default:
		return nil, ErrInvalidOperation
	}
}

type tomlStellar struct {
	SigningKey string `toml:"URI_REQUEST_SIGNING_KEY"`
}

func (u *unvalidatedURI) originDomainSigningKey(getter HTTPGetter) (string, error) {
	wellKnownURL := fmt.Sprintf("https://%s/.well-known/stellar.toml", u.OriginDomain)
	_, err := url.Parse(wellKnownURL)
	if err != nil {
		return "", ErrInvalidParameter{Key: "origin_domain"}
	}

	resp, err := getter.Get(wellKnownURL)
	if err != nil {
		return "", ErrNetworkWellKnownOrigin{Wrapped: err}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", ErrInvalidWellKnownOrigin{Wrapped: err}
	}
	if resp.StatusCode != http.StatusOK {
		return "", ErrInvalidWellKnownOrigin{Wrapped: errors.New("stellar.toml not found")}
	}

	var sdoc tomlStellar
	if _, err := toml.Decode(string(body), &sdoc); err != nil {
		return "", ErrInvalidWellKnownOrigin{Wrapped: err}
	}

	return strings.TrimSpace(sdoc.SigningKey), nil
}

func (u *unvalidatedURI) validateOriginDomain(getter HTTPGetter) error {
	signingKey, err := u.originDomainSigningKey(getter)
	if err != nil {
		return err
	}

	if signingKey == "" {
		return ErrInvalidWellKnownOrigin{Wrapped: errors.New("no signing key")}
	}

	kp, err := keypair.Parse(signingKey)
	if err != nil {
		return ErrInvalidWellKnownOrigin{Wrapped: errors.New("invalid signing key")}
	}

	signature, err := base64.StdEncoding.DecodeString(u.Signature)
	if err != nil {
		return ErrBadSignature
	}

	if err := kp.Verify(u.payload(), signature); err != nil {
		return ErrBadSignature
	}

	return nil
}

func (u *unvalidatedURI) payload() []byte {
	// get the portion of the URI that was signed by stripping &signature off the end
	index := strings.LastIndex(u.raw, "&signature=")
	if index == -1 {
		// this shouldn't happen because we already checked that signature
		// exists
		return nil
	}

	return payloadFromString(u.raw[0:index])
}

func (u *unvalidatedURI) validatePay(getter HTTPGetter) (*ValidatedStellarURI, error) {
	if err := u.validateOriginDomain(getter); err != nil {
		return nil, err
	}

	return &ValidatedStellarURI{Operation: "pay", OriginDomain: u.OriginDomain}, nil
}

func (u *unvalidatedURI) validateTx(getter HTTPGetter) (*ValidatedStellarURI, error) {
	xdr := u.value("xdr")
	if xdr == "" {
		return nil, ErrMissingParameter{Key: "xdr"}
	}

	if err := u.validateOriginDomain(getter); err != nil {
		return nil, err
	}

	return &ValidatedStellarURI{Operation: "tx", OriginDomain: u.OriginDomain}, nil
}

func (u *unvalidatedURI) value(key string) string {
	return strings.TrimSpace(u.values.Get(key))
}

func payloadFromString(data string) []byte {
	payload := make([]byte, 36)
	payload[35] = 4

	payload = append(payload, []byte("stellar.sep.7 - URI Scheme")...)
	payload = append(payload, []byte(data)...)

	return payload
}
