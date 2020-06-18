package actions

import (
	"context"
	"net/http"
	"strings"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// AccountInfo returns the information about an account identified by addr.
func AccountInfo(ctx context.Context, hq *history.Q, addr string) (*protocol.Account, error) {
	var (
		record     history.AccountEntry
		data       []history.Data
		signers    []history.AccountSigner
		trustlines []history.TrustLine
		resouce    protocol.Account
	)

	record, err := hq.GetAccountByID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history account record")
	}

	data, err = hq.GetAccountDataByAccountID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history account data")
	}

	signers, err = hq.GetAccountSignersByAccountID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history signers")
	}

	trustlines, err = hq.GetSortedTrustLinesByAccountID(addr)
	if err != nil {
		return nil, errors.Wrap(err, "getting history trustlines")
	}

	ledger, err := getLedgerBySequence(hq, int32(record.LastModifiedLedger))
	if err != nil {
		return nil, err
	}

	err = resourceadapter.PopulateAccountEntry(
		ctx,
		&resouce,
		record,
		data,
		signers,
		trustlines,
		ledger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "populating account entry")
	}

	return &resouce, nil
}

// AccountsQuery query struct for accounts end-point
type AccountsQuery struct {
	Signer      string `schema:"signer" valid:"accountID,optional"`
	AssetFilter string `schema:"asset" valid:"asset,optional"`
}

// URITemplate returns a rfc6570 URI template the query struct
func (q AccountsQuery) URITemplate() string {
	return "/accounts{?" + strings.Join(GetURIParams(&q, true), ",") + "}"
}

var invalidAccountsParams = problem.P{
	Type:   "invalid_accounts_params",
	Title:  "Invalid Accounts Parameters",
	Status: http.StatusBadRequest,
	Detail: "A filter is required. Please ensure that you are including a signer or an asset.",
}

// Validate runs custom validations.
func (q AccountsQuery) Validate() error {
	if q.AssetFilter == "native" {
		return problem.MakeInvalidFieldProblem(
			"asset",
			errors.New("you can't filter by asset: native"),
		)
	}

	if len(q.Signer) == 0 && q.Asset() == nil {
		return invalidAccountsParams
	}

	if len(q.Signer) > 0 && q.Asset() != nil {
		return problem.MakeInvalidFieldProblem(
			"signer",
			errors.New("you can't filter by signer and asset at the same time"),
		)
	}

	return nil
}

// Asset returns an xdr.Asset representing the Asset we want to find the trustees by.
func (q AccountsQuery) Asset() *xdr.Asset {
	if len(q.AssetFilter) == 0 {
		return nil
	}

	parts := strings.Split(q.AssetFilter, ":")
	asset := xdr.MustNewCreditAsset(parts[0], parts[1])

	return &asset
}

// GetAccountsHandler is the action handler for the /accounts endpoint
type GetAccountsHandler struct {
}

// GetResourcePage returns a page containing the account records that have
// `signer` as a signer or have a trustline to the given asset.
func (handler GetAccountsHandler) GetResourcePage(
	w HeaderWriter,
	r *http.Request,
) ([]hal.Pageable, error) {
	ctx := r.Context()
	pq, err := GetPageQuery(r, DisableCursorValidation)
	if err != nil {
		return nil, err
	}

	historyQ, err := HistoryQFromRequest(r)
	if err != nil {
		return nil, err
	}

	qp := AccountsQuery{}
	err = GetParams(&qp, r)
	if err != nil {
		return nil, err
	}

	var records []history.AccountEntry

	if len(qp.Signer) > 0 {
		records, err = historyQ.AccountEntriesForSigner(qp.Signer, pq)
		if err != nil {
			return nil, errors.Wrap(err, "loading account records")
		}
	} else {
		records, err = historyQ.AccountsForAsset(*qp.Asset(), pq)
		if err != nil {
			return nil, errors.Wrap(err, "loading account records")
		}
	}

	accounts := make([]hal.Pageable, 0, len(records))

	if len(records) == 0 {
		// early return
		return accounts, nil
	}

	accountIDs := make([]string, 0, len(records))
	for _, record := range records {
		accountIDs = append(accountIDs, record.AccountID)
	}

	signers, err := handler.loadSigners(historyQ, accountIDs)
	if err != nil {
		return nil, err
	}

	trustlines, err := handler.loadTrustlines(historyQ, accountIDs)
	if err != nil {
		return nil, err
	}

	data, err := handler.loadData(historyQ, accountIDs)
	if err != nil {
		return nil, err
	}

	ledgerCache := history.LedgerCache{}
	for _, record := range records {
		ledgerCache.Queue(int32(record.LastModifiedLedger))
	}

	if err := ledgerCache.Load(historyQ); err != nil {
		return nil, errors.Wrap(err, "failed to load ledger batch")
	}

	for _, record := range records {
		var res protocol.Account
		s := signers[record.AccountID]
		t := trustlines[record.AccountID]
		d := data[record.AccountID]
		var ledger *history.Ledger
		if l, ok := ledgerCache.Records[int32(record.LastModifiedLedger)]; ok {
			ledger = &l
		}
		resourceadapter.PopulateAccountEntry(ctx, &res, record, d, s, t, ledger)

		accounts = append(accounts, res)
	}

	return accounts, nil
}

func (handler GetAccountsHandler) loadData(historyQ *history.Q, accounts []string) (map[string][]history.Data, error) {
	data := make(map[string][]history.Data)

	records, err := historyQ.GetAccountDataByAccountsID(accounts)
	if err != nil {
		return data, errors.Wrap(err, "loading account data records by accounts id")
	}

	for _, record := range records {
		data[record.AccountID] = append(data[record.AccountID], record)
	}

	return data, nil
}

func (handler GetAccountsHandler) loadTrustlines(historyQ *history.Q, accounts []string) (map[string][]history.TrustLine, error) {
	trustLines := make(map[string][]history.TrustLine)

	records, err := historyQ.GetSortedTrustLinesByAccountIDs(accounts)
	if err != nil {
		return trustLines, errors.Wrap(err, "loading trustline records by accounts")
	}

	for _, record := range records {
		trustLines[record.AccountID] = append(trustLines[record.AccountID], record)
	}

	return trustLines, nil
}

func (handler GetAccountsHandler) loadSigners(historyQ *history.Q, accounts []string) (map[string][]history.AccountSigner, error) {
	signers := make(map[string][]history.AccountSigner)

	records, err := historyQ.SignersForAccounts(accounts)
	if err != nil {
		return signers, errors.Wrap(err, "loading account signers by account")
	}

	for _, record := range records {
		signers[record.Account] = append(signers[record.Account], record)
	}

	return signers, nil
}

func getLedgerBySequence(hq *history.Q, sequence int32) (*history.Ledger, error) {
	ledger := &history.Ledger{}
	err := hq.LedgerBySequence(ledger, sequence)
	switch {
	case hq.NoRows(err):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return ledger, nil
	}
}
