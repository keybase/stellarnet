package horizon

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/paths"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	horizonProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/simplepath"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// FindPathsHandler is the http handler for the find payment paths endpoint
type FindPathsHandler struct {
	staleThreshold       uint
	maxPathLength        uint
	checkHistoryIsStale  bool
	setLastLedgerHeader  bool
	maxAssetsParamLength int
	pathFinder           paths.Finder
	historyQ             *history.Q
}

// StrictReceivePathsQuery query struct for paths/strict-send end-point
type StrictReceivePathsQuery struct {
	SourceAssets           string `schema:"source_assets" valid:"-"`
	SourceAccount          string `schema:"source_account" valid:"accountID,optional"`
	DestinationAccount     string `schema:"destination_account" valid:"accountID,optional"`
	DestinationAssetType   string `schema:"destination_asset_type" valid:"assetType"`
	DestinationAssetIssuer string `schema:"destination_asset_issuer" valid:"accountID,optional"`
	DestinationAssetCode   string `schema:"destination_asset_code" valid:"-"`
	DestinationAmount      string `schema:"destination_amount" valid:"amount"`
}

// Assets returns a list of xdr.Asset
func (q StrictReceivePathsQuery) Assets() ([]xdr.Asset, error) {
	return xdr.BuildAssets(q.SourceAssets)
}

// Amount returns source amount
func (q StrictReceivePathsQuery) Amount() xdr.Int64 {
	parsed, err := amount.Parse(q.DestinationAmount)
	if err != nil {
		panic(err)
	}
	return parsed
}

// DestinationAsset returns an xdr.Asset
func (q StrictReceivePathsQuery) DestinationAsset() xdr.Asset {
	asset, err := xdr.BuildAsset(
		q.DestinationAssetType,
		q.DestinationAssetIssuer,
		q.DestinationAssetCode,
	)

	if err != nil {
		panic(err)
	}

	return asset
}

// URITemplate returns a rfc6570 URI template for the query struct
func (q StrictReceivePathsQuery) URITemplate() string {
	return "/paths/strict-receive{?" + strings.Join(actions.GetURIParams(&q, false), ",") + "}"
}

// Validate runs custom validations.
func (q StrictReceivePathsQuery) Validate() error {
	if (len(q.SourceAccount) > 0) == (len(q.SourceAssets) > 0) {
		return sourceAssetsOrSourceAccount
	}

	err := actions.ValidateAssetParams(
		q.DestinationAssetType,
		q.DestinationAssetCode,
		q.DestinationAssetIssuer,
		"destination_",
	)

	if err != nil {
		return err
	}

	_, err = q.Assets()

	if err != nil {
		return problem.MakeInvalidFieldProblem(
			"source_assets",
			err,
		)
	}

	return nil
}

var sourceAssetsOrSourceAccount = problem.P{
	Type:   "bad_request",
	Title:  "Bad Request",
	Status: http.StatusBadRequest,
	Detail: "The request requires either a list of source assets or a source account. " +
		"Both fields cannot be present.",
}

// ServeHTTP implements the http.Handler interface
func (handler FindPathsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if handler.checkHistoryIsStale && isHistoryStale(handler.staleThreshold) {
		ls := ledger.CurrentState()
		err := hProblem.StaleHistory
		err.Extras = map[string]interface{}{
			"history_latest_ledger": ls.HistoryLatest,
			"core_latest_ledger":    ls.CoreLatest,
		}
		problem.Render(ctx, w, err)
		return
	}
	qp := StrictReceivePathsQuery{}
	err := actions.GetParams(&qp, r)
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	query := paths.Query{}
	query.DestinationAmount = qp.Amount()
	sourceAccount := qp.SourceAccount
	query.SourceAssets, _ = qp.Assets()

	if len(query.SourceAssets) > handler.maxAssetsParamLength {
		p := problem.MakeInvalidFieldProblem(
			"source_assets",
			fmt.Errorf("list of assets exceeds maximum length of %d", handler.maxPathLength),
		)
		problem.Render(ctx, w, p)
		return
	}
	query.DestinationAsset = qp.DestinationAsset()

	if sourceAccount != "" {
		sourceAccount := xdr.MustAddress(sourceAccount)
		query.SourceAccount = &sourceAccount
		query.ValidateSourceBalance = true
		query.SourceAssets, query.SourceAssetBalances, err = assetsForAddress(handler.historyQ, query.SourceAccount.Address())
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}
	} else {
		for range query.SourceAssets {
			query.SourceAssetBalances = append(query.SourceAssetBalances, 0)
		}
	}

	records := []paths.Path{}
	if len(query.SourceAssets) > 0 {
		var lastIngestedLedger uint32
		records, lastIngestedLedger, err = handler.pathFinder.Find(query, handler.maxPathLength)
		if err == simplepath.ErrEmptyInMemoryOrderBook {
			err = horizonProblem.StillIngesting
		}
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		if handler.setLastLedgerHeader {
			// To make the Last-Ledger header consistent with the response content,
			// we need to extract it from the ledger and not the DB.
			// Thus, we overwrite the header if it was previously set.
			actions.SetLastLedgerHeader(w, lastIngestedLedger)
		}
	}

	renderPaths(ctx, records, w)
}

