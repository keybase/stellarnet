package main

import (
	"fmt"

	"github.com/stellar/go/ingest/ledgerbackend"
)

var (
	config = captiveCoreConfig()
)

func captiveCoreConfig() ledgerbackend.CaptiveCoreConfig {
	archiveURLs := []string{
		"https://history.stellar.org/prd/core-testnet/core_testnet_001",
		"https://history.stellar.org/prd/core-testnet/core_testnet_002",
		"https://history.stellar.org/prd/core-testnet/core_testnet_003",
	}
	networkPassphrase := "Test SDF Network ; September 2015"
	captiveCoreToml, err := ledgerbackend.NewCaptiveCoreToml(ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  networkPassphrase,
		HistoryArchiveURLs: archiveURLs,
	})
	panicIf(err)

	return ledgerbackend.CaptiveCoreConfig{
		// Change these based on your environment:
		BinaryPath:         "/usr/local/bin/stellar-core",
		NetworkPassphrase:  networkPassphrase,
		HistoryArchiveURLs: archiveURLs,
		Toml:               captiveCoreToml,
	}
}

func panicIf(err error) {
	if err != nil {
		panic(fmt.Errorf("An error occurred, panicking: %s\n", err))
	}
}
