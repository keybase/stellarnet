package resourceadapter

import (
	. "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
)

func PopulateAccountFlags(dest *AccountFlags, row core.Account) {
	dest.AuthRequired = row.IsAuthRequired()
	dest.AuthRevocable = row.IsAuthRevocable()
	dest.AuthImmutable = row.IsAuthImmutable()
}
