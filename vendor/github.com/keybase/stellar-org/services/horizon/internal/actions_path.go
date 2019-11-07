package horizon

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/core"
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
	maxAssetsParamLength int
	pathFinder           paths.Finder
	coreQ                *core.Q
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

	query := paths.Query{}
	var err error
	query.DestinationAmount, err = actions.GetPositiveAmount(r, "destination_amount")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	sourceAccount, err := getAccountID(r, "source_account", false)
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	query.SourceAssets, err = actions.GetAssets(r, "source_assets")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	if (len(query.SourceAssets) > 0) == (len(sourceAccount) > 0) {
		problem.Render(ctx, w, sourceAssetsOrSourceAccount)
		return
	}

	if len(query.SourceAssets) > handler.maxAssetsParamLength {
		p := problem.MakeInvalidFieldProblem(
			"source_assets",
			fmt.Errorf("list of assets exceeds maximum length of %d", handler.maxPathLength),
		)
		problem.Render(ctx, w, p)
		return
	}

	if query.DestinationAsset, err = actions.GetAsset(r, "destination_"); err != nil {
		problem.Render(ctx, w, err)
		return
	}

	if sourceAccount != "" {
		sourceAccount := xdr.MustAddress(sourceAccount)
		query.SourceAccount = &sourceAccount
		query.ValidateSourceBalance = true
		query.SourceAssets, query.SourceAssetBalances, err = handler.coreQ.AssetsForAddress(
			query.SourceAccount.Address(),
		)
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
		records, err = handler.pathFinder.Find(query, handler.maxPathLength)
		if err == simplepath.ErrEmptyInMemoryOrderBook {
			err = horizonProblem.StillIngesting
		}
		if err != nil {
			problem.Render(ctx, w, err)
			return
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
	pathFinder           paths.Finder
	coreQ                *core.Q
}

var destinationAssetsOrDestinationAccount = problem.P{
	Type:   "bad_request",
	Title:  "Bad Request",
	Status: http.StatusBadRequest,
	Detail: "The request requires either a list of destination assets or a destination account. " +
		"Both fields cannot be present.",
}

// ServeHTTP implements the http.Handler interface
func (handler FindFixedPathsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	destinationAccount, err := getAccountID(r, "destination_account", false)
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	destinationAssets, err := actions.GetAssets(r, "destination_assets")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	if (len(destinationAccount) > 0) == (len(destinationAssets) > 0) {
		problem.Render(ctx, w, destinationAssetsOrDestinationAccount)
		return
	}

	if len(destinationAssets) > handler.maxAssetsParamLength {
		p := problem.MakeInvalidFieldProblem(
			"destination_assets",
			fmt.Errorf("list of assets exceeds maximum length of %d", handler.maxPathLength),
		)
		problem.Render(ctx, w, p)
		return
	}

	if destinationAccount != "" {
		destinationAssets, _, err = handler.coreQ.AssetsForAddress(
			destinationAccount,
		)
		if err != nil {
			problem.Render(ctx, w, err)
			return
		}
	}

	sourceAsset, err := actions.GetAsset(r, "source_")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	amountToSpend, err := actions.GetPositiveAmount(r, "source_amount")
	if err != nil {
		problem.Render(ctx, w, err)
		return
	}

	records := []paths.Path{}
	if len(destinationAssets) > 0 {
		records, err = handler.pathFinder.FindFixedPaths(
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
	}

	renderPaths(ctx, records, w)
}
