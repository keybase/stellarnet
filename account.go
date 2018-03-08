package stellarnet

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
)

var client = horizon.DefaultPublicNetClient
var network = build.PublicNetwork

type Account struct {
	address  AddressStr
	internal *horizon.Account
}

func NewAccount(address AddressStr) *Account {
	return &Account{address: address}
}

func (a *Account) load() error {
	internal, err := client.LoadAccount(a.address.String())
	if err != nil {
		if herr, ok := err.(*horizon.Error); ok {
			if herr.Problem.Status == 404 {
				return ErrAccountNotFound
			}
		}
		return err
	}

	a.internal = &internal

	return nil
}

func (a *Account) BalanceXLM() (string, error) {
	if err := a.load(); err != nil {
		return "", err
	}

	return a.internal.GetNativeBalance(), nil
}

func (a *Account) RecentPayments() ([]string, error) {
	link, err := a.paymentsLink()
	if err != nil {
		return nil, err
	}
	res, err := client.HTTP.Get(link + "?order=desc&limit=10")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var page PaymentsPage
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		return nil, err
	}

	var payments []string
	for _, rec := range page.Embedded.Records {
		var s string
		switch rec.Type {
		case "create_account":
			s = fmt.Sprintf("%s\tcreate account %s, starting balance: %s", rec.ID, rec.Account, rec.StartingBalance)
		case "payment":
			s = fmt.Sprintf("%s\tpayment from %s to %s: %s", rec.ID, rec.From, rec.To, rec.Amount)
		}
		payments = append(payments, s)
	}
	return payments, nil
}

func (a *Account) RecentTransactions() ([]Transaction, error) {
	link, err := a.transactionsLink()
	if err != nil {
		return nil, err
	}
	res, err := client.HTTP.Get(link + "?order=desc&limit=10")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var page TransactionsPage
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		return nil, err
	}

	transactions := make([]Transaction, len(page.Embedded.Records))
	for i := 0; i < len(page.Embedded.Records); i++ {
		transactions[i] = Transaction{Internal: page.Embedded.Records[i]}
		ops, err := a.loadOperations(transactions[i])
		if err != nil {
			return nil, err
		}
		transactions[i].Operations = ops
	}

	return transactions, nil
}

func (a *Account) loadOperations(tx Transaction) ([]Operation, error) {
	link := a.linkHref(tx.Internal.Links.Operations)
	res, err := client.HTTP.Get(link)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var page OperationsPage
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		return nil, err
	}
	return page.Embedded.Records, nil
}

func (a *Account) IsOpNoDestination(inErr error) bool {
	herr, ok := inErr.(*horizon.Error)
	if !ok {
		return false
	}
	resultCodes, err := herr.ResultCodes()
	if err != nil {
		return false
	}
	if resultCodes.TransactionCode != "tx_failed" {
		return false
	}
	if len(resultCodes.OperationCodes) != 1 {
		// only handle one operation now
		return false
	}
	return resultCodes.OperationCodes[0] == "op_no_destination"
}

func (a *Account) Send(from SeedStr, to AddressStr, amount string) (ledger int32, err error) {
	// try payment first
	ledger, err = a.payment(from, to, amount)

	if err != nil {
		if !a.IsOpNoDestination(err) {
			return 0, err
		}

		// if payment failed due to op_no_destination, then
		// should try createAccount instead
		return a.createAccount(from, to, amount)
	}

	return ledger, nil
}

func (a *Account) payment(from SeedStr, to AddressStr, amount string) (ledger int32, err error) {
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: from.String()},
		network,
		build.AutoSequence{SequenceProvider: client},
		build.Payment(
			build.Destination{AddressOrSeed: to.String()},
			build.NativeAmount{Amount: amount},
		),
		build.MemoText{Value: "via keybase"},
	)
	if err != nil {
		return 0, err
	}

	return a.signAndSubmit(from, tx)
}

func (a *Account) createAccount(from SeedStr, to AddressStr, amount string) (ledger int32, err error) {
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: from.String()},
		network,
		build.AutoSequence{SequenceProvider: client},
		build.CreateAccount(
			build.Destination{AddressOrSeed: to.String()},
			build.NativeAmount{Amount: amount},
		),
		build.MemoText{Value: "via keybase"},
	)
	if err != nil {
		return 0, err
	}

	return a.signAndSubmit(from, tx)
}

func (a *Account) signAndSubmit(from SeedStr, tx *build.TransactionBuilder) (ledger int32, err error) {
	txe, err := tx.Sign(from.String())
	if err != nil {
		return 0, err
	}

	txeB64, err := txe.Base64()
	if err != nil {
		return 0, err
	}

	resp, err := client.SubmitTransaction(txeB64)
	if err != nil {
		return 0, err
	}

	return resp.Ledger, nil
}

func (a *Account) paymentsLink() (string, error) {
	if a.internal == nil {
		if err := a.load(); err != nil {
			return "", err
		}
	}

	return a.linkHref(a.internal.Links.Payments), nil
}

func (a *Account) transactionsLink() (string, error) {
	if a.internal == nil {
		if err := a.load(); err != nil {
			return "", err
		}
	}

	return a.linkHref(a.internal.Links.Transactions), nil
}

func (a *Account) linkHref(link horizon.Link) string {
	if link.Templated {
		return strings.Split(link.Href, "{")[0]
	}
	return link.Href

}
