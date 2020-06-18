package horizonclient

import (
	"fmt"
	"net/url"

	"github.com/stellar/go/support/errors"
)

// BuildURL creates the endpoint to be queried based on the data in the AccountsRequest struct.
// Either "Signer" or "Asset" fields should be set when retrieving Accounts.
// At the moment, you can't use both filters at the same time.
func (r AccountsRequest) BuildURL() (endpoint string, err error) {

	nParams := countParams(r.Signer, r.Asset)

	if nParams <= 0 {
		err = errors.New("invalid request: no parameters - Signer or Asset must be provided")
	}

	if nParams >= 2 {
		err = errors.New("invalid request: too many parameters - Signer and Asset provided, provide a single filter")
	}

	if err != nil {
		return endpoint, err
	}
	query := url.Values{}
	switch {
	case len(r.Signer) > 0:
		query.Add("signer", r.Signer)

	case len(r.Asset) > 0:
		query.Add("asset", r.Asset)
	}

	endpoint = fmt.Sprintf(
		"accounts?%s",
		query.Encode(),
	)

	if pageParams := addQueryParams(cursor(r.Cursor), limit(r.Limit), r.Order); len(pageParams) > 0 {
		endpoint = fmt.Sprintf(
			"%s&%s",
			endpoint,
			pageParams,
		)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err
}
