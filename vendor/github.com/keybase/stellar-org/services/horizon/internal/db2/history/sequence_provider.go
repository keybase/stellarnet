package history

import (
	sq "github.com/Masterminds/squirrel"

	"github.com/stellar/go/support/errors"
)

func (q *Q) GetSequenceNumbers(addresses []string) (map[string]uint64, error) {
	var accounts []AccountEntry
	sql := sq.Select("account_id, sequence_number").From("accounts").
		Where(map[string]interface{}{"accounts.account_id": addresses})
	if err := q.Select(&accounts, sql); err != nil {
		return nil, errors.Wrap(err, "could not query accounts")
	}

	sequenceNumbers := map[string]uint64{}
	for _, account := range accounts {
		sequenceNumbers[account.AccountID] = uint64(account.SequenceNumber)
	}

	return sequenceNumbers, nil
}
