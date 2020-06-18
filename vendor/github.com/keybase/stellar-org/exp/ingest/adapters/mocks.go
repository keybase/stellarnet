package adapters

import (
	"context"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
)

type MockHistoryArchiveAdapter struct {
	mock.Mock
}

func (m *MockHistoryArchiveAdapter) GetLatestLedgerSequence() (uint32, error) {
	args := m.Called()
	return args.Get(0).(uint32), args.Error(1)
}

func (m *MockHistoryArchiveAdapter) BucketListHash(sequence uint32) (xdr.Hash, error) {
	args := m.Called(sequence)
	return args.Get(0).(xdr.Hash), args.Error(1)
}

func (m *MockHistoryArchiveAdapter) GetState(
	ctx context.Context, sequence uint32, maxStreamRetries int,
) (io.ChangeReader, error) {
	args := m.Called(ctx, sequence, maxStreamRetries)
	return args.Get(0).(io.ChangeReader), args.Error(1)
}