func renderPaths(ctx context.Context, records []paths.Path, w http.ResponseWriter) {
	var page hal.BasePage
	page.Init()
	for _, p := range records {
		var res horizon.Path
		err := resourceadapter.PopulatePath(ctx, &res, p)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}
		page.Add(res)
	}
	httpjson.Render(w, page, httpjson.HALJSON)
}

// FindFixedPathsHandler is the http handler for the find fixed payment paths endpoint
// Fixed payment paths are payment paths where both the source and destination asset are fixed
type FindFixedPathsHandler struct {
	maxPathLength        uint
	maxAssetsParamLength int
	setLastLedgerHeader  bool
	pathFinder           paths.Finder
	historyQ             *history.Q
}

var destinationAssetsOrDestinationAccount = problem.P{
	Type:   "bad_request",
	Title:  "Bad Request",
	Status: http.StatusBadRequest,
	Detail: "The request requires either a list of destination assets or a destination account. " +
		"Both fields cannot be present.",
}

// FindFixedPathsQuery query struct for paths/strict-send end-point
type FindFixedPathsQuery struct {
	DestinationAccount string `schema:"destination_account" valid:"accountID,optional"`
	DestinationAssets  string `schema:"destination_assets" valid:"-"`
	SourceAssetType    string `schema:"source_asset_type" valid:"assetType"`
	SourceAssetIssuer  string `schema:"source_asset_issuer" valid:"accountID,optional"`
	SourceAssetCode    string `schema:"source_asset_code" valid:"-"`
	SourceAmount       string `schema:"source_amount" valid:"amount"`
}

// URITemplate returns a rfc6570 URI template for the query struct
func (q FindFixedPathsQuery) URITemplate() string {
	return "/paths/strict-send{?" + strings.Join(actions.GetURIParams(&q, false), ",") + "}"
}

// Validate runs custom validations.
func (q FindFixedPathsQuery) Validate() error {
	if (len(q.DestinationAccount) > 0) == (len(q.DestinationAssets) > 0) {
		return destinationAssetsOrDestinationAccount
	}

	err := actions.ValidateAssetParams(
		q.SourceAssetType,
		q.SourceAssetCode,
		q.SourceAssetIssuer,
		"source_",
	)

	if err != nil {
		return err
	}

	_, err = q.Assets()

	if err != nil {
		return problem.MakeInvalidFieldProblem(
			"destination_assets",
			err,
		)
	}

	return nil
}

// Assets returns a list of xdr.Asset
func (q FindFixedPathsQuery) Assets() ([]xdr.Asset, error) {
	return xdr.BuildAssets(q.DestinationAssets)
}

// Amount returns source amount
func (q FindFixedPathsQuery) Amount() xdr.Int64 {
	parsed, err := amount.Parse(q.SourceAmount)
	if err != nil {
		panic(err)
	}
	return parsed
}

// SourceAsset returns an xdr.Asset
func (q FindFixedPathsQuery) SourceAsset() xdr.Asset {
	asset, err := xdr.BuildAsset(
		q.SourceAssetType,
		q.SourceAssetIssuer,
		q.SourceAssetCode,
	)

	if err != nil {
		panic(err)
	}

	return asset
}

// ServeHTTP implements the http.Handler interface
func (handler FindFixedPathsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	qp := FindFixedPathsQuery{}
	err := actions.GetParams(&qp, r)
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	destinationAccount := qp.DestinationAccount
	destinationAssets, _ := qp.Assets()

	if len(destinationAssets) > handler.maxAssetsParamLength {
		p := problem.MakeInvalidFieldProblem(
			"destination_assets",
			fmt.Errorf("list of assets exceeds maximum length of %d", handler.maxPathLength),
		)
		problem.Render(ctx, w, p)
		return
	}

	if destinationAccount != "" {
		destinationAssets, _, err = assetsForAddress(handler.historyQ, destinationAccount)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}
	}

	sourceAsset := qp.SourceAsset()
	amountToSpend := qp.Amount()

	records := []paths.Path{}
	if len(destinationAssets) > 0 {
		var lastIngestedLedger uint32
		records, lastIngestedLedger, err = handler.pathFinder.FindFixedPaths(
			sourceAsset,
			amountToSpend,
			destinationAssets,
			handler.maxPathLength,
		)
		if err == simplepath.ErrEmptyInMemoryOrderBook {
			err = horizonProblem.StillIngesting
		}
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}

		if handler.setLastLedgerHeader {
			// To make the Last-Ledger header consistent with the response content,
			// we need to extract it from the ledger and not the DB.
			// Thus, we overwrite the header if it was previously set.
			actions.SetLastLedgerHeader(w, lastIngestedLedger)
		}
	}

	renderPaths(ctx, records, w)
}

func assetsForAddress(historyQ *history.Q, addy string) ([]xdr.Asset, []xdr.Int64, error) {
	err := historyQ.BeginTx(&sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, nil, err
	}
	defer historyQ.Rollback()
	return historyQ.AssetsForAddress(addy)
}
