---
title: Overview
---

The Go SDK is a set of packages for interacting with most aspects of the Stellar ecosystem. The primary component is the Horizon SDK, which provides convenient access to Horizon services. There are also packages for other Stellar services such as [TOML support](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0001.md) and [federation](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0002.md).

## Horizon SDK

The Horizon SDK is composed of two complementary libraries: `txnbuild` + `horizonclient`.
The `txnbuild` ([source](https://github.com/stellar/go/tree/master/txnbuild), [docs](https://godoc.org/github.com/stellar/go/txnbuild)) package enables the construction, signing and encoding of Stellar [transactions](https://www.stellar.org/developers/guides/concepts/transactions.html) and [operations](https://www.stellar.org/developers/guides/concepts/list-of-operations.html) in Go. The `horizonclient` ([source](https://github.com/stellar/go/tree/master/clients/horizonclient), [docs](https://godoc.org/github.com/stellar/go/clients/horizonclient)) package provides a web client for interfacing with [Horizon](https://www.stellar.org/developers/guides/get-started/) server REST endpoints to retrieve ledger information, and to submit transactions built with `txnbuild`.

## List of major SDK packages

- `horizonclient` ([source](https://github.com/stellar/go/tree/master/clients/horizonclient), [docs](https://godoc.org/github.com/stellar/go/clients/horizonclient)) - programmatic client access to Horizon
- `txnbuild` ([source](https://github.com/stellar/go/tree/master/txnbuild), [docs](https://godoc.org/github.com/stellar/go/txnbuild)) - construction, signing and encoding of Stellar transactions and operations
- `stellartoml` ([source](https://github.com/stellar/go/tree/master/clients/horizonclient), [docs](https://godoc.org/github.com/stellar/go/clients/stellartoml)) - parse [Stellar.toml](../../guides/concepts/stellar-toml.md) files from the internet
- `federation` ([source](https://godoc.org/github.com/stellar/go/clients/federation)) - resolve federation addresses  into stellar account IDs, suitable for use within a transaction

