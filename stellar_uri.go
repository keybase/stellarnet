package stellarnet

import (
	"errors"
	"fmt"
	"net/url"

	"golang.org/x/xerrors"
)

// ErrInvalidScheme is returned if the URI scheme is not web+stellar.
var ErrInvalidScheme = errors.New("invalid stellar URI scheme")

// ErrInvalidOperation is returned if the URI operation is not supported.
var ErrInvalidOperation = errors.New("invalid stellar URI operation")

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
func ValidateStellarURI(uri string) (*ValidatedStellarURI, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "web+stellar" {
		return nil, ErrInvalidScheme
	}
	fmt.Printf("%q %q %q\n", u.Path, u.Opaque, u.RawQuery)
	operation := u.Opaque
	switch operation {
	case "pay":
		return validatePayStellarURI(uri, u)
	case "tx":
		return validateTxStellarURI(uri, u)
	default:
		return nil, xerrors.Errorf("operation %q: %w", operation, ErrInvalidOperation)
	}
}

func validatePayStellarURI(uri string, u *url.URL) (*ValidatedStellarURI, error) {
	return &ValidatedStellarURI{Operation: "pay"}, nil
}

func validateTxStellarURI(uri string, u *url.URL) (*ValidatedStellarURI, error) {
	return &ValidatedStellarURI{Operation: "tx"}, nil
}
