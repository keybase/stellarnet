package horizon

import (
	"fmt"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/render/sse"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	supportProblem "github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// This file contains the actions:
//
// OperationIndexAction: pages of operations
// OperationShowAction: single operation by id

// Interface verifications
var _ actions.JSONer = (*OperationIndexAction)(nil)
var _ actions.EventStreamer = (*OperationIndexAction)(nil)

// OperationIndexAction renders a page of operations resources, identified by
// a normal page query and optionally filtered by an account, ledger, or
// transaction.
type OperationIndexAction struct {
	Action
	LedgerFilter      int32
	AccountFilter     string
	TransactionFilter string
	PagingParams      db2.PageQuery
	Records           []history.Operation
	Ledgers           *history.LedgerCache
	Page              hal.Page
	IncludeFailed     bool
}

// JSON is a method for actions.JSON
func (action *OperationIndexAction) JSON() error {
	action.Do(
		action.EnsureHistoryFreshness,
		action.loadParams,
		action.ValidateCursorWithinHistory,
		action.loadRecords,
		action.loadLedgers,
		action.loadPage,
		func() { hal.Render(action.W, action.Page) },
	)
	return action.Err
}

// SSE is a method for actions.SSE
func (action *OperationIndexAction) SSE(stream *sse.Stream) error {
	action.Setup(
		action.EnsureHistoryFreshness,
		action.loadParams,
		action.ValidateCursorWithinHistory,
	)
	action.Do(
		action.loadRecords,
		action.loadLedgers,
		func() {
			stream.SetLimit(int(action.PagingParams.Limit))
			records := action.Records[stream.SentCount():]
			for _, record := range records {
				ledger, found := action.Ledgers.Records[record.LedgerSequence()]
				if !found {
					action.Err = errors.New(fmt.Sprintf("could not find ledger data for sequence %d", record.LedgerSequence()))
					return
				}

				res, err := resourceadapter.NewOperation(action.R.Context(), record, ledger)
				if err != nil {
					action.Err = err
					return
				}

				stream.Send(sse.Event{
					ID:   res.PagingToken(),
					Data: res,
				})
			}
		},
	)

	return action.Err
}

func (action *OperationIndexAction) loadParams() {
	action.ValidateCursorAsDefault()
	action.AccountFilter = action.GetAddress("account_id")
	action.LedgerFilter = action.GetInt32("ledger_id")
	action.TransactionFilter = action.GetStringFromURLParam("tx_id")
	action.PagingParams = action.GetPageQuery()
	action.IncludeFailed = action.GetBool("include_failed")

	filters, err := countNonEmpty(
		action.AccountFilter,
		action.LedgerFilter,
		action.TransactionFilter,
	)

	if err != nil {
		action.Err = errors.Wrap(err, "Error in countNonEmpty")
		return
	}

	if filters > 1 {
		action.Err = supportProblem.BadRequest
		return
	}

	// Double check TransactionFilter as it's used to determine if failed txs should be returned
	if action.TransactionFilter != "" && !isValidTransactionHash(action.TransactionFilter) {
		action.Err = supportProblem.MakeInvalidFieldProblem("tx_id", errors.New("Invalid transaction hash"))
		return
	}

	if action.IncludeFailed == true && !action.App.config.IngestFailedTransactions {
		err := errors.New("`include_failed` parameter is unavailable when Horizon is not ingesting failed " +
			"transactions. Set `INGEST_FAILED_TRANSACTIONS=true` to start ingesting them.")
		action.Err = supportProblem.MakeInvalidFieldProblem("include_failed", err)
		return
	}
}

func (action *OperationIndexAction) loadRecords() {
	q := action.HistoryQ()
	ops := q.Operations()

	switch {
	case action.AccountFilter != "":
		ops.ForAccount(action.AccountFilter)
	case action.LedgerFilter > 0:
		ops.ForLedger(action.LedgerFilter)
	case action.TransactionFilter != "":
		ops.ForTransaction(action.TransactionFilter)
	}

	// When querying operations for transaction return both successful
	// and failed operations. We assume that because user is querying
	// this specific transactions, she knows it's status.
	if action.TransactionFilter != "" || action.IncludeFailed {
		ops.IncludeFailed()
	}

	action.Err = ops.Page(action.PagingParams).Select(&action.Records)
	if action.Err != nil {
		return
	}

	for _, o := range action.Records {
		if !action.IncludeFailed && action.TransactionFilter == "" {
			if !o.IsTransactionSuccessful() {
				action.Err = errors.Errorf("Corrupted data! `include_failed=false` but returned transaction in /operations is failed: %s", o.TransactionHash)
				return
			}

			var resultXDR xdr.TransactionResult
			action.Err = xdr.SafeUnmarshalBase64(o.TxResult, &resultXDR)
			if action.Err != nil {
				return
			}

			if resultXDR.Result.Code != xdr.TransactionResultCodeTxSuccess {
				action.Err = errors.Errorf("Corrupted data! `include_failed=false` but returned transaction /operations is failed: %s %s", o.TransactionHash, o.TxResult)
				return
			}
		}
	}
}

// loadLedgers populates the ledger cache for this action
func (action *OperationIndexAction) loadLedgers() {
	action.Ledgers = &history.LedgerCache{}
	for _, op := range action.Records {
		action.Ledgers.Queue(op.LedgerSequence())
	}
	action.Err = action.Ledgers.Load(action.HistoryQ())
}

func (action *OperationIndexAction) loadPage() {
	for _, record := range action.Records {
		ledger, found := action.Ledgers.Records[record.LedgerSequence()]
		if !found {
			msg := fmt.Sprintf("could not find ledger data for sequence %d", record.LedgerSequence())
			action.Err = errors.New(msg)
			return
		}

		var res hal.Pageable
		res, action.Err = resourceadapter.NewOperation(action.R.Context(), record, ledger)
		if action.Err != nil {
			return
		}
		action.Page.Add(res)
	}

	action.Page.FullURL = action.FullURL()
	action.Page.Limit = action.PagingParams.Limit
	action.Page.Cursor = action.PagingParams.Cursor
	action.Page.Order = action.PagingParams.Order
	action.Page.PopulateLinks()
}

// Interface verification
var _ actions.JSONer = (*OperationShowAction)(nil)

// OperationShowAction renders a ledger found by its sequence number.
type OperationShowAction struct {
	Action
	ID       int64
	Record   history.Operation
	Ledger   history.Ledger
	Resource interface{}
}

func (action *OperationShowAction) loadParams() {
	action.ID = action.GetInt64("id")
}

func (action *OperationShowAction) loadRecord() {
	action.Err = action.HistoryQ().OperationByID(&action.Record, action.ID)
}

func (action *OperationShowAction) loadLedger() {
	action.Err = action.HistoryQ().LedgerBySequence(&action.Ledger, action.Record.LedgerSequence())
}

func (action *OperationShowAction) loadResource() {
	action.Resource, action.Err = resourceadapter.NewOperation(action.R.Context(), action.Record, action.Ledger)
}

// JSON is a method for actions.JSON
func (action *OperationShowAction) JSON() error {
	action.Do(
		action.EnsureHistoryFreshness,
		action.loadParams,
		action.verifyWithinHistory,
		action.loadRecord,
		action.loadLedger,
		action.loadResource,
		func() { hal.Render(action.W, action.Resource) },
	)
	return action.Err
}

func (action *OperationShowAction) verifyWithinHistory() {
	parsed := toid.Parse(action.ID)
	if parsed.LedgerSequence < ledger.CurrentState().HistoryElder {
		action.Err = &problem.BeforeHistory
	}
}
