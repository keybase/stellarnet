package horizon

import (
	"net/http"
	"strconv"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/support/render/problem"
)

// This file contains the actions:
//
// FeeStatsAction: stats representing current state of network fees

var _ actions.JSONer = (*FeeStatsAction)(nil)

// FeeStatsAction renders a few useful statistics that describe the
// current state of operation fees on the network.
type FeeStatsAction struct {
	Action
	feeStats hProtocol.FeeStats
}

// JSON is a method for actions.JSON
func (action *FeeStatsAction) JSON() error {
	if !action.App.config.IngestFailedTransactions {
		// If Horizon is not ingesting failed transaction it does not make sense to display
		// operation fee stats because they will be incorrect.
		p := problem.P{
			Type:   "endpoint_not_available",
			Title:  "Endpoint Not Available",
			Status: http.StatusNotImplemented,
			Detail: "/fee_stats is unavailable when Horizon is not ingesting failed " +
				"transactions. Set `INGEST_FAILED_TRANSACTIONS=true` to start ingesting them.",
		}
		problem.Render(action.R.Context(), action.W, p)
		return nil
	}

	action.Do(
		action.loadRecords,
		func() {
			httpjson.Render(
				action.W,
				action.feeStats,
				httpjson.HALJSON,
			)
		},
	)
	return action.Err
}

func (action *FeeStatsAction) loadRecords() {
	cur, ok := operationfeestats.CurrentState()
	action.feeStats.LastLedgerBaseFee = cur.LastBaseFee
	action.feeStats.LastLedger = cur.LastLedger

	// LedgerCapacityUsage is the empty string when operationfeestats has not had its state set
	if ok {
		action.feeStats.LedgerCapacityUsage, action.Err = strconv.ParseFloat(
			cur.LedgerCapacityUsage,
			64,
		)
		if action.Err != nil {
			return
		}
	}

	// FeeCharged
	action.feeStats.FeeCharged.Max = cur.FeeChargedMax
	action.feeStats.FeeCharged.Min = cur.FeeChargedMin
	action.feeStats.FeeCharged.Mode = cur.FeeChargedMode
	action.feeStats.FeeCharged.P10 = cur.FeeChargedP10
	action.feeStats.FeeCharged.P20 = cur.FeeChargedP20
	action.feeStats.FeeCharged.P30 = cur.FeeChargedP30
	action.feeStats.FeeCharged.P40 = cur.FeeChargedP40
	action.feeStats.FeeCharged.P50 = cur.FeeChargedP50
	action.feeStats.FeeCharged.P60 = cur.FeeChargedP60
	action.feeStats.FeeCharged.P70 = cur.FeeChargedP70
	action.feeStats.FeeCharged.P80 = cur.FeeChargedP80
	action.feeStats.FeeCharged.P90 = cur.FeeChargedP90
	action.feeStats.FeeCharged.P95 = cur.FeeChargedP95
	action.feeStats.FeeCharged.P99 = cur.FeeChargedP99

	// MaxFee
	action.feeStats.MaxFee.Max = cur.MaxFeeMax
	action.feeStats.MaxFee.Min = cur.MaxFeeMin
	action.feeStats.MaxFee.Mode = cur.MaxFeeMode
	action.feeStats.MaxFee.P10 = cur.MaxFeeP10
	action.feeStats.MaxFee.P20 = cur.MaxFeeP20
	action.feeStats.MaxFee.P30 = cur.MaxFeeP30
	action.feeStats.MaxFee.P40 = cur.MaxFeeP40
	action.feeStats.MaxFee.P50 = cur.MaxFeeP50
	action.feeStats.MaxFee.P60 = cur.MaxFeeP60
	action.feeStats.MaxFee.P70 = cur.MaxFeeP70
	action.feeStats.MaxFee.P80 = cur.MaxFeeP80
	action.feeStats.MaxFee.P90 = cur.MaxFeeP90
	action.feeStats.MaxFee.P95 = cur.MaxFeeP95
	action.feeStats.MaxFee.P99 = cur.MaxFeeP99
}
