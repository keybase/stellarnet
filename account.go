package stellarnet

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
)

var client = horizon.DefaultPublicNetClient
var network = build.PublicNetwork

var (
	ErrAccountNotFound = errors.New("account not found")
)

func NewKeyPair() (*keypair.Full, error) {
	return keypair.Random()
}

type Account struct {
	id           string
	paymentsLink string
}

func NewAccount(id string) *Account {
	return &Account{id: id}
}

func (a *Account) load() (*horizon.Account, error) {
	acct, err := client.LoadAccount(a.id)
	if err != nil {
		if herr, ok := err.(*horizon.Error); ok {
			if herr.Problem.Status == 404 {
				return nil, ErrAccountNotFound
			}
		}
		return nil, err
	}

	a.paymentsLink = strings.Split(acct.Links.Payments.Href, "{")[0]

	return &acct, nil
}

func (a *Account) BalanceXLM() (string, error) {
	acct, err := a.load()
	if err != nil {
		return "", err
	}

	return acct.GetNativeBalance(), nil
}

func (a *Account) RecentPayments() ([]string, error) {
	if a.paymentsLink == "" {
		if _, err := a.load(); err != nil {
			return nil, err
		}
		if a.paymentsLink == "" {
			return nil, errors.New("no payments link")
		}
	}

	res, err := client.HTTP.Get(a.paymentsLink + "?order=desc&limit=10")
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

func (a *Account) Send(from, to, amount string) error {
	// try payment first
	if err := a.payment(from, to, amount); err != nil {
		if !a.IsOpNoDestination(err) {
			return err
		}

		// if payment failed due to op_no_destination, then
		// should try createAccount instead
		return a.createAccount(from, to, amount)
	}

	return nil
}

func (a *Account) payment(from, to, amount string) error {
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: from},
		network,
		build.AutoSequence{SequenceProvider: client},
		build.Payment(
			build.Destination{AddressOrSeed: to},
			build.NativeAmount{Amount: amount},
		),
		build.MemoText{Value: "via keybase"},
	)
	if err != nil {
		return err
	}

	txe, err := tx.Sign(from)
	if err != nil {
		return err
	}

	txeB64, err := txe.Base64()
	if err != nil {
		return err
	}

	resp, err := client.SubmitTransaction(txeB64)
	if err != nil {
		return err
	}

	fmt.Println("transaction posted in ledger:", resp.Ledger)
	return nil
}

func (a *Account) createAccount(from, to, amount string) error {
	tx, err := build.Transaction(
		build.SourceAccount{AddressOrSeed: from},
		network,
		build.AutoSequence{SequenceProvider: client},
		build.CreateAccount(
			build.Destination{AddressOrSeed: to},
			build.NativeAmount{Amount: amount},
		),
		build.MemoText{Value: "via keybase"},
	)
	if err != nil {
		return err
	}

	txe, err := tx.Sign(from)
	if err != nil {
		return err
	}

	txeB64, err := txe.Base64()
	if err != nil {
		return err
	}

	resp, err := client.SubmitTransaction(txeB64)
	if err != nil {
		return err
	}

	fmt.Println("transaction posted in ledger:", resp.Ledger)
	return nil

}

type TransactionsPage struct {
	Links struct {
		Self horizon.Link `json:"self"`
		Next horizon.Link `json:"next"`
		Prev horizon.Link `json:"prev"`
	} `json:"_links"`
	Embedded struct {
		Records []horizon.Transaction `json:"records"`
	} `json:"_embedded"`
}

type PaymentsPage struct {
	Links struct {
		Self horizon.Link `json:"self"`
		Next horizon.Link `json:"next"`
		Prev horizon.Link `json:"prev"`
	} `json:"_links"`
	Embedded struct {
		Records []horizon.Payment `json:"records"`
	} `json:"_embedded"`
}
