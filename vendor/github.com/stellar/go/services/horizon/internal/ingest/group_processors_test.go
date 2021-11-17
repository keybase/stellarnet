//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/ingest"
)

var _ horizonChangeProcessor = (*mockHorizonChangeProcessor)(nil)

type mockHorizonChangeProcessor struct {
	mock.Mock
}

func (m *mockHorizonChangeProcessor) ProcessChange(ctx context.Context, change ingest.Change) error {
	args := m.Called(ctx, change)
	return args.Error(0)
}

func (m *mockHorizonChangeProcessor) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

var _ horizonTransactionProcessor = (*mockHorizonTransactionProcessor)(nil)

type mockHorizonTransactionProcessor struct {
	mock.Mock
}

func (m *mockHorizonTransactionProcessor) ProcessTransaction(ctx context.Context, transaction ingest.LedgerTransaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *mockHorizonTransactionProcessor) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type GroupChangeProcessorsTestSuiteLedger struct {
	suite.Suite
	ctx        context.Context
	processors *groupChangeProcessors
	processorA *mockHorizonChangeProcessor
	processorB *mockHorizonChangeProcessor
}

func TestGroupChangeProcessorsTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(GroupChangeProcessorsTestSuiteLedger))
}

func (s *GroupChangeProcessorsTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.processorA = &mockHorizonChangeProcessor{}
	s.processorB = &mockHorizonChangeProcessor{}
	s.processors = newGroupChangeProcessors([]horizonChangeProcessor{
		s.processorA,
		s.processorB,
	})
}

func (s *GroupChangeProcessorsTestSuiteLedger) TearDownTest() {
	s.processorA.AssertExpectations(s.T())
	s.processorB.AssertExpectations(s.T())
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestProcessChangeFails() {
	change := ingest.Change{}
	s.processorA.
		On("ProcessChange", s.ctx, change).
		Return(errors.New("transient error")).Once()

	err := s.processors.ProcessChange(s.ctx, change)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *ingest.mockHorizonChangeProcessor.ProcessChange: transient error")
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestProcessChangeSucceeds() {
	change := ingest.Change{}
	s.processorA.
		On("ProcessChange", s.ctx, change).
		Return(nil).Once()
	s.processorB.
		On("ProcessChange", s.ctx, change).
		Return(nil).Once()

	err := s.processors.ProcessChange(s.ctx, change)
	s.Assert().NoError(err)
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestCommitFails() {
	s.processorA.
		On("Commit", s.ctx).
		Return(errors.New("transient error")).Once()

	err := s.processors.Commit(s.ctx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *ingest.mockHorizonChangeProcessor.Commit: transient error")
}

func (s *GroupChangeProcessorsTestSuiteLedger) TestCommitSucceeds() {
	s.processorA.
		On("Commit", s.ctx).
		Return(nil).Once()
	s.processorB.
		On("Commit", s.ctx).
		Return(nil).Once()

	err := s.processors.Commit(s.ctx)
	s.Assert().NoError(err)
}

type GroupTransactionProcessorsTestSuiteLedger struct {
	suite.Suite
	ctx        context.Context
	processors *groupTransactionProcessors
	processorA *mockHorizonTransactionProcessor
	processorB *mockHorizonTransactionProcessor
}

func TestGroupTransactionProcessorsTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(GroupTransactionProcessorsTestSuiteLedger))
}

func (s *GroupTransactionProcessorsTestSuiteLedger) SetupTest() {
	s.ctx = context.Background()
	s.processorA = &mockHorizonTransactionProcessor{}
	s.processorB = &mockHorizonTransactionProcessor{}
	s.processors = newGroupTransactionProcessors([]horizonTransactionProcessor{
		s.processorA,
		s.processorB,
	})
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TearDownTest() {
	s.processorA.AssertExpectations(s.T())
	s.processorB.AssertExpectations(s.T())
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestProcessTransactionFails() {
	transaction := ingest.LedgerTransaction{}
	s.processorA.
		On("ProcessTransaction", s.ctx, transaction).
		Return(errors.New("transient error")).Once()

	err := s.processors.ProcessTransaction(s.ctx, transaction)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *ingest.mockHorizonTransactionProcessor.ProcessTransaction: transient error")
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestProcessTransactionSucceeds() {
	transaction := ingest.LedgerTransaction{}
	s.processorA.
		On("ProcessTransaction", s.ctx, transaction).
		Return(nil).Once()
	s.processorB.
		On("ProcessTransaction", s.ctx, transaction).
		Return(nil).Once()

	err := s.processors.ProcessTransaction(s.ctx, transaction)
	s.Assert().NoError(err)
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestCommitFails() {
	s.processorA.
		On("Commit", s.ctx).
		Return(errors.New("transient error")).Once()

	err := s.processors.Commit(s.ctx)
	s.Assert().Error(err)
	s.Assert().EqualError(err, "error in *ingest.mockHorizonTransactionProcessor.Commit: transient error")
}

func (s *GroupTransactionProcessorsTestSuiteLedger) TestCommitSucceeds() {
	s.processorA.
		On("Commit", s.ctx).
		Return(nil).Once()
	s.processorB.
		On("Commit", s.ctx).
		Return(nil).Once()

	err := s.processors.Commit(s.ctx)
	s.Assert().NoError(err)
}
