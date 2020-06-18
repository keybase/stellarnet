# Changelog

All notable changes to this project will be documented in this
file. This project adheres to [Semantic Versioning](http://semver.org/).

## v1.4.0

* Drop support for MuxedAccounts strkeys (spec'ed in [SEP23](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0023.md)).
  SEP23 is still a draft and we don't want to encourage storing strkeys which may not be definite.
* Replace `SequenceProvider` implementation with one which queries the Horizon DB for sequence numbers instead of the Stellar Core DB.
* Use the Horizon DB instead of Horizon's in memory order book graph to query orderbook details for the /order_book endpoint.
* Remove JSON variant of `GET /metrics`, both in the server and client code. It's using Prometheus format by default now.
* Decreased a memory usage of initial state ingestion stage and state verifier ([#2618](https://github.com/stellar/go/pull/2618)).
* Remove `--exp-ingest-in-memory-only` Horizon flag. The in memory order book graph which powers the path finding endpoints is now updated using the Horizon DB instead of directly via ingestion ([#2630](https://github.com/stellar/go/pull/2630)).

## v1.3.0

### Breaking changes

* The type for the following attributes has been changed from `int64` to `string` ([#2555](https://github.com/stellar/go/pull/2555)):
  - Attribute `fee_charged` in [Transaction](https://www.stellar.org/developers/horizon/reference/resources/transaction.html) resource.
  - Attribute `max_fee` in [Transaction](https://www.stellar.org/developers/horizon/reference/resources/transaction.html) resource.

### Changes

* Add `last_modified_time` to account responses. `last_modified_time` is the closing time of the most recent ledger in which the account was modified ([#2528](https://github.com/stellar/go/pull/2528)).
* Balances in the Account resource are now sorted by asset code and asset issuer ([#2516](https://github.com/stellar/go/pull/2516)).
* Ingestion system has its dedicated DB connection pool ([#2560](https://github.com/stellar/go/pull/2560)).
* A new metric has been added to `/metrics` ([#2537](https://github.com/stellar/go/pull/2537) and [#2553](https://github.com/stellar/go/pull/2553)):
  - `ingest.local_latest_ledger`: a gauge with the local latest ledger,
  - `txsub.v0`: a meter counting `v0` transactions in `POST /transaction`,
  - `txsub.v1`: a meter counting `v1` transactions in `POST /transaction`,
  - `txsub.feebump`: a meter counting `feebump` transactions in `POST /transaction`.
* Fix a memory leak in the code responsible for streaming ([#2548](https://github.com/stellar/go/pull/2548), [#2575](https://github.com/stellar/go/pull/2575) and [#2576](https://github.com/stellar/go/pull/2576)).

## v1.2.2

* Fix bug which occurs when ingesting ledgers containing both fee bump and normal transactions.

## v1.2.1

### Database migration notes

This version removes two unused columns that could overflow in catchup complete deployments. If your Horizon database contains entire public network history, you should upgrade to this version as soon as possible and run `horizon db migrate up`.

### Changes

* Remove `id` columns from `history_operation_participants` and `history_transaction_participants` to prevent possible integer overflow [#2532](https://github.com/stellar/go/pull/2532).
## v1.2.0

### Scheduled Breaking Changes

* The type for the following attributes will be changed from `int64` to `string` in 1.3.0:
  - Attribute `fee_charged` in [Transaction](https://www.stellar.org/developers/horizon/reference/resources/transaction.html) resource.
  - Attribute `max_fee` in [Transaction](https://www.stellar.org/developers/horizon/reference/resources/transaction.html) resource.

The changes are required by [CAP-15](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0015.md).

### Changes

* Added support for [CAP-27](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0027.md) and [SEP-23](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0023.md) [#2491](https://github.com/stellar/go/pull/2491).
* The XDR definition of a transaction memo is a string.
However, XDR strings are actually binary blobs with no enforced encoding. 
It is possible to set the memo in a transaction envelope to a binary sequence which is not valid ASCII or unicode. 
Previously, if you wanted to recover the original binary sequence for a transaction memo, you would have to decode the transaction's envelope.
In this release, we have added a `memo_bytes` field to the Horizon transaction response for transactions with `memo_type` equal `text`.
`memo_bytes` stores the base 64 encoding of the memo bytes set in the transaction envelope [#2485](https://github.com/stellar/go/pull/2485).

## v1.1.0

### **IMPORTANT**: Database migration

This version includes a significant database migration which changes the column types of `fee_charged` and `max_fee` in the `history_transactions` table from `integer` to `bigint`. This essential change paves the way for fee bump transactions ([CAP 15](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0015.md)), a major improvement that will be released soon in Stellar Protocol 13.

This migration will run for a long time, especially if you have a Horizon database with full history. For reference, it took around 8 hours and 42 minutes to complete this migration on a AWS db.r4.8xlarge instance with full transaction history.

To execute the migration run `horizon db migrate up` using the Horizon v1.1.0 binary.

**Important Note**: Horizon should not be serving requests or ingesting while the migration is running. For service continuity, if you run a production Horizon deployment it is recommended that you perform the migration on a second instance and then switch over.

### Changes
* DB: Remove unnecessary duplicate indexes: `index_history_transactions_on_id`, `index_history_ledgers_on_id`, `exp_asset_stats_by_code`, and `asset_by_code` ([#2419](https://github.com/stellar/go/pull/2419)).
* DB: Remove asset_stats table which is no longer necessary ([#2419](https://github.com/stellar/go/pull/2419)).
* Validate transaction hash IDs as 64 lowercase hex chars. As such, wrongly-formatted parameters which used to cause 404 (`Not found`) errors will now cause 400 (`Bad request`) HTTP errors ([#2394](https://github.com/stellar/go/pull/2394)).
* Fix ask and bid price levels of `GET /order_book` when encountering non-canonical price values. The `limit` parameter is now respected and levels are coallesced properly. Also, `price_r` is now in canonical form ([#2400](https://github.com/stellar/go/pull/2400)).
* Added missing top-level HAL links to the `GET /` response ([#2407](https://github.com/stellar/go/pull/2407)).
* Full transaction details are now included in the `POST /transactions` response. If you submit a transaction and it succeeds, the response will match the `GET /transactions/{hash}` response ([#2406](https://github.com/stellar/go/pull/2406)).
* The following attributes are now included in the transaction resource:
    * `fee_account` (the account which paid the transaction fee)
    * `fee_bump_transaction` (only present in Protocol 13 fee bump transactions)
    * `inner_transaction` (only present in Protocol 13 fee bump transactions) ([#2406](https://github.com/stellar/go/pull/2406)).
* Add support for [CAP0018](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0018.md): Fine-Grained Control of Authorization (Protocol 13) ([#2423](https://github.com/stellar/go/pull/2423)).
  - Add `is_authorized_to_maintain_liabilities` to `Balance`.
    <pre>
    "balances": [
      {
        "is_authorized": true,
        <b>"is_authorized_to_maintain_liabilities": true,</b>
        "balance": "27.1374422",
        "limit": "922337203685.4775807",
        "buying_liabilities": "0.0000000",
        "selling_liabilities": "0.0000000",
        "last_modified_ledger": 28893780,
        "asset_type": "credit_alphanum4",
        "asset_code": "USD",
        "asset_issuer": "GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK"
      },
      {
        "balance": "1.5000000",
        "buying_liabilities": "0.0000000",
        "selling_liabilities": "0.0000000",
        "asset_type": "native"
      }
    ]
    </pre>
  - Add `authorize_to_maintain_liabilities` to `AllowTrust` operation.
    <pre>
    {
      "id": "124042211741474817",
      "paging_token": "124042211741474817",
      "transaction_successful": true,
      "source_account": "GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK",
      "type": "allow_trust",
      "type_i": 7,
      "created_at": "2020-03-27T03:40:10Z",
      "transaction_hash": "a77d4ee5346d55fb8026cdcdad6e4b5e0c440c96b4627e3727f4ccfa6d199e94",
      "asset_type": "credit_alphanum4",
      "asset_code": "USD",
      "asset_issuer": "GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK",
      "trustee": "GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK",
      "trustor": "GA332TXN6BX2DYKGYB7FW5BWV2JLQKERNX4T7EUJT4MHWOW2TSGC2SPM",
      "authorize": true,
      <b>"authorize_to_maintain_liabilities": true,</b>
    }
    </pre>
  - Add effect `trustline_authorized_to_maintain_liabilities`.
    <pre>
    {
      "id": "0124042211741474817-0000000001",
      "paging_token": "124042211741474817-1",
      "account": "GBSTRUSD7IRX73RQZBL3RQUH6KS3O4NYFY3QCALDLZD77XMZOPWAVTUK",
      <b>"type": "trustline_authorized_to_maintain_liabilities",</b>
      <b>"type_i": 25,</b>
      "created_at": "2020-03-27T03:40:10Z",
      "trustor": "GA332TXN6BX2DYKGYB7FW5BWV2JLQKERNX4T7EUJT4MHWOW2TSGC2SPM",
      "asset_type": "credit_alphanum4",
      "asset_code": "USD"
    }    
    </pre>
* It is no longer possible to use Redis as a mechanism for rate-limiting requests ([#2409](https://github.com/stellar/go/pull/2409)).

* Make `GET /trades` generate an empty response instead of a 404 when no
 trades are found.

## v1.0.1

### Fixed
* Fix `horizon db reap` bug which caused the command to exit without deleting any history table rows ([#2336](https://github.com/stellar/go/pull/2336)).
* The horizon reap system now also deletes rows from `history_trades`. Previously, the reap system only deleted rows from `history_operation_participants`, `history_operations`, `history_transaction_participants`, `history_transactions`, `history_ledgers`, and `history_effects` ([#2336](https://github.com/stellar/go/pull/2336)).
* Fix deadlock when running `horizon db reingest range` ([#2373](https://github.com/stellar/go/pull/2373)).
* Fix signer update effects ([#2375](https://github.com/stellar/go/pull/2375)).
* Fix incorrect error in log when shutting down the system while `verifyState` is running ([#2366](https://github.com/stellar/go/pull/2366)).
* Expose date header to CORS clients ([#2316](https://github.com/stellar/go/pull/2316)).
* Fix inconsistent ledger view in `/accounts/{id}` when streaming ([#2344](https://github.com/stellar/go/pull/2344)).

### Removed
* Dropped support for Go 1.12. ([#2346](https://github.com/stellar/go/pull/2346)).

## v1.0.0

### Before you upgrade

* If you were using the new ingestion in one of the previous versions of Horizon, you must first remove `ENABLE_EXPERIMENTAL_INGESTION` feature flag and restart all Horizon instances before deploying a new version.
* The init stage (state ingestion) for the public Stellar network requires around 1.5GB of RAM. This memory is released after the state ingestion. State ingestion is performed only once. Restarting the server will not trigger it unless Horizon has been upgraded to a newer version (with an updated ingestion pipeline). It's worth noting that the required memory will become smaller and smaller as more of the buckets in the history archive become [CAP-20](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0020.md) compatible. Some endpoints are **not available** during state ingestion.
* The CPU footprint of the new ingestion is modest. We were able to successfully run ingestion on an [AWS `c5.xlarge`](https://aws.amazon.com/ec2/instance-types/c5/) instance. The init stage takes a few minutes on `c5.xlarge`. `c5.xlarge` is the equivalent of 4 vCPUs and 8GB of RAM. The definition of vCPU for the c5 large family in AWS is the following:
> The 2nd generation Intel Xeon Scalable Processors (Cascade Lake) or 1st generation Intel Xeon Platinum 8000 series (Skylake-SP) processor with a sustained all core Turbo frequency of up to 3.4GHz, and single core turbo frequency of up to 3.5 GHz.

* The state data requires an additional 6GB DB disk space for the public Stellar network (as of February 2020). The disk usage will increase when the number of Stellar ledger entries increases.
  * `accounts_signers` table: 2340 MB
  * `trust_lines` table: 2052 MB
  * `accounts` table: 1545 MB
  * `offers` table: 61 MB
  * `accounts_data` table: 15 MB
  * `exp_asset_stats` table: less than 1 MB
* A new environment variable (or command line flag) needs to be set so that Horizon can ingest state from the history archives:
   * `HISTORY_ARCHIVE_URLS="archive1,archive2,archive3"` (if you don't have your own pubnet history archive, you can use one of SDF's archives, for example `https://history.stellar.org/prd/core-live/core_live_001`)
* Horizon serves the endpoints `/paths` and `/order_book` from an in-memory graph, which is only available on ingesting instances. If some of the instances in your cluster are not configured to ingest, you can configure your proxy server to route those endpoints to the ingesting instances. This is beyond the scope of this document - consult the relevant documentation for your proxy server. A better solution for this will be released in the next Horizon version.

### New Ingestion System

The most substantial element of this release is a full rewrite of Horizon's ledger ingestion engine, which enables some key features:

* A set of important new endpoints (see below). Some of these were impossible under the previous ingestion architecture.
* An in-memory order book graph for rapid querying.
* The ability to run parallel ingestion over multiple Horizon hosts, improving service availability for production deployments.

The new engine resolves multiple issues that were present in the old system. For example:

* Horizon's coupling to Stellar-Core's database is dramatically reduced.
* Data inconsistency due to lag between endpoints is eliminated.
* Slow endpoints (path-finding for example) are now speedy.

Finally, the rearchitecting makes new reliability features possible. An example is the new internal state verifier, which guarantees consistency between the local Horizon state and the public history archives.

The [admin guide](https://github.com/stellar/go/blob/release-horizon-v0.25.0/services/horizon/internal/docs/admin.md) contains all the information needed to operate the new ingestion system.

### Added

- Add [/accounts](https://www.stellar.org/developers/horizon/reference/endpoints/accounts.html) endpoint, which allows filtering accounts that have a given signer or a trustline to an asset.
- Add [/offers](https://www.stellar.org/developers/horizon/reference/endpoints/offers.html) endpoint, which lists all offers on the network and allows filtering by seller account or by selling or buying asset.
- Add [/paths/strict-send](https://www.stellar.org/developers/horizon/reference/endpoints/path-finding-strict-send.html) endpoint, which enables discovery of optimal "strict send" paths between assets.
- Add [/paths/strict-receive](https://www.stellar.org/developers/horizon/reference/endpoints/path-finding-strict-receive.html) endpoint, which enables discovery of optimal "strict receive" paths between assets.
- Add the fields `max_fee` and `fee_charged` to [/fee_stats](https://www.stellar.org/developers/horizon/reference/endpoints/fee-stats.html).

### Breaking changes

### Changed

- Change multiple operation types to their canonical names for [operation resources](https://www.stellar.org/developers/horizon/reference/resources/operation.html) ([#2134](https://github.com/stellar/go/pull/2134)).
- Change the type of the following fields from `number` to `string`:

    - Attribute `offer_id` in [manage buy offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-buy-offer) and [manage sell offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-sell-offer) operations.
    - Attribute `offer_id` in `Trade` [effect](https://www.stellar.org/developers/horizon/reference/resources/effect.html#trading-effects).
    - Attribute `id` in [Offer](https://www.stellar.org/developers/horizon/reference/resources/offer.html) resource.
    - Attribute `timestamp` and `trade_count` in [Trade Aggregation](https://www.stellar.org/developers/horizon/reference/resources/trade_aggregation.html) resource.

    See [#1609](https://github.com/stellar/go/issues/1609), [#1909](https://github.com/stellar/go/pull/1909) and [#1912](https://github.com/stellar/go/issues/1912) for more details.

### Removed

- `/metrics` endpoint is no longer part of the public API. It is now served on `ADMIN_PORT/metrics`. `ADMIN_PORT` can be set using env variable or `--admin-port` CLI param.
- Remove the following fields from [/fee_stats](https://www.stellar.org/developers/horizon/reference/endpoints/fee-stats.html):

    - `min_accepted_fee`
    - `mode_accepted_fee`
    - `p10_accepted_fee`
    - `p20_accepted_fee`
    - `p30_accepted_fee`
    - `p40_accepted_fee`
    - `p50_accepted_fee`
    - `p60_accepted_fee`
    - `p70_accepted_fee`
    - `p80_accepted_fee`
    - `p90_accepted_fee`
    - `p95_accepted_fee`
    - `p99_accepted_fee`

- Remove `fee_paid` field from [Transaction resource](https://www.stellar.org/developers/horizon/reference/resources/transaction.html) (Use `fee_charged` and `max_fee` fields instead - see [#1372](https://github.com/stellar/go/issues/1372)).

## v0.24.1

* Add cache to improve performance of experimental ingestion system (#[2004](https://github.com/stellar/go/pull/2004)).
* Fix experimental ingestion bug where ledger changes were not applied in the correct order (#[2050](https://github.com/stellar/go/pull/2050)).
* Fix experimental ingestion bug where unique constraint errors are incurred when the ingestion system has to reingest state from history archive checkpoints (#[2055](https://github.com/stellar/go/pull/2055)).
* Fix experimental ingestion bug where a race condition during shutdown leads to a crash (#[2058](https://github.com/stellar/go/pull/2058)).

## v0.24.0

* Add `fee_charged` and `max_fee` objects to `/fee_stats` endpoint ([#1964](https://github.com/stellar/go/pull/1964)).
* Experimental ledger header ingestion processor ([#1949](https://github.com/stellar/go/pull/1949)).
* Improved performance of asset stats processor ([#1987](https://github.com/stellar/go/pull/1987)).
* Provide mechanism for retrying XDR stream errors ([#1899](https://github.com/stellar/go/pull/1899)).
* Emit error level log after 3 failed attempts to validate state ([#1918](https://github.com/stellar/go/pull/1918)).
* Fixed out of bounds error in ledger backend reader ([#1914](https://github.com/stellar/go/pull/1914)).
* Fixed out of bounds error in URL params handler ([#1973](https://github.com/stellar/go/pull/1973)).
* Rename `OperationFeeStats` to `FeeStats` ([#1952](https://github.com/stellar/go/pull/1952)).
* All DB queries are now cancelled when request is cancelled/timeout. ([#1950](https://github.com/stellar/go/pull/1950)).
* Fixed multiple issues connected to graceful shutdown of Horizon.

### Scheduled Breaking Changes

* All `*_accepted_fee` fields in `/fee_stats` endpoint are deprecated. Fields will be removed in Horizon 0.25.0.

Previously scheduled breaking changes reminders:

* The following operation type names have been deprecated: `path_payment`, `manage_offer` and `create_passive_offer`. The names will be changed to: `path_payment_strict_receive`, `manage_sell_offer` and `create_passive_sell_offer` in 0.25.0. This has been previously scheduled for 0.22.0 release.
* `fee_paid` field on Transaction resource has been deprecated and will be removed in 0.25.0 (previously scheduled for 0.22.0). Please use new fields added in 0.18.0: `max_fee` that defines the maximum fee the source account is willing to pay and `fee_charged` that defines the fee that was actually paid for a transaction. See [CAP-0005](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0005.md) for more information.
* The type for the following attributes will be changed from `int64` to `string` in 0.25.0 (previously scheduled for 0.22.0):
  - Attribute `offer_id` in [manage buy offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-buy-offer) and [manage sell offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-sell-offer) operations.
  - Attribute `offer_id` in `Trade` effect.
  - Attribute `id` in [Offer](https://www.stellar.org/developers/horizon/reference/resources/offer.html) resource.
  - Attribute `timestamp` and `trade_count` in [Trade Aggregation](https://www.stellar.org/developers/horizon/reference/resources/trade_aggregation.html) resource.

Check [Beta Testing New Ingestion System](https://github.com/stellar/go/blob/master/services/horizon/internal/expingest/BETA_TESTING.md) if you want to test the new ingestion system.

## v0.23.1

* Add `ReadTimeout` to Horizon HTTP server configuration to fix potential DoS vector.

## v0.23.0

* New features in experimental ingestion (to enable: set `--enable-experimental-ingestion` CLI param or `ENABLE_EXPERIMENTAL_INGESTION=true` env variable):
  * All state-related endpoints (i.e. ledger entries) are now served from Horizon DB (except `/account/{account_id}`)

  * `/order_book` offers data is served from in-memory store ([#1761](https://github.com/stellar/go/pull/1761))

  * Add `Latest-Ledger` header with the sequence number of the most recent ledger processed by the experimental ingestion system. Endpoints built on the experimental ingestion system will always respond with data which is consistent with the ledger in `Latest-Ledger` ([#1830](https://github.com/stellar/go/pull/1830))

  * Add experimental support for filtering accounts who are trustees to an asset via `/accounts`. Example:\
  `/accounts?asset=COP:GC2GFGZ5CZCFCDJSQF3YYEAYBOS3ZREXJSPU7LUJ7JU3LP3BQNHY7YKS`\
  returns all accounts who have a trustline to the asset `COP` issued by account `GC2GFG...` ([#1835](https://github.com/stellar/go/pull/1835))

  * Experimental "Accounts For Signers" end-point now returns a full account resource ([#1876](https://github.com/stellar/go/issues/1875))
* Prevent "`multiple response.WriteHeader calls`" errors when streaming ([#1870](https://github.com/stellar/go/issues/1870))
* Fix an interpolation bug in `/fee_stats` ([#1857](https://github.com/stellar/go/pull/1857))
* Fix a bug in `/paths/strict-send` where occasionally bad paths were returned ([#1863](https://github.com/stellar/go/pull/1863))

## v0.22.2

* Fixes a bug in accounts for signer ingestion processor.

## v0.22.1

* Fixes a bug in path payment ingestion code.

## v0.22.0

* Adds support for Stellar Protocol v12.

### Scheduled Breaking Changes

* The following operation type names have been deprecated: `path_payment`, `manage_offer` and `create_passive_offer`. The names will be changed to: `path_payment_strict_receive`, `manage_sell_offer` and `create_passive_sell_offer` in 0.25.0. This has been previously scheduled for 0.22.0 release.
* `fee_paid` field on Transaction resource has been deprecated and will be removed in 0.23.0 (previously scheduled for 0.22.0). Please use new fields added in 0.18.0: `max_fee` that defines the maximum fee the source account is willing to pay and `fee_charged` that defines the fee that was actually paid for a transaction. See [CAP-0005](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0005.md) for more information.
* The type for the following attributes will be changed from `int64` to `string` in 0.23.0 (previously scheduled for 0.22.0):
  - Attribute `offer_id` in [manage buy offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-buy-offer) and [manage sell offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-sell-offer) operations.
  - Attribute `offer_id` in `Trade` effect.
  - Attribute `id` in [Offer](https://www.stellar.org/developers/horizon/reference/resources/offer.html) resource.
  - Attribute `timestamp` and `trade_count` in [Trade Aggregation](https://www.stellar.org/developers/horizon/reference/resources/trade_aggregation.html) resource.

## v0.21.1

* Fixes a bug in initial schema migration file.

## v0.21.0

### Database migration notes

This version adds a new index on a table used by experimental ingestion system. If it has not been enabled, migration will be instant. If you migrate from a previous version with experimental ingestion system enabled database migration can take a couple minutes.

### Changes

* `/paths/strict-send` can now accept a `destination_account` parameter. If `destination_account` is provided then the endpoint will return all payment paths which terminate with an asset held by `destination_account`. Note that the endpoint will accept `destination_account` or `destination_assets` but not both. `destination_assets` is a comma separated list of assets encoded as `native` or `code:issuer`.
* `/paths/strict-receive` can now accept a `source_assets` parameter instead of `source_account` parameter. If `source_assets` is provided the endpoint will return all payment paths originating from an asset in `source_assets`. Note that the endpoint will accept `source_account` or `source_assets` but not both. `source_assets` is a comma separated list of assets encoded as `native` or `code:issuer`.
* Add experimental support for `/offers`. To enable it, set `--enable-experimental-ingestion` CLI param or `ENABLE_EXPERIMENTAL_INGESTION=true` env variable.
* When experimental ingestion is enabled a state verification routine is started every 64 ledgers to ensure a local state is the same as in history buckets. This can be disabled by setting `--ingest-disable-state-verification` CLI param or `INGEST-DISABLE-STATE-VERIFICATION` env variable.
* Add flag to apply pending migrations before running horizon. If there are pending migrations, previously you needed to run `horizon db migrate up` before running `horizon`. Those two steps can be combined into one with the `--apply-migrations` flag (`APPLY_MIGRATIONS` env variable).
* Improved the speed of state ingestion in experimental ingestion system.
* Fixed a bug in "Signers for Account" (experimental) transaction meta ingesting code.
* Fixed performance issue in Effects related endpoints.
* Fixed DoS vector in Go HTTP/2 implementation.
* Dropped support for Go 1.10, 1.11.

Check [Beta Testing New Ingestion System](https://github.com/stellar/go/blob/master/services/horizon/internal/expingest/BETA_TESTING.md) if you want to test new ingestion system.

## v0.20.1

* Add `--ingest-state-reader-temp-set` flag (`INGEST_STATE_READER_TEMP_SET` env variable) which defines the storage type used for temporary objects during state ingestion in the new ingestion system. The possible options are: `memory` (requires ~1.5GB RAM, fast) and `postgres` (stores data in temporary table in Postgres, less RAM but slower).

Check [Beta Testing New Ingestion System](https://github.com/stellar/go/blob/master/services/horizon/internal/expingest/BETA_TESTING.md) if you want to test new ingestion system.

## v0.20.0

If you want to use experimental ingestion skip this version and use v0.20.1 instead. v0.20.0 has a performance issue.

### Changes

* Experimental ingestion system is now run concurrently on all Horizon servers (with feature flag set - see below). This improves ingestion availability.
* Add experimental path finding endpoints which use an in memory order book for improved performance. To enable the endpoints set `--enable-experimental-ingestion` CLI param or `ENABLE_EXPERIMENTAL_INGESTION=true` env variable. Note that the `enable-experimental-ingestion` flag enables both the new path finding endpoints and the accounts for signer endpoint. There are two path finding endpoints. `/paths/strict-send` returns payment paths where both the source and destination assets are fixed. This endpoint is able to answer questions like: "Get me the most EUR possible for my 10 USD." `/paths/strict-receive` is the other endpoint which is an alias to the existing `/paths` endpoint.
* `--enable-accounts-for-signer` CLI param or `ENABLE_ACCOUNTS_FOR_SIGNER=true` env variable are merged with `--enable-experimental-ingestion` CLI param or `ENABLE_EXPERIMENTAL_INGESTION=true` env variable.
* Add experimental get offers by id endpoint`/offers/{id}` which uses the new ingestion system to fill up the offers table. To enable it, set `--enable-experimental-ingestion` CLI param or `ENABLE_EXPERIMENTAL_INGESTION=true` env variable.

Check [Beta Testing New Ingestion System](https://github.com/stellar/go/blob/master/services/horizon/internal/expingest/BETA_TESTING.md) if you want to test new ingestion system.

### Scheduled Breaking Changes

* `fee_paid` field on Transaction resource has been deprecated and will be removed in 0.22.0. Please use new fields added in 0.18.0: `max_fee` that defines the maximum fee the source account is willing to pay and `fee_charged` that defines the fee that was actually paid for a transaction. See [CAP-0005](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0005.md) for more information. This change has been previously scheduled for 0.19.0 release.
* The following operation type names have been deprecated: `manage_offer` and `create_passive_offer`. The names will be changed to: `manage_sell_offer` and `create_passive_offer` in 0.22.0. This has been previously scheduled for 0.19.0 release.
* The type for the following attributes will be changed from `int64` to `string` in 0.22.0:
  - Attribute `offer_id` in [manage buy offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-buy-offer) and [manage sell offer](https://www.stellar.org/developers/horizon/reference/resources/operation.html#manage-sell-offer) operations.
  - Attribute `offer_id` in `Trade` effect.
  - Attribute `id` in [Offer](https://www.stellar.org/developers/horizon/reference/resources/offer.html) resource.
  - Attribute `timestamp` and `trade_count` in [Trade Aggregation](https://www.stellar.org/developers/horizon/reference/resources/trade_aggregation.html) resource.

If you are an SDK maintainer, update your code to prepare for this change.

## v0.19.0

* Add `join` parameter to operations and payments endpoints. Currently, the only valid value for the parameter is `transactions`. If `join=transactions` is included in a request then the response will include a `transaction` field for each operation in the response.
* Add experimental "Accounts For Signers" endpoint. To enable it set `--enable-accounts-for-signer` CLI param or `ENABLE_ACCOUNTS_FOR_SIGNER=true` env variable. Additionally new feature requires links to history archive: CLI: `--history-archive-urls="archive1,archive2,archive3"`, env variable: `HISTORY_ARCHIVE_URLS="archive1,archive2,archive3"`. This will expose `/accounts` endpoint. This requires around 4GB of RAM for initial state ingestion.

Check [Beta Testing New Ingestion System](https://github.com/stellar/go/blob/master/services/horizon/internal/expingest/BETA_TESTING.md) if you want to test new ingestion system.

## v0.18.1

* Fixed `/fee_stats` to correctly calculate ledger capacity in protocol v11.
* Fixed `horizon db clean` command to truncate all tables.

## v0.18.0

### Breaking changes

* Horizon requires Postgres 9.5+.
* Removed `paging_token` field from `/accounts/{id}` endpoint.
* Removed `/operation_fee_stats` endpoint. Please use `/fee_stats`.

### Deprecations

* `fee_paid` field on Transaction resource has been deprecated and will be removed in 0.19.0. Two new fields have been added: `max_fee` that defines the maximum fee the source account is willing to pay and `fee_charged` that defines the fee that was actually paid for a transaction. See [CAP-0005](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0005.md) for more information.
* The following operation type names have been deprecated: `manage_offer` and `create_passive_offer`. The names will be changed to: `manage_sell_offer` and `create_passive_offer` in 0.19.0.

### Changes

* The following new config parameters were added. When old `max-db-connections` config parameter is set, it has a priority over the the new params. Run `horizon help` for more information.
  * `horizon-db-max-open-connections`,
  * `horizon-db-max-idle-connections`,
  * `core-db-max-open-connections`,
  * `core-db-max-idle-connections`.
* Fixed `fee_paid` value in Transaction resource (#1358).
* Fix "int64: value out of range" errors in trade aggregations (#1319).
* Improved `horizon db reingest range` command.

## v0.17.6 - 2019-04-29

* Fixed a bug in `/order_book` when sum of amounts at a single price level exceeds `int64_max` (#1037).
* Fixed a bug generating `ERROR` level log entries for bad requests (#1186).

## v0.17.5 - 2019-04-24

* Support for stellar-core [v11.0.0](https://github.com/stellar/stellar-core/releases/tag/v11.0.0).
* Display trustline authorization state in the balances list.
* Improved actions code.
* Improved `horizon db reingest` command handling code.
* Tracking app name and version that connects to Horizon (`X-App-Name`, `X-App-Version`).

## v0.17.4 - 2019-03-14

* Support for Stellar-Core 10.3.0 (new database schema v9).
* Fix a bug in `horizon db reingest` command (no log output).
* Multiple code improvements.

## v0.17.3 - 2019-03-01

* Fix a bug in `txsub` package that caused returning invalid status when resubmitting old transactions (#969).

## v0.17.2 - 2019-02-28

* Critical fix bug

## v0.17.1 - 2019-02-28

### Changes

* Fixes high severity error in ingestion system.
* Account detail endpoint (`/accounts/{id}`) includes `last_modified_ledger` field for account and for each non-native asset balance.

## v0.17.0 - 2019-02-26

### Upgrade notes

This release introduces ingestion of failed transactions. This feature is turned off by default. To turn it on set environment variable: `INGEST_FAILED_TRANSACTIONS=true` or CLI param: `--ingest-failed-transactions=true`. Please note that ingesting failed transactions can double DB space requirements (especially important for full history deployments).

### Database migration notes

Previous versions work fine with new schema so you can migrate (`horizon db migrate up` using new binary) database without stopping the Horizon process. To reingest ledgers run `horizon db reingest` using Horizon 0.17.0 binary. You can take advantage of the new `horizon db reingest range` for parallel reingestion.

### Deprecations

* `/operation_fee_stats` is deprecated in favour of `/fee_stats`. Will be removed in v0.18.0.

### Breaking changes

* Fields removed in this version:
  * Root > `protocol_version`, use `current_protocol_version` and `core_supported_protocol_version`.
  * Ledger > `transaction_count`, use `successful_transaction_count` and `failed_transaction_count`.
  * Signer > `public_key`, use `key`.
* This Horizon version no longer supports Core <10.0.0. Horizon can still ingest version <10 ledgers.
* Error event name during streaming changed to `error` to follow W3C specification.

### Changes

* Added ingestion of failed transactions (see Upgrade notes). Use `include_failed=true` GET parameter to display failed transactions and operations in collection endpoints.
* `/fee_stats` endpoint has been extended with fee percentiles and ledger capacity usage. Both are useful in transaction fee estimations.
* Fixed a bug causing slice bounds out of range at `/account/{id}/offers` endpoint during streaming.
* Added `horizon db reingest range X Y` that reingests ledgers between X and Y sequence number (closed intervals).
* Many code improvements.

## v0.16.0 - 2019-02-04

### Upgrade notes

* Ledger > Admins need to reingest old ledgers because we introduced `successful_transaction_count` and `failed_transaction_count`.

### Database migration notes

Previous versions work fine with Horizon 0.16.0 schema so you can migrate (`horizon db migrate up`) database without stopping the Horizon process. To reingest ledgers run `horizon db reingest` using Horizon 0.16.0 binary.

### Deprecations

* Root > `protocol_version` will be removed in v0.17.0. It is replaced by `current_protocol_version` and `core_supported_protocol_version`.
* Ledger > `transaction_count` will be removed in v0.17.0.
* Signer > `public_key` will be removed in v0.17.0.

### Changes

* Improved `horizon db migrate` script. It will now either success or show a detailed message regarding why it failed.
* Fixed effects ingestion of circular payments.
* Improved account query performances for payments and operations.
* Added `successful_transaction_count` and `failed_transaction_count` to `ledger` resource.
* Fixed the wrong protocol version displayed in `root` resource by adding `current_protocol_version` and `core_supported_protocol_version`.
* Improved streaming for single objects. It won't send an event back if the current event is the same as the last event sent.
* Fixed ingesting effects of empty trades. Empty trades will be ignored during ingestion.

## v0.15.4 - 2019-01-17

* Fixed multiple issues in transaction submission subsystem.
* Support for client fingerprint headers.
* Fixed parameter checking in `horizon db backfill` command.

## v0.15.3 - 2019-01-07

* Fixed a bug in Horizon DB reaping code.
* Fixed query checking code that generated `ERROR`-level log entries for invalid input.

## v0.15.2 - 2018-12-13

* Added `horizon db init-asset-stats` command to initialize `asset_stats` table. This command should be run once before starting ingestion if asset stats are enabled (`ENABLE_ASSET_STATS=true`).
* Fixed `asset_stats` table to support longer `home_domain`s.
* Fixed slow trades DB query.

## v0.15.1 - 2018-11-09

* Fixed memory leak in SSE stream code.

## v0.15.0 - 2018-11-06

DB migrations add a new fields and indexes on `history_trades` table. This is a very large table in `CATCHUP_COMPLETE` deployments so migration may take a long time (depending on your DB hardware). Please test the migrations execution time on the copy of your production DB first.

This release contains several bug fixes and improvements:

* New `/operation_fee_stats` endpoint includes fee stats for the last 5 ledgers.
* ["Trades"](https://www.stellar.org/developers/horizon/reference/endpoints/trades.html) endpoint can now be streamed.
* In ["Trade Aggregations"](https://www.stellar.org/developers/horizon/reference/endpoints/trade_aggregations.html) endpoint, `offset` parameter has been added.
* Path finding bugs have been fixed and the algorithm has been improved. Check [#719](https://github.com/stellar/go/pull/719) for more information.
* Connections (including streams) are closed after timeout defined using `--connection-timeout` CLI param or `CONNECTION_TIMEOUT` environment variable. If Horizon is behind a load balancer with idle timeout set, it is recommended to set this to a value equal a few seconds less than idle timeout so streams can be properly closed by Horizon.
* Streams have been improved to check for updates every `--sse-update-frequency` CLI param or `SSE_UPDATE_FREQUENCY` environment variable seconds. If a new ledger has been closed in this period, new events will be sent to a stream. Previously streams checked for new events every 1 second, even when there were no new ledgers.
* Rate limiting algorithm has been changed to [GCRA](https://brandur.org/rate-limiting#gcra).
* Rate limiting in streams has been changed to be more fair. Now 1 *credit* has to be *paid* every time there's a new ledger instead of per request.
* Rate limiting can be disabled completely by setting `--per-hour-rate-limit=0` CLI param or `PER_HOUR_RATE_LIMIT=0` environment variable.
* Account flags now display `auth_immutable` value.
* Logs can be sent to a file. Destination file can be set using an environment variable (`LOG_FILE={file}`) or CLI parameter (`--log-file={file}`).

### Breaking changes

* Assets stats are disabled by default. This can be changed using an environment variable (`ENABLE_ASSET_STATS=true`) or CLI parameter (`--enable-asset-stats=true`). Please note that it has a negative impact on a DB and ingestion time.
* In ["Offers for Account"](https://www.stellar.org/developers/horizon/reference/endpoints/offers-for-account.html), `last_modified_time` field  endpoint can be `null` when ledger data is not available (has not been ingested yet).
* ["Trades for Offer"](https://www.stellar.org/developers/horizon/reference/endpoints/trades-for-offer.html) endpoint will query for trades that match the given offer on either side of trades, rather than just the "sell" offer. Offer IDs are now [synthetic](https://www.stellar.org/developers/horizon/reference/resources/trade.html#synthetic-offer-ids). You have to reingest history to update offer IDs.

### Other bug fixes

* `horizon db backfill` command has been fixed.
* Fixed `remoteAddrIP` function to support IPv6.
* Fixed `route` field in the logs when the request is rate limited.

## v0.14.2 - 2018-09-27

### Bug fixes

* Fixed and improved `txsub` package (#695). This should resolve many issues connected to `Timeout` responses.
* Improve stream error reporting (#680).
* Checking `ingest.Cursor` errors in `Session` (#679).
* Added account ID validation in `/account/{id}` endpoints (#684).

## v0.14.1 - 2018-09-19

This release contains several bug fixes:

* Assets stats can cause high CPU usage on stellar-core DB. If this slows down the database it's now possible to turn off this feature by setting `DISABLE_ASSET_STATS` feature flag. This can be set as environment variable (`DISABLE_ASSET_STATS=true`) or CLI parameter (`--disable-asset-stats=true`).
* Sometimes `/accounts/{id}/offers` returns `500 Internal Server Error` response when ledger data is not available yet (for new ledgers) or no longer available (`CATCHUP_RECENT` deployments). It's possible to set `ALLOW_EMPTY_LEDGER_DATA_RESPONSES` feature flag as environment variable (`ALLOW_EMPTY_LEDGER_DATA_RESPONSES=true`) or CLI parameter (`--allow-empty-ledger-data-responses=true`). With the flag set to `true` "Offers for Account" endpoint will return `null` in `last_modified_time` field when ledger data is not available, instead of `500 Internal Server Error` error.

### Bug fixes

* Feature flag to disable asset stats (#668).
* Feature flag to allow null ledger data in responses (#672).
* Fix empty memo field in JSON when memo_type is text (#635).
* Improved logging: some bad requests no longer generate `ERROR` level log entries (#654).
* `/friendbot` endpoint is available only when `FriendbotURL` is set in the config.

## v0.14.0 - 2018-09-06

### Breaking changes

* Offer resource `last_modified` field removed (see Bug Fixes section).
* Trade aggregations endpoint accepts only specific time ranges now (1/5/15 minutes, 1 hour, 1 day, 1 week).
* Horizon sends `Cache-Control: no-cache, no-store, max-age=0` HTTP header for all responses.

### Deprecations

* Account > Signers collection `public_key` field is deprecated, replaced by `key`.

### Changes

* Protocol V10 features:
  * New `bump_sequence` operation (as in [CAP-0001](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0001.md)).
    * New [`bump_sequence`](https://www.stellar.org/developers/horizon/reference/resources/operation.html#bump-sequence) operation.
    * New `sequence_bumped` effect.
    * Please check [CAP-0001](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0001.md) for new error codes for transaction submission.
  * Offer liabilities (as in [CAP-0003](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0003.md)):
    * `/accounts/{id}` resources contain new fields: `buying_liabilities` and `selling_liabilities` for each entry in `balances`.
    * Please check [CAP-0003](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0003.md) for new error codes for transaction submission.
* Added `source_amount` field to `path_payment` operations.
* Added `account_credited` and `account_debited` effects for `path_payment` operations.
* Friendbot link in Root endpoint is empty if not set in configuration.
* Improved `ingest` package logging.
* Improved HTTP logging (`forwarded_ip`, `route` fields, `duration` is always in seconds).
* `LOGGLY_HOST` env variable has been replaced with `LOGGLY_TAG` and is adding a tag to every log event.
* Dropped support for Go 1.8.

### Bug fixes

* New fields in Offer resource: `last_modified_ledger` and `last_modified_time`, replace buggy `last_modified` (#478).
* Fixed pagination in Trades for account endpoint (#486).
* Fixed a synchronization issue in `ingest` package (#603).
* Fixed Order Book resource links in Root endpoint.
* Fixed streaming in Offers for Account endpoint.

## v0.13.3 - 2018-08-23

### Bug fixes

* Fixed large amounts rendering in `/assets`.

## v0.13.2 - 2018-08-13

### Bug fixes

* Fixed a bug in `amount` and `price` packages triggering long calculations.

## v0.13.1 - 2018-07-26

### Bug fixes

* Fixed a conversion bug when `timebounds.max_time` is set to `INT64_MAX`.

## v0.13.0 - 2018-06-06

### Breaking changes

- `amount` field in `/assets` is now a String (to support Stellar amounts larger than `int64`).

### Changes

- Effect resource contains a new `created_at` field.
- Horizon responses are compressed.
- Ingestion errors have been improved.
- `horizon rebase` command was improved.

### Bug fixes

- Horizon now returns `400 Bad Request` for negative `cursor` values.

**Upgrade notes**

DB migrations add a new indexes on `history_trades`. This is very large table so migration may take a long time (depending on your DB hardware). Please test the migrations execution time on the copy of your production DB first.

## v0.12.3 - 2017-03-20

### Bug fixes

- Fix a service stutter caused by excessive `info` commands being issued from the root endpoint.


## v0.12.2 - 2017-03-14

This release is a bug fix release for v0.12.1 and v0.12.2.  *Please see the upgrade notes below if you did not already migrate your db for v0.12.0*

### Changes

- Remove strict validation on the `resolution` parameter for trade aggregations endpoint.  We will add this feature back in to the next major release.


## v0.12.1 - 2017-03-13

This release is a bug fix release for v0.12.0.  *Please see the upgrade notes below if you did not already migrate your db for v0.12.0*

### Bug fixes

- Fixed an issue caused by un-migrated trade rows. (https://github.com/stellar/go/issues/357)
- Command line flags are now useable for subcommands of horizon.


## v0.12.0 - 2017-03-08

Big release this time for horizon:  We've made a number of breaking changes since v0.11.0 and have revised both our database schema as well as our data ingestion system.  We recommend that you take a backup of your horizon database prior to upgrading, just in case.

### Upgrade Notes

Since this release changes both the schema and the data ingestion system, we recommend the following upgrade path to minimize downtime:

1. Upgrade horizon binaries, but do not restart the service
2. Run `horizon db migrate up` to migrate the db schema
3. Run `horizon db reingest` in a background session to begin the data reingestion process
4. Restart horizon

### Added

- Operation and payment resources were changed to add `transaction_hash` and `created_at` properties.
- The ledger resource was changed to add a `header_xdr` property.  Existing horizon installations should re-ingest all ledgers to populate the history database tables with the data.  In future versions of horizon we will disallow null values in this column.  Going forward, this change reduces the coupling of horizon to stellar-core, ensuring that horizon can re-import history even when the data is no longer stored within stellar-core's database.
- All Assets endpoint (`/assets`) that returns a list of all the assets in the system along with some stats per asset. The filters allow you to narrow down to any specific asset of interest.
- Trade Aggregations endpoint (`/trade_aggregations`) allow for efficient gathering of historical trade data. This is done by dividing a given time range into segments and aggregate statistics, for a given asset pair (`base`, `counter`) over each of these segments.

### Bug fixes

- Ingestion performance and stability has been improved.
- Changes to an account's inflation destination no longer produce erroneous "signer_updated" effects. (https://github.com/stellar/horizon/issues/390)


### Changed

- BREAKING CHANGE: The `base_fee` property of the ledger resource has been renamed to `base_fee_in_stroops`
- BREAKING CHANGE: The `base_reserve` property of the ledger resource has been renamed to `base_reserve_in_stroops` and is now expressed in stroops (rather than lumens) and as a JSON number.
- BREAKING CHANGE: The "Orderbook Trades" (`/orderbook/trades`) endpoint has been removed and replaced by the "All Trades" (`/trades`) endpoint.
- BREAKING CHANGE: The Trade resource has been modified to generalize assets as (`base`, `counter`) pairs, rather than the previous (`sold`,`bought`) pairs.
- Full reingestion (i.e. running `horizon db reingest`) now runs in reverse chronological order.

### Removed

- BREAKING CHANGE: Friendbot has been extracted to an external microservice.


## [v0.11.0] - 2017-08-15

### Bug fixes

- The ingestion system can now properly import envelopes that contain signatures that are zero-length strings.
- BREAKING CHANGE: specifying a `limit` of `0` now triggers an error instead of interpreting the value to mean "use the default limit".
- Requests that ask for more records than the maximum page size now trigger a bad request error, instead of an internal server error.
- Upstream bug fixes to xdr decoding from `github.com/stellar/go`.

### Changed

- BREAKING CHANGE: The payments endpoint now includes `account_merge` operations in the response.
- "Finished Request" log lines now include additional fields: `streaming`, `path`, `ip`, and `host`.
- Responses now include a `Content-Disposition: inline` header.


## [v0.10.1] - 2017-03-29

### Fixed
- Ingestion was fixed to protect against text memos that contain null bytes.  While memos with null bytes are legal in stellar-core, PostgreSQL does not support such values in string columns.  Horizon now strips those null bytes to fix the issue.

## [v0.10.0] - 2017-03-20

This is a fix release for v0.9.0 and v0.9.1


### Added
- Added `horizon db clear` helper command to clear previously ingested history.

### Fixed

- Embedded sql files for the database schema have been fixed agsain to be compatible with postgres 9.5. The configuration setting `row_security` has been removed from the dumped files.

## [v0.9.1] - 2017-03-20

### Fixed

- Embedded sql files for the database schema have been fixed to be compatible with postgres 9.5. The configuration setting `idle_in_transaction_session_timeout` has been removed from the dumped files.

## [v0.9.0] - 2017-03-20

This release was retracted due to a bug discovered after release.

### Added
- Horizon now exposes the stellar network protocol in several places:  It shows the currently reported protocol version (as returned by the stellar-core `info` command) on the root endpoint, and it reports the protocol version of each ledger resource.
- Trade resources now include a `created_at` timestamp.

### Fixed

- BREAKING CHANGE: The reingestion process has been updated.  Prior versions of horizon would enter a failed state when a gap between the imported history and the stellar-core database formed or when a previously imported ledger was no longer found in the stellar-core database.  This usually occurs when running stellar-core with the `CATCHUP_RECENT` config option.  With these changed, horizon will automatically trim "abandonded" ledgers: ledgers that are older than the core elder ledger.


## [v0.8.0] - 2017-02-07

### Added

- account signer resources now contain a type specifying the type of the signer: `ed25519_public_key`, `sha256_hash`, and `preauth_tx` are the present values used for the respective signer types.

### Changed

- The `public_key` field on signer effects and an account's signer summary has been renamed to `key` to reflect that new signer types are not necessarily specifying a public key anymore.

### Deprecated

- The `public_key` field on account signers and signer effects are deprecated

## [v0.7.1] - 2017-01-12

### Bug fixes

- Trade resources now include "bought_amount" and "sold_amount" fields when being viewed through the "Orderbook Trades" endpoint.
- Fixes #322: orderbook summaries with over 20 bids now return the correct price levels, starting with the closest to the spread.

## [v0.7.0] - 2017-01-10

### Added

- The account resource now includes links to the account's trades and data values.

### Bug fixes

- Fixes paging_token attribute of account resource
- Fixes race conditions in friendbot
- Fixes #202: Add price and price_r to "manage_offer" operation resources
- Fixes #318: order books for the native currency now filters correctly.

## [v0.6.2] - 2016-08-18

### Bug fixes

- Fixes streaming (SSE) requests, which were broken in v0.6.0

## [v0.6.1] - 2016-07-26

### Bug fixes

- Fixed an issue where accounts were not being properly returned when the  history database had no record of the account.


## [v0.6.0] - 2016-07-20

This release contains the initial implementation of the "Abridged History System".  It allows a horizon system to be operated without complete knowledge of the ledger's history.  With this release, horizon will start ingesting data from the earliest point known to the connected stellar-core instance, rather than ledger 1 as it behaved previously.  See the admin guide section titled "Ingesting stellar-core data" for more details.

### Added

- *Elder* ledgers have been introduced:  An elder ledger is the oldest ledger known to a db.  For example, the `core_elder_ledger` attribute on the root endpoint refers to the oldest known ledger stored in the connected stellar-core database.
- Added the `history-retention-count` command line flag, used to specify the amount of historical data to keep in the history db.  This is expressed as a number of ledgers, for example a value of `362880` would retain roughly 6 weeks of data given an average of 10 seconds per ledger.
- Added the `history-stale-threshold` command line flag to enable stale history protection.  See the admin guide for more info.
- Horizon now reports the last ledger ingested to stellar-core using the `setcursor` command.
- Requests for data that precede the recorded window of history stored by horizon will receive a `410 Gone` http response to allow software to differentiate from other "not found" situations.
- The new `db reap` command will manually trigger the deletion of unretained historical data
- A background process on the server now deletes unretained historical once per hour.

### Changed

- BREAKING: When making a streaming request, a normal error response will be returned if an error occurs prior to sending the first event.  Additionally, the initial http response and streaming preamble will not be sent until the first event is available.
- BREAKING: `horizon_latest_ledger` has renamed to `history_latest_ledger`
- Horizon no longer needs to begin the ingestion of historical data from ledger sequence 1.
- Rows in the `history_accounts` table are no longer identified using the "Total Order ID" that other historical records  use, but are rather using a simple auto-incremented id.

### Removed

- The `/accounts` endpoint, which lets a consumer page through the entire set of accounts in the ledger, has been removed.  The change from complete to an abridged history in horizon makes the endpoint mostly useless, and after consulting with the community we have decided to remove the endpoint.

## [v0.5.1] - 2016-04-28

### Added

  - ManageData operation data is now rendered in the various operation end points.

### Bug fixes

- Transaction memos that contain utf-8 are now properly rendered in browsers by properly setting the charset of the http response.

## [v0.5.0] - 2016-04-22

### Added

- BREAKING: Horizon can now import data from stellar-core without the aid of the horizon-importer project.  This process is now known as "ingestion", and is enabled by either setting the `INGEST` environment variable to "true" or specifying "--ingest" on the launch arguments for the horizon process.  Only one process should be running in this mode for any given horizon database.
- Add `horizon db init`, used to install the latest bundled schema for the horizon database.
- Add `horizon db reingest` command, used to update outdated or corrupt horizon database information.  Admins may now use `horizon db reingest outdated` to migrate any old data when updated horizon.
- Added `network_passphrase` field to root resource.
- Added `fee_meta_xdr` field to transaction resource.

### Bug fixes
- Corrected casing on the "offers" link of an account resource.

## [v0.4.0] - 2016-02-19

### Added

- Add `horizon db migrate [up|down|redo]` commands, used for installing schema migrations.  This work is in service of porting the horizon-importer project directly to horizon.
- Add support for TLS: specify `--tls-cert` and `--tls-key` to enable.
- Add support for HTTP/2.  To enable, use TLS.

### Removed

- BREAKING CHANGE: Removed support for building on go versions lower than 1.6

## [v0.3.0] - 2016-01-29

### Changes

- Fixed incorrect `source_amount` attribute on pathfinding responses.
- BREAKING CHANGE: Sequence numbers are now encoded as strings in JSON responses.
- Fixed broken link in the successful response to a posted transaction

## [v0.2.0] - 2015-12-01
### Changes

- BREAKING CHANGE: the `address` field of a signer in the account resource has been renamed to `public_key`.
- BREAKING CHANGE: the `address` on the account resource has been renamed to `account_id`.

## [v0.1.1] - 2015-12-01

### Added
- Github releases are created from tagged travis builds automatically

[v0.11.0]: https://github.com/stellar/horizon/compare/v0.10.1...v0.11.0
[v0.10.1]: https://github.com/stellar/horizon/compare/v0.10.0...v0.10.1
[v0.10.0]: https://github.com/stellar/horizon/compare/v0.9.1...v0.10.0
[v0.9.1]: https://github.com/stellar/horizon/compare/v0.9.0...v0.9.1
[v0.9.0]: https://github.com/stellar/horizon/compare/v0.8.0...v0.9.0
[v0.8.0]: https://github.com/stellar/horizon/compare/v0.7.1...v0.8.0
[v0.7.1]: https://github.com/stellar/horizon/compare/v0.7.0...v0.7.1
[v0.7.0]: https://github.com/stellar/horizon/compare/v0.6.2...v0.7.0
[v0.6.2]: https://github.com/stellar/horizon/compare/v0.6.1...v0.6.2
[v0.6.1]: https://github.com/stellar/horizon/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/stellar/horizon/compare/v0.5.1...v0.6.0
[v0.5.1]: https://github.com/stellar/horizon/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/stellar/horizon/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/stellar/horizon/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/stellar/horizon/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/stellar/horizon/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/stellar/horizon/compare/v0.1.0...v0.1.1
