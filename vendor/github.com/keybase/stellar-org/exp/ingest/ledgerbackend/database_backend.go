package ledgerbackend

import (
	"database/sql"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	latestLedgerSeqQuery = "select ledgerseq, closetime from ledgerheaders order by ledgerseq desc limit 1"
	txHistoryQuery       = "select txbody, txresult, txmeta, txindex from txhistory where ledgerseq = ? "
	ledgerHeaderQuery    = "select ledgerhash, data from ledgerheaders where ledgerseq = ? "
	txFeeHistoryQuery    = "select txchanges, txindex from txfeehistory where ledgerseq = ? "
	upgradeHistoryQuery  = "select ledgerseq, upgradeindex, upgrade, changes from upgradehistory where ledgerseq = ? order by upgradeindex asc"
	orderBy              = "order by txindex asc"
	dbDriver             = "postgres"
)

// Ensure DatabaseBackend implements LedgerBackend
var _ LedgerBackend = (*DatabaseBackend)(nil)

// DatabaseBackend implements a database data store.
type DatabaseBackend struct {
	session session
}

func NewDatabaseBackend(dataSourceName string) (*DatabaseBackend, error) {
	session, err := createSession(dataSourceName)
	if err != nil {
		return nil, err
	}

	return &DatabaseBackend{session: session}, nil
}

func NewDatabaseBackendFromSession(session *db.Session) (*DatabaseBackend, error) {
	return &DatabaseBackend{session: session}, nil
}

// GetLatestLedgerSequence returns the most recent ledger sequence number present in the database.
func (dbb *DatabaseBackend) GetLatestLedgerSequence() (uint32, error) {
	var ledger []ledgerHeader
	err := dbb.session.SelectRaw(&ledger, latestLedgerSeqQuery)
	if err != nil {
		return 0, errors.Wrap(err, "couldn't select ledger sequence")
	}
	if len(ledger) == 0 {
		return 0, errors.New("no ledgers exist in ledgerheaders table")
	}

	return ledger[0].LedgerSeq, nil
}

// GetLedger returns the LedgerCloseMeta for the given ledger sequence number.
// The first returned value is false when the ledger does not exist in the database.
func (dbb *DatabaseBackend) GetLedger(sequence uint32) (bool, LedgerCloseMeta, error) {
	lcm := LedgerCloseMeta{}

	// Query - ledgerheader
	var lRow ledgerHeaderHistory

	err := dbb.session.GetRaw(&lRow, ledgerHeaderQuery, sequence)
	// Return errors...
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			// Ledger was not found
			return false, LedgerCloseMeta{}, nil
		default:
			return false, LedgerCloseMeta{}, errors.Wrap(err, "Error getting ledger header")
		}
	}

	// ...otherwise store the header
	lcm.LedgerHeader = xdr.LedgerHeaderHistoryEntry{
		Hash:   lRow.Hash,
		Header: lRow.Header,
		Ext:    xdr.LedgerHeaderHistoryEntryExt{},
	}

	// Query - txhistory
	var txhRows []txHistory
	err = dbb.session.SelectRaw(&txhRows, txHistoryQuery+orderBy, sequence)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting txHistory")
	}

	// ...otherwise store the data
	for i, tx := range txhRows {
		// Sanity check index. Note that first TXIndex in a ledger is 1
		if i != int(tx.TXIndex)-1 {
			return false, LedgerCloseMeta{}, errors.New("transactions read from DB history table are misordered")
		}

		lcm.TransactionEnvelope = append(lcm.TransactionEnvelope, tx.TXBody)
		lcm.TransactionResult = append(lcm.TransactionResult, tx.TXResult)
		lcm.TransactionMeta = append(lcm.TransactionMeta, tx.TXMeta)

		if lcm.LedgerHeader.Header.LedgerVersion < 10 && tx.TXMeta.V != 2 {
			return false, lcm,
				errors.New("TransactionMeta.V=2 is required in protocol version older than version 10. " +
					"Please process ledgers again using stellar-core with SUPPORTED_META_VERSION=2 in the config file.")
		}
	}

	// Query - txfeehistory
	var txfhRows []txFeeHistory
	err = dbb.session.SelectRaw(&txfhRows, txFeeHistoryQuery+orderBy, sequence)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting txFeeHistory")
	}

	// ...otherwise store the data
	for i, tx := range txfhRows {
		// Sanity check index. Note that first TXIndex in a ledger is 1
		if i != int(tx.TXIndex)-1 {
			return false, LedgerCloseMeta{}, errors.New("transactions read from DB fee history table are misordered")
		}
		lcm.TransactionFeeChanges = append(lcm.TransactionFeeChanges, tx.TXChanges)
	}

	// Query - upgradehistory
	var upgradeHistoryRows []upgradeHistory
	err = dbb.session.SelectRaw(&upgradeHistoryRows, upgradeHistoryQuery, sequence)
	// Return errors...
	if err != nil {
		return false, lcm, errors.Wrap(err, "Error getting upgradeHistoryRows")
	}

	// ...otherwise store the data
	var upgradesMeta []xdr.LedgerEntryChanges
	for _, upgradeHistoryRow := range upgradeHistoryRows {
		upgradesMeta = append(upgradesMeta, upgradeHistoryRow.Changes)
	}
	lcm.UpgradesMeta = upgradesMeta

	return true, lcm, nil
}

// CreateSession returns a new db.Session that connects to the given DB settings.
func createSession(dataSourceName string) (*db.Session, error) {
	if dataSourceName == "" {
		return nil, errors.New("missing DatabaseBackend.DataSourceName (e.g. \"postgres://stellar:postgres@localhost:8002/core\")")
	}

	return db.Open(dbDriver, dataSourceName)
}

// Close disconnects an active database session.
func (dbb *DatabaseBackend) Close() error {
	return dbb.session.Close()
}
