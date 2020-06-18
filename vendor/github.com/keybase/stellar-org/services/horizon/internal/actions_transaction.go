package horizon

import (
	"encoding/hex"
	"net/http"

	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/actions"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
	"github.com/stellar/go/services/horizon/internal/txsub"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

// Interface verification
var _ actions.JSONer = (*TransactionCreateAction)(nil)

type envelopeInfo struct {
	hash   string
	raw    string
	parsed xdr.TransactionEnvelope
}

func extractEnvelopeInfo(raw string, passphrase string) (envelopeInfo, error) {
	result := envelopeInfo{raw: raw}
	err := xdr.SafeUnmarshalBase64(raw, &result.parsed)
	if err != nil {
		return result, err
	}

	var hash [32]byte
	hash, err = network.HashTransactionInEnvelope(result.parsed, passphrase)
	if err != nil {
		return result, err
	}
	result.hash = hex.EncodeToString(hash[:])
	return result, nil
}

// TransactionCreateAction submits a transaction to the stellar-core network
// on behalf of the requesting client.
type TransactionCreateAction struct {
	Action
	TX       envelopeInfo
	Result   txsub.Result
	Resource horizon.Transaction
}

// JSON format action handler
func (action *TransactionCreateAction) JSON() error {
	action.Do(
		action.loadTX,
		action.loadResult,
		action.loadResource,
		func() { hal.Render(action.W, action.Resource) },
	)
	return action.Err
}

func (action *TransactionCreateAction) loadTX() {
	action.ValidateBodyType()
	if action.Err == nil {
		raw := action.GetString("tx")
		if info, err := extractEnvelopeInfo(raw, action.App.config.NetworkPassphrase); err != nil {
			action.Err = &problem.P{
				Type:   "transaction_malformed",
				Title:  "Transaction Malformed",
				Status: http.StatusBadRequest,
				Detail: "Horizon could not decode the transaction envelope in this " +
					"request. A transaction should be an XDR TransactionEnvelope struct " +
					"encoded using base64.  The envelope read from this request is " +
					"echoed in the `extras.envelope_xdr` field of this response for your " +
					"convenience.",
				Extras: map[string]interface{}{
					"envelope_xdr": raw,
				},
			}
		} else {
			action.TX = info
		}
	}
}

func (action *TransactionCreateAction) loadResult() {
	submission := action.App.submitter.Submit(
		action.R.Context(),
		action.TX.raw,
		action.TX.parsed,
		action.TX.hash,
	)

	select {
	case result := <-submission:
		action.Result = result
	case <-action.R.Context().Done():
		action.Err = &hProblem.Timeout
	}
}

func (action *TransactionCreateAction) loadResource() {
	if action.Result.Err == nil {
		action.Err = resourceadapter.PopulateTransaction(
			action.R.Context(),
			action.TX.hash,
			&action.Resource,
			action.Result.Transaction,
		)
		return
	}

	if action.Result.Err == txsub.ErrTimeout {
		action.Err = &hProblem.Timeout
		return
	}

	if action.Result.Err == txsub.ErrCanceled {
		action.Err = &hProblem.Timeout
		return
	}

	switch err := action.Result.Err.(type) {
	case *txsub.FailedTransactionError:
		rcr := horizon.TransactionResultCodes{}
		resourceadapter.PopulateTransactionResultCodes(
			action.R.Context(),
			action.TX.hash,
			&rcr,
			err,
		)

		action.Err = &problem.P{
			Type:   "transaction_failed",
			Title:  "Transaction Failed",
			Status: http.StatusBadRequest,
			Detail: "The transaction failed when submitted to the stellar network. " +
				"The `extras.result_codes` field on this response contains further " +
				"details.  Descriptions of each code can be found at: " +
				"https://www.stellar.org/developers/guides/concepts/list-of-operations.html",
			Extras: map[string]interface{}{
				"envelope_xdr": action.TX.raw,
				"result_xdr":   err.ResultXDR,
				"result_codes": rcr,
			},
		}
	default:
		action.Err = err
	}
}
