package xdr

// Successful returns true if the transaction succeeded
func (r TransactionResult) Successful() bool {
	return r.Result.Code == TransactionResultCodeTxSuccess ||
		r.Result.Code == TransactionResultCodeTxFeeBumpInnerSuccess
}

// OperationResults returns the operation results for the transaction
func (r TransactionResult) OperationResults() ([]OperationResult, bool) {
	innerResults, ok := r.Result.GetInnerResultPair()
	if ok {
		return innerResults.Result.Result.GetResults()
	}
	return r.Result.GetResults()
}

// Successful returns true if the transaction succeeded
func (r TransactionResultPair) Successful() bool {
	return r.Result.Successful()
}

// OperationResults returns the operation results for the transaction
func (r TransactionResultPair) OperationResults() ([]OperationResult, bool) {
	return r.Result.OperationResults()
}

// InnerHash returns the hash of the inner transaction.
// This function can only be called on fee bump transactions.
func (r TransactionResultPair) InnerHash() Hash {
	return r.Result.Result.MustInnerResultPair().TransactionHash
}
