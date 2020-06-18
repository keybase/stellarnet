package processors

import (
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// ParticipantsProcessor is a processor which ingests various participants
// from different sources (transactions, operations, etc)
type ParticipantsProcessor struct {
	participantsQ  history.QParticipants
	sequence       uint32
	participantSet map[string]participant
}

func NewParticipantsProcessor(participantsQ history.QParticipants, sequence uint32) *ParticipantsProcessor {
	return &ParticipantsProcessor{
		participantsQ:  participantsQ,
		sequence:       sequence,
		participantSet: map[string]participant{},
	}
}

type participant struct {
	accountID      int64
	transactionSet map[int64]struct{}
	operationSet   map[int64]struct{}
}

func (p *participant) addTransactionID(id int64) {
	if p.transactionSet == nil {
		p.transactionSet = map[int64]struct{}{}
	}
	p.transactionSet[id] = struct{}{}
}

func (p *participant) addOperationID(id int64) {
	if p.operationSet == nil {
		p.operationSet = map[int64]struct{}{}
	}
	p.operationSet[id] = struct{}{}
}

func (p *ParticipantsProcessor) loadAccountIDs(participantSet map[string]participant) error {
	addresses := make([]string, 0, len(participantSet))
	for address := range participantSet {
		addresses = append(addresses, address)
	}

	addressToID, err := p.participantsQ.CreateAccounts(addresses, maxBatchSize)
	if err != nil {
		return errors.Wrap(err, "Could not create account ids")
	}

	for _, address := range addresses {
		id, ok := addressToID[address]
		if !ok {
			return errors.Errorf("no id found for account address %s", address)
		}

		participantForAddress := participantSet[address]
		participantForAddress.accountID = id
		participantSet[address] = participantForAddress
	}

	return nil
}

func participantsForChanges(
	changes xdr.LedgerEntryChanges,
) ([]xdr.AccountId, error) {
	var participants []xdr.AccountId

	for _, c := range changes {
		var participant *xdr.AccountId

		switch c.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			participant = participantsForLedgerEntry(c.MustCreated())
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			participant = participantsForLedgerKey(c.MustRemoved())
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			participant = participantsForLedgerEntry(c.MustUpdated())
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			participant = participantsForLedgerEntry(c.MustState())
		default:
			return nil, errors.Errorf("Unknown change type: %s", c.Type)
		}

		if participant != nil {
			participants = append(participants, *participant)
		}
	}

	return participants, nil
}

func participantsForLedgerEntry(le xdr.LedgerEntry) *xdr.AccountId {
	if le.Data.Type != xdr.LedgerEntryTypeAccount {
		return nil
	}
	aid := le.Data.MustAccount().AccountId
	return &aid
}

func participantsForLedgerKey(lk xdr.LedgerKey) *xdr.AccountId {
	if lk.Type != xdr.LedgerEntryTypeAccount {
		return nil
	}
	aid := lk.MustAccount().AccountId
	return &aid
}

func participantsForMeta(
	meta xdr.TransactionMeta,
) ([]xdr.AccountId, error) {
	var participants []xdr.AccountId
	if meta.Operations == nil {
		return participants, nil
	}

	for _, op := range *meta.Operations {
		var accounts []xdr.AccountId
		accounts, err := participantsForChanges(op.Changes)
		if err != nil {
			return nil, err
		}

		participants = append(participants, accounts...)
	}

	return participants, nil
}

func participantsForTransaction(
	sequence uint32,
	transaction io.LedgerTransaction,
) ([]xdr.AccountId, error) {
	participants := []xdr.AccountId{
		transaction.Envelope.SourceAccount().ToAccountId(),
	}
	if transaction.Envelope.IsFeeBump() {
		participants = append(participants, transaction.Envelope.FeeBumpAccount().ToAccountId())
	}

	p, err := participantsForMeta(transaction.Meta)
	if err != nil {
		return nil, err
	}
	participants = append(participants, p...)

	p, err = participantsForChanges(transaction.FeeChanges)
	if err != nil {
		return nil, err
	}
	participants = append(participants, p...)

	for opi, op := range transaction.Envelope.Operations() {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		p, err := operation.Participants()
		if err != nil {
			return nil, errors.Wrapf(
				err, "could not determin operation %v participants", operation.ID(),
			)
		}
		participants = append(participants, p...)
	}

	return dedupe(participants), nil
}

func (p *ParticipantsProcessor) addTransactionParticipants(
	participantSet map[string]participant,
	sequence uint32,
	transaction io.LedgerTransaction,
) error {
	transactionID := toid.New(int32(sequence), int32(transaction.Index), 0).ToInt64()
	transactionParticipants, err := participantsForTransaction(
		sequence,
		transaction,
	)
	if err != nil {
		return errors.Wrap(err, "Could not determine participants for transaction")
	}

	for _, participant := range transactionParticipants {
		address := participant.Address()
		entry := participantSet[address]
		entry.addTransactionID(transactionID)
		participantSet[address] = entry
	}

	return nil
}

func (p *ParticipantsProcessor) addOperationsParticipants(
	participantSet map[string]participant,
	sequence uint32,
	transaction io.LedgerTransaction,
) error {
	participants, err := operationsParticipants(transaction, sequence)
	if err != nil {
		return errors.Wrap(err, "could not determine operation participants")
	}

	for operationID, p := range participants {
		for _, participant := range p {
			address := participant.Address()
			entry := participantSet[address]
			entry.addOperationID(operationID)
			participantSet[address] = entry
		}
	}

	return nil
}

func (p *ParticipantsProcessor) insertDBTransactionParticipants(participantSet map[string]participant) error {
	batch := p.participantsQ.NewTransactionParticipantsBatchInsertBuilder(maxBatchSize)

	for _, entry := range participantSet {
		for transactionID := range entry.transactionSet {
			if err := batch.Add(transactionID, entry.accountID); err != nil {
				return errors.Wrap(err, "Could not insert transaction participant in db")
			}
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "Could not flush transaction participants to db")
	}
	return nil
}

func (p *ParticipantsProcessor) insertDBOperationsParticipants(participantSet map[string]participant) error {
	batch := p.participantsQ.NewOperationParticipantBatchInsertBuilder(maxBatchSize)

	for _, entry := range participantSet {
		for operationID := range entry.operationSet {
			if err := batch.Add(operationID, entry.accountID); err != nil {
				return errors.Wrap(err, "could not insert operation participant in db")
			}
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "could not flush operation participants to db")
	}
	return nil
}

func (p *ParticipantsProcessor) ProcessTransaction(transaction io.LedgerTransaction) (err error) {
	err = p.addTransactionParticipants(p.participantSet, p.sequence, transaction)
	if err != nil {
		return err
	}

	err = p.addOperationsParticipants(p.participantSet, p.sequence, transaction)
	if err != nil {
		return err
	}

	return nil
}

func (p *ParticipantsProcessor) Commit() (err error) {
	if len(p.participantSet) > 0 {
		if err = p.loadAccountIDs(p.participantSet); err != nil {
			return err
		}

		if err = p.insertDBTransactionParticipants(p.participantSet); err != nil {
			return err
		}

		if err = p.insertDBOperationsParticipants(p.participantSet); err != nil {
			return err
		}
	}

	return err
}
