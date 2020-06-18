package txsub

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/stellar/go/services/horizon/internal/codes"
	"github.com/stellar/go/xdr"
)

var (
	ErrNoResults = errors.New("No result found")
	ErrCanceled  = errors.New("canceled")
	ErrTimeout   = errors.New("timeout")

	// ErrBadSequence is a canned error response for transactions whose sequence
	// number is wrong.
	ErrBadSequence = &FailedTransactionError{"AAAAAAAAAAD////7AAAAAA=="}
	// ErrNoAccount is returned when the source account for the transaction
	// cannot be found in the database
	ErrNoAccount = &FailedTransactionError{"AAAAAAAAAAD////4AAAAAA=="}
)

// FailedTransactionError represent an error that occurred because
// stellar-core rejected the transaction.  ResultXDR is a base64
// encoded TransactionResult struct
type FailedTransactionError struct {
	ResultXDR string
}

func (err *FailedTransactionError) Error() string {
	return fmt.Sprintf("tx failed: %s", err.ResultXDR)
}

func (fte *FailedTransactionError) Result() (result xdr.TransactionResult, err error) {
	err = xdr.SafeUnmarshalBase64(fte.ResultXDR, &result)
	return
}

func (fte *FailedTransactionError) TransactionResultCode(transactionHash string) (result string, err error) {
	r, err := fte.Result()
	if err != nil {
		return
	}

	innerResult, ok := r.Result.GetInnerResultPair()
	if ok && transactionHash == hex.EncodeToString(innerResult.TransactionHash[:]) {
		result, err = codes.String(innerResult.Result.Result.Code)
		return
	}
	result, err = codes.String(r.Result.Code)
	return
}

func (fte *FailedTransactionError) OperationResultCodes() (result []string, err error) {
	r, err := fte.Result()
	if err != nil {
		return
	}
	oprs, ok := r.OperationResults()

	if !ok {
		return
	}

	result = make([]string, len(oprs))

	for i, opr := range oprs {
		result[i], err = codes.ForOperationResult(opr)
		if err != nil {
			return
		}
	}

	return
}
