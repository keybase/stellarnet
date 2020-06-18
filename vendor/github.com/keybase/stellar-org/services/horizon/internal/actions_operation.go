package horizon

import (
	"fmt"
	"strings"

	"github.com/stellar/go/services/horizon/internal/actions"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/hal"
	supportProblem "github.com/stellar/go/support/render/problem"
)

const (
	joinTransactions = "transactions"
)

func parseJoinField(action *actions.Base) (map[string]bool, error) {
	join := action.GetString("join")
	validJoins := map[string]bool{}
	if join != "" {
		for _, part := range strings.Split(join, ",") {
			if part == joinTransactions {
				validJoins[joinTransactions] = true
			} else {
				return nil, supportProblem.MakeInvalidFieldProblem(
					"join",
					fmt.Errorf("it is not possible to join '%s'", part),
				)
			}
		}
	}
	return validJoins, nil
}

// This check has been moved to history/operation with unit test
func validateTransactionForOperation(transaction history.Transaction, operation history.Operation) error {
	if transaction.ID != operation.TransactionID {
		return errors.Errorf(
			"transaction id %v does not match transaction id in operation %v",
			transaction.ID,
			operation.TransactionID,
		)
	}
	if transaction.TransactionHash != operation.TransactionHash {
		return errors.Errorf(
			"transaction hash %v does not match transaction hash in operation %v",
			transaction.TransactionHash,
			operation.TransactionHash,
		)
	}
	if transaction.TxResult != operation.TxResult {
		return errors.Errorf(
			"transaction result %v does not match transaction result in operation %v",
			transaction.TxResult,
			operation.TxResult,
		)
	}
	if transaction.Successful != operation.TransactionSuccessful {
		return errors.Errorf(
			"transaction successful flag %v does not match transaction successful flag in operation %v",
			transaction.Successful,
			operation.TransactionSuccessful,
		)
	}

	return nil
}

// Interface verification
var _ actions.JSONer = (*OperationShowAction)(nil)

// OperationShowAction renders a page of operation resources.
type OperationShowAction struct {
	Action
	ID                  int64
	OperationRecord     history.Operation
	TransactionRecord   *history.Transaction
	Ledger              history.Ledger
	IncludeTransactions bool
	Resource            interface{}
}

func (action *OperationShowAction) loadParams() {
	action.ID = action.GetInt64("id")
	parsed, err := parseJoinField(&action.Action.Base)
	if err != nil {
		action.Err = err
		return
	}
	action.IncludeTransactions = parsed[joinTransactions]
}

func (action *OperationShowAction) loadRecord() {
	action.OperationRecord, action.TransactionRecord, action.Err = action.HistoryQ().OperationByID(
		action.IncludeTransactions, action.ID,
	)
	if action.Err != nil {
		return
	}

	if action.IncludeTransactions {
		if action.TransactionRecord == nil {
			action.Err = errors.Errorf("could not find transaction for operation %v", action.ID)
			return
		}
		action.Err = validateTransactionForOperation(*action.TransactionRecord, action.OperationRecord)
	}
}

func (action *OperationShowAction) loadLedger() {
	action.Err = action.HistoryQ().LedgerBySequence(&action.Ledger, action.OperationRecord.LedgerSequence())
}

func (action *OperationShowAction) loadResource() {
	action.Resource, action.Err = resourceadapter.NewOperation(
		action.R.Context(),
		action.OperationRecord,
		action.OperationRecord.TransactionHash,
		action.TransactionRecord,
		action.Ledger,
	)
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
