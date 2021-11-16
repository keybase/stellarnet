package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateXlmPriceRequest(t *testing.T) {
	req, err := createXlmPriceRequest()
	assert.NoError(t, err)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, stelExURL, req.URL.String())
}

func TestParseStellarExpertResponse(t *testing.T) {
	body := "hello"
	gotPrice, gotErr := parseStellarExpertLatestPrice(body)
	assert.EqualError(t, gotErr, "mis-formed response from stellar expert")

	body = "hello,"
	gotPrice, gotErr = parseStellarExpertLatestPrice(body)
	assert.EqualError(t, gotErr, "mis-formed price from stellar expert")

	body = "[[10001,hello]"
	gotPrice, gotErr = parseStellarExpertLatestPrice(body)
	assert.Error(t, gotErr)

	body = "[[100001,5.00],[100002,6.00]]"
	wantPrice := 5.00
	gotPrice, gotErr = parseStellarExpertLatestPrice(body)
	assert.NoError(t, gotErr)
	assert.Equal(t, wantPrice, gotPrice)
}
