package main

import (
	"net/http"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/friendbot/internal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInitFriendbot_createMinionAccounts_success(t *testing.T) {

	randSecretKey := "SDLNA2YUQSFIWVEB57M6D3OOCJHFVCVQZJ33LPA656KJESVRK5DQUZOH"
	botKP, err := keypair.Parse(randSecretKey)
	assert.NoError(t, err)

	botKeypair := botKP.(*keypair.Full)
	botAccountID := botKeypair.Address()
	botAccountMock := horizon.Account{
		AccountID: botAccountID,
		Sequence:  "1",
	}
	botAccount := internal.Account{AccountID: botAccountID, Sequence: 1}

	horizonClientMock := horizonclient.MockClient{}
	horizonClientMock.
		On("AccountDetail", horizonclient.AccountRequest{
			AccountID: botAccountID,
		}).
		Return(botAccountMock, nil)
	horizonClientMock.
		On("SubmitTransactionXDR", mock.Anything).
		Return(horizon.Transaction{}, nil)

	numMinion := 1000
	minionBatchSize := 50
	submitTxRetriesAllowed := 5
	createdMinions, err := createMinionAccounts(botAccount, botKeypair, "Test SDF Network ; September 2015", "10000", "101", numMinion, minionBatchSize, submitTxRetriesAllowed, 1000, &horizonClientMock)
	assert.NoError(t, err)

	assert.Equal(t, 1000, len(createdMinions))
}

func TestInitFriendbot_createMinionAccounts_timeoutError(t *testing.T) {
	randSecretKey := "SDLNA2YUQSFIWVEB57M6D3OOCJHFVCVQZJ33LPA656KJESVRK5DQUZOH"
	botKP, err := keypair.Parse(randSecretKey)
	assert.NoError(t, err)

	botKeypair := botKP.(*keypair.Full)
	botAccountID := botKeypair.Address()
	botAccountMock := horizon.Account{
		AccountID: botAccountID,
		Sequence:  "1",
	}
	botAccount := internal.Account{AccountID: botAccountID, Sequence: 1}

	horizonClientMock := horizonclient.MockClient{}
	horizonClientMock.
		On("AccountDetail", horizonclient.AccountRequest{
			AccountID: botAccountID,
		}).
		Return(botAccountMock, nil)

	// Successful on first 3 calls only, and then a timeout error occurs
	horizonClientMock.
		On("SubmitTransactionXDR", mock.Anything).
		Return(horizon.Transaction{}, nil).Times(3)
	hError := &horizonclient.Error{
		Problem: problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
		},
	}
	horizonClientMock.
		On("SubmitTransactionXDR", mock.Anything).
		Return(horizon.Transaction{}, hError)

	numMinion := 1000
	minionBatchSize := 50
	submitTxRetriesAllowed := 5
	createdMinions, err := createMinionAccounts(botAccount, botKeypair, "Test SDF Network ; September 2015", "10000", "101", numMinion, minionBatchSize, submitTxRetriesAllowed, 1000, &horizonClientMock)
	assert.Equal(t, 150, len(createdMinions))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "after retrying 5 times: submitting create accounts tx:")
}
