package io

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

// StatsLedgerTransactionProcessor is a state processors that counts number of changes types
// and entry types.
type StatsLedgerTransactionProcessor struct {
	results StatsLedgerTransactionProcessorResults
}

// StatsLedgerTransactionProcessorResults contains results after running StatsLedgerTransactionProcessor.
type StatsLedgerTransactionProcessorResults struct {
	Transactions           int64
	TransactionsSuccessful int64
	TransactionsFailed     int64

	Operations             int64
	OperationsInSuccessful int64
	OperationsInFailed     int64

	OperationsCreateAccount            int64
	OperationsPayment                  int64
	OperationsPathPaymentStrictReceive int64
	OperationsManageSellOffer          int64
	OperationsCreatePassiveSellOffer   int64
	OperationsSetOptions               int64
	OperationsChangeTrust              int64
	OperationsAllowTrust               int64
	OperationsAccountMerge             int64
	OperationsInflation                int64
	OperationsManageData               int64
	OperationsBumpSequence             int64
	OperationsManageBuyOffer           int64
	OperationsPathPaymentStrictSend    int64
}

func (p *StatsLedgerTransactionProcessor) ProcessTransaction(transaction LedgerTransaction) error {
	p.results.Transactions++
	ops := int64(len(transaction.Envelope.Operations()))
	p.results.Operations += ops

	if transaction.Result.Successful() {
		p.results.TransactionsSuccessful++
		p.results.OperationsInSuccessful += ops

	} else {
		p.results.TransactionsFailed++
		p.results.OperationsInFailed += ops
	}

	for _, op := range transaction.Envelope.Operations() {
		switch op.Body.Type {
		case xdr.OperationTypeCreateAccount:
			p.results.OperationsCreateAccount++
		case xdr.OperationTypePayment:
			p.results.OperationsPayment++
		case xdr.OperationTypePathPaymentStrictReceive:
			p.results.OperationsPathPaymentStrictReceive++
		case xdr.OperationTypeManageSellOffer:
			p.results.OperationsManageSellOffer++
		case xdr.OperationTypeCreatePassiveSellOffer:
			p.results.OperationsCreatePassiveSellOffer++
		case xdr.OperationTypeSetOptions:
			p.results.OperationsSetOptions++
		case xdr.OperationTypeChangeTrust:
			p.results.OperationsChangeTrust++
		case xdr.OperationTypeAllowTrust:
			p.results.OperationsAllowTrust++
		case xdr.OperationTypeAccountMerge:
			p.results.OperationsAccountMerge++
		case xdr.OperationTypeInflation:
			p.results.OperationsInflation++
		case xdr.OperationTypeManageData:
			p.results.OperationsManageData++
		case xdr.OperationTypeBumpSequence:
			p.results.OperationsBumpSequence++
		case xdr.OperationTypeManageBuyOffer:
			p.results.OperationsManageBuyOffer++
		case xdr.OperationTypePathPaymentStrictSend:
			p.results.OperationsPathPaymentStrictSend++
		default:
			panic(fmt.Sprintf("Unkown operation type: %d", op.Body.Type))
		}
	}

	return nil
}

func (p *StatsLedgerTransactionProcessor) GetResults() StatsLedgerTransactionProcessorResults {
	return p.results
}

func (stats *StatsLedgerTransactionProcessorResults) Map() map[string]interface{} {
	return map[string]interface{}{
		"stats_transactions":            stats.Transactions,
		"stats_transactions_successful": stats.TransactionsSuccessful,
		"stats_transactions_failed":     stats.TransactionsFailed,

		"stats_operations":               stats.Operations,
		"stats_operations_in_successful": stats.OperationsInSuccessful,
		"stats_operations_in_failed":     stats.OperationsInFailed,

		"stats_operations_create_account":              stats.OperationsCreateAccount,
		"stats_operations_payment":                     stats.OperationsPayment,
		"stats_operations_path_payment_strict_receive": stats.OperationsPathPaymentStrictReceive,
		"stats_operations_manage_sell_offer":           stats.OperationsManageSellOffer,
		"stats_operations_create_passive_sell_offer":   stats.OperationsCreatePassiveSellOffer,
		"stats_operations_set_options":                 stats.OperationsSetOptions,
		"stats_operations_change_trust":                stats.OperationsChangeTrust,
		"stats_operations_allow_trust":                 stats.OperationsAllowTrust,
		"stats_operations_account_merge":               stats.OperationsAccountMerge,
		"stats_operations_inflation":                   stats.OperationsInflation,
		"stats_operations_manage_data":                 stats.OperationsManageData,
		"stats_operations_bump_sequence":               stats.OperationsBumpSequence,
		"stats_operations_manage_buy_offer":            stats.OperationsManageBuyOffer,
		"stats_operations_path_payment_strict_send":    stats.OperationsPathPaymentStrictSend,
	}
}
