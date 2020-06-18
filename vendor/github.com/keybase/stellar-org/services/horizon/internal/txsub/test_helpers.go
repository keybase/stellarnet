package txsub

// This file provides mock implementations for the txsub interfaces
// which are useful in a testing context.
//
// NOTE:  this file is not a test file so that other packages may import
// txsub and use these mocks in their own tests

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockSubmitter is a test helper that simplements the Submitter interface
type MockSubmitter struct {
	R              SubmissionResult
	WasSubmittedTo bool
}

// Submit implements `txsub.Submitter`
func (sub *MockSubmitter) Submit(ctx context.Context, env string) SubmissionResult {
	sub.WasSubmittedTo = true
	return sub.R
}

// MockResultProvider is a test helper that simplements the ResultProvider
// interface
type MockResultProvider struct {
	Results            []Result
	ResultForInnerHash map[string]Result
}

// ResultByHash implements `txsub.ResultProvider`
func (results *MockResultProvider) ResultByHash(ctx context.Context, hash string) Result {
	if r, ok := results.ResultForInnerHash[hash]; ok {
		return r
	}

	if len(results.Results) > 0 {
		r := results.Results[0]
		results.Results = results.Results[1:]
		return r
	}

	return Result{Err: ErrNoResults}
}

// MockSequenceProvider is a test helper that simplements the SequenceProvider
// interface
type MockSequenceProvider struct {
	mock.Mock
}

// Get implements `txsub.SequenceProvider`
func (o *MockSequenceProvider) GetSequenceNumbers(addresses []string) (map[string]uint64, error) {
	args := o.Called(addresses)
	return args.Get(0).(map[string]uint64), args.Error(1)
}
