// Package examplehorizonclient provides a dummy client for use with the GoDoc examples.
package examplehorizonclient

import (
	hProtocol "github.com/stellar/go/protocols/horizon"
)

// AccountRequest is a simple mock
type AccountRequest struct {
	AccountID string
}

// Client is a simple mock
type Client struct {
}

// DefaultTestNetClient is a simple mock
var DefaultTestNetClient = Client{}

// AccountDetail returns a minimal, static Account object
func (client *Client) AccountDetail(req AccountRequest) (hProtocol.Account, error) {
	return hProtocol.Account{
		AccountID: req.AccountID,
		Sequence:  "3556091187167235",
	}, nil
}
