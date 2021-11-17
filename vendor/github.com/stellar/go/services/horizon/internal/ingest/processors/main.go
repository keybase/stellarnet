package processors

import (
	"github.com/guregu/null"
	logpkg "github.com/stellar/go/support/log"
	"github.com/stellar/go/xdr"
)

var log = logpkg.DefaultLogger.WithField("service", "ingest")

const maxBatchSize = 100000

func ledgerEntrySponsorToNullString(entry xdr.LedgerEntry) null.String {
	sponsoringID := entry.SponsoringID()

	var sponsor null.String
	if sponsoringID != nil {
		sponsor.SetValid((*sponsoringID).Address())
	}

	return sponsor
}
