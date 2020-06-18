package core

import (
	"encoding/base64"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// IsAuthRequired returns true if the account has the "AUTH_REQUIRED" option
// turned on.
func (ac Account) IsAuthRequired() bool {
	return (ac.Flags & xdr.AccountFlagsAuthRequiredFlag) != 0
}

// IsAuthRevocable returns true if the account has the "AUTH_REVOCABLE" option
// turned on.
func (ac Account) IsAuthRevocable() bool {
	return (ac.Flags & xdr.AccountFlagsAuthRevocableFlag) != 0
}

// IsAuthImmutable returns true if the account has the "AUTH_IMMUTABLE" option
// turned on.
func (ac Account) IsAuthImmutable() bool {
	return (ac.Flags & xdr.AccountFlagsAuthImmutableFlag) != 0
}

// AccountByAddress loads a row from `accounts`, by address
func (q *Q) AccountByAddress(dest *Account, addy string) error {
	sql := selectAccount.Limit(1).Where("accountid = ?", addy)
	err := q.Get(dest, sql)
	if err != nil {
		return err
	}

	schemaVersion, err := q.SchemaVersion()
	if err != nil {
		return err
	}

	if schemaVersion >= 9 {
		// Since schema version 9, home_domain is base64 encoded.
		decoded, err := base64.StdEncoding.DecodeString(dest.HomeDomain.String)
		if err != nil {
			return errors.Wrap(err, "Unable to base64 decode HomeDomain")
		}
		dest.HomeDomain.String = string(decoded)
	}

	return nil
}

var selectAccount = sq.Select(
	"a.accountid",
	"a.balance",
	"a.seqnum",
	"a.numsubentries",
	"a.inflationdest",
	"a.homedomain",
	"a.thresholds",
	"a.flags",
	"a.lastmodified",
	// Liabilities can be NULL so can error without `coalesce`:
	// `Invalid value for xdr.Int64`
	"coalesce(a.buyingliabilities, 0) as buyingliabilities",
	"coalesce(a.sellingliabilities, 0) as sellingliabilities",
).From("accounts a")
