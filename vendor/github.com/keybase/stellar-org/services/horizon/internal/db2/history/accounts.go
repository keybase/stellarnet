package history

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// IsAuthRequired returns true if the account has the "AUTH_REQUIRED" option
// turned on.
func (account AccountEntry) IsAuthRequired() bool {
	return xdr.AccountFlags(account.Flags).IsAuthRequired()
}

// IsAuthRevocable returns true if the account has the "AUTH_REVOCABLE" option
// turned on.
func (account AccountEntry) IsAuthRevocable() bool {
	return xdr.AccountFlags(account.Flags).IsAuthRevocable()
}

// IsAuthImmutable returns true if the account has the "AUTH_IMMUTABLE" option
// turned on.
func (account AccountEntry) IsAuthImmutable() bool {
	return xdr.AccountFlags(account.Flags).IsAuthImmutable()
}

func (q *Q) CountAccounts() (int, error) {
	sql := sq.Select("count(*)").From("accounts")

	var count int
	if err := q.Get(&count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

func (q *Q) GetAccountByID(id string) (AccountEntry, error) {
	var account AccountEntry
	sql := selectAccounts.Where(sq.Eq{"account_id": id})
	err := q.Get(&account, sql)
	return account, err
}

func (q *Q) GetAccountsByIDs(ids []string) ([]AccountEntry, error) {
	var accounts []AccountEntry
	sql := selectAccounts.Where(map[string]interface{}{"accounts.account_id": ids})
	err := q.Select(&accounts, sql)
	return accounts, err
}

func accountToMap(account xdr.AccountEntry, lastModifiedLedger xdr.Uint32) map[string]interface{} {
	var buyingliabilities, sellingliabilities xdr.Int64
	if account.Ext.V1 != nil {
		v1 := account.Ext.V1
		buyingliabilities = v1.Liabilities.Buying
		sellingliabilities = v1.Liabilities.Selling
	}

	var inflationDestination = ""
	if account.InflationDest != nil {
		inflationDestination = account.InflationDest.Address()
	}

	return map[string]interface{}{
		"account_id":            account.AccountId.Address(),
		"balance":               account.Balance,
		"buying_liabilities":    buyingliabilities,
		"selling_liabilities":   sellingliabilities,
		"sequence_number":       account.SeqNum,
		"num_subentries":        account.NumSubEntries,
		"inflation_destination": inflationDestination,
		"flags":                 account.Flags,
		"home_domain":           account.HomeDomain,
		"master_weight":         account.MasterKeyWeight(),
		"threshold_low":         account.ThresholdLow(),
		"threshold_medium":      account.ThresholdMedium(),
		"threshold_high":        account.ThresholdHigh(),
		"last_modified_ledger":  lastModifiedLedger,
	}
}

// UpsertAccounts upserts a batch of accounts in the accounts table.
// There's currently no limit of the number of accounts this method can
// accept other than 2GB limit of the query string length what should be enough
// for each ledger with the current limits.
func (q *Q) UpsertAccounts(accounts []xdr.LedgerEntry) error {
	var accountID, inflationDestination []string
	var homeDomain []xdr.String32
	var balance, buyingLiabilities, sellingLiabilities []xdr.Int64
	var sequenceNumber []xdr.SequenceNumber
	var numSubEntries, flags, lastModifiedLedger []xdr.Uint32
	var masterWeight, thresholdLow, thresholdMedium, thresholdHigh []uint8

	for _, entry := range accounts {
		if entry.Data.Type != xdr.LedgerEntryTypeAccount {
			return errors.Errorf("Invalid entry type: %d", entry.Data.Type)
		}

		m := accountToMap(entry.Data.MustAccount(), entry.LastModifiedLedgerSeq)

		accountID = append(accountID, m["account_id"].(string))
		balance = append(balance, m["balance"].(xdr.Int64))
		buyingLiabilities = append(buyingLiabilities, m["buying_liabilities"].(xdr.Int64))
		sellingLiabilities = append(sellingLiabilities, m["selling_liabilities"].(xdr.Int64))
		sequenceNumber = append(sequenceNumber, m["sequence_number"].(xdr.SequenceNumber))
		numSubEntries = append(numSubEntries, m["num_subentries"].(xdr.Uint32))
		inflationDestination = append(inflationDestination, m["inflation_destination"].(string))
		flags = append(flags, m["flags"].(xdr.Uint32))
		homeDomain = append(homeDomain, m["home_domain"].(xdr.String32))
		masterWeight = append(masterWeight, m["master_weight"].(uint8))
		thresholdLow = append(thresholdLow, m["threshold_low"].(uint8))
		thresholdMedium = append(thresholdMedium, m["threshold_medium"].(uint8))
		thresholdHigh = append(thresholdHigh, m["threshold_high"].(uint8))
		lastModifiedLedger = append(lastModifiedLedger, m["last_modified_ledger"].(xdr.Uint32))
	}

	sql := `
	WITH r AS
		(SELECT
			unnest(?::text[]),   /* account_id */
			unnest(?::bigint[]), /*	balance */
			unnest(?::bigint[]), /*	buying_liabilities */
			unnest(?::bigint[]), /*	selling_liabilities */
			unnest(?::bigint[]), /*	sequence_number */
			unnest(?::int[]),    /*	num_subentries */
			unnest(?::text[]),   /*	inflation_destination */
			unnest(?::int[]),    /*	flags */
			unnest(?::text[]),   /*	home_domain */
			unnest(?::int[]),    /*	master_weight */
			unnest(?::int[]),    /*	threshold_low */
			unnest(?::int[]),    /*	threshold_medium */
			unnest(?::int[]),    /*	threshold_high */
			unnest(?::int[])     /*	last_modified_ledger */
		)
	INSERT INTO accounts ( 
		account_id,
		balance,
		buying_liabilities,
		selling_liabilities,
		sequence_number,
		num_subentries,
		inflation_destination,
		flags,
		home_domain,
		master_weight,
		threshold_low,
		threshold_medium,
		threshold_high,
		last_modified_ledger
	)
	SELECT * from r 
	ON CONFLICT (account_id) DO UPDATE SET 
		account_id = excluded.account_id,
		balance = excluded.balance,
		buying_liabilities = excluded.buying_liabilities,
		selling_liabilities = excluded.selling_liabilities,
		sequence_number = excluded.sequence_number,
		num_subentries = excluded.num_subentries,
		inflation_destination = excluded.inflation_destination,
		flags = excluded.flags,
		home_domain = excluded.home_domain,
		master_weight = excluded.master_weight,
		threshold_low = excluded.threshold_low,
		threshold_medium = excluded.threshold_medium,
		threshold_high = excluded.threshold_high,
		last_modified_ledger = excluded.last_modified_ledger`

	_, err := q.ExecRaw(sql,
		pq.Array(accountID),
		pq.Array(balance),
		pq.Array(buyingLiabilities),
		pq.Array(sellingLiabilities),
		pq.Array(sequenceNumber),
		pq.Array(numSubEntries),
		pq.Array(inflationDestination),
		pq.Array(flags),
		pq.Array(homeDomain),
		pq.Array(masterWeight),
		pq.Array(thresholdLow),
		pq.Array(thresholdMedium),
		pq.Array(thresholdHigh),
		pq.Array(lastModifiedLedger))
	return err
}

// RemoveAccount deletes a row in the accounts table.
// Returns number of rows affected and error.
func (q *Q) RemoveAccount(accountID string) (int64, error) {
	sql := sq.Delete("accounts").Where(sq.Eq{"account_id": accountID})
	result, err := q.Exec(sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// AccountsForAsset returns a list of `AccountEntry` rows who are trustee to an
// asset
func (q *Q) AccountsForAsset(asset xdr.Asset, page db2.PageQuery) ([]AccountEntry, error) {
	var assetType, code, issuer string
	asset.MustExtract(&assetType, &code, &issuer)

	sql := sq.
		Select("accounts.*").
		From("accounts").
		Join("trust_lines ON accounts.account_id = trust_lines.account_id").
		Where(map[string]interface{}{
			"trust_lines.asset_type":   int32(asset.Type),
			"trust_lines.asset_issuer": issuer,
			"trust_lines.asset_code":   code,
		})

	sql, err := page.ApplyToUsingCursor(sql, "trust_lines.account_id", page.Cursor)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	var results []AccountEntry
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

// AccountEntriesForSigner returns a list of `AccountEntry` rows for a given signer
func (q *Q) AccountEntriesForSigner(signer string, page db2.PageQuery) ([]AccountEntry, error) {
	sql := sq.
		Select("accounts.*").
		From("accounts").
		Join("accounts_signers ON accounts.account_id = accounts_signers.account_id").
		Where(map[string]interface{}{
			"accounts_signers.signer": signer,
		})

	sql, err := page.ApplyToUsingCursor(sql, "accounts_signers.account_id", page.Cursor)
	if err != nil {
		return nil, errors.Wrap(err, "could not apply query to page")
	}

	var results []AccountEntry
	if err := q.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "could not run select query")
	}

	return results, nil
}

var selectAccounts = sq.Select(`
	account_id,
	balance,
	buying_liabilities,
	selling_liabilities,
	sequence_number,
	num_subentries,
	inflation_destination,
	flags,
	home_domain,
	master_weight,
	threshold_low,
	threshold_medium,
	threshold_high,
	last_modified_ledger
`).From("accounts")
