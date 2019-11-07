# Beta Testing New Ingestion System

This document aims to be a guide with all information needed to start testing new ingestion system in Horizon. If there are any issues you are facing not covered in this document, please add an issue in this repository. For questions please use [Stack Exchange](https://stellar.stackexchange.com) or ask it in one of our online [communities](https://www.stellar.org/community/#communities).

Please remember that this is still experimental version of the system!

Thank you for helping us test new ingestion system in Horizon!

## What is the new ingestion system?

The new ingestion system solves issues found in the previous version like: inconsistent data, relying on Stellar-Core database directly, slow responses for specific queries, etc. It allows new kind of features in Horizon, ex. faster path-finding. We published a [blog post](https://www.stellar.org/developers/blog/our-new-horizon-ingestion-engine) with more details, please check it out!

## Prerequisities

* The init stage (state ingestion) for public network requires around 1.5GB of RAM. The memory is released after the state ingestion. State ingestion is performed only once, restarting the server will not trigger it unless Horizon has been upgraded to a newer version (with updated ingestion pipeline). We are currently working on alternative solutions to make RAM requirements smaller however we believe that it will become smaller and smaller as more buckets are [CAP-20](https://github.com/stellar/stellar-protocol/blob/master/core/cap-0020.md) compatible. The CPU footprint of the new ingestion is really small. We were able to run experimental ingestion on `c5.large` instance on AWS. The init stage takes a few minutes on `c5.large`.
* The state data requires additional 1.3GB DB disk space (as of version 0.20.1).
* Flags needed to enable experimental ingestion:
  * `ENABLE_EXPERIMENTAL_INGESTION=true`
  * `HISTORY_ARCHIVE_URLS="archive1,archive2,archive3"` (for public network you can use one of SDF's archives, ex. `https://history.stellar.org/prd/core-live/core_live_001`)

## Known issues

### Some endpoints are not available during state ingestion

Endpoints that display state information like `/paths` (built using offers) are not available during state ingestion and return `503 Service Unavailable`/`Still Ingesting` error. We are currently thinking whether to change this behaviour. The main argument against it is displaying stale data.

### State ingestion is taking a lot of time

Since Horizon 0.21.0 the state ingestion shouldn't take more than a couple minutes on AWS `c5.large` instance.

It's possible that the progress logs (see below) will not show anything new for a longer period of time or print a lot of progress entries every few seconds. This is happening because of the way history archives are designed, the ingestion is still working but it's processing so called `DEADENTRY`'ies: if there is a lot of them in the bucket, there are no _active_ entries to process. We plan to improve the progress logs to display actual percentage progress so it's easier to estimate ETA.

If you see that ingestion is not proceeding for a very long period of time:
1. Check the RAM usage on the machine. It's possible that system run out of RAM and it using swap memory that is extremely slow.
2. If above is not the case, file a new issue in this repository.

### CPU usage goes high every few minutes

This is _by design_. Horizon 0.21.0 introduced a state verifier routine that compares state in local storage to history archives every 64 ledgers to ensure data changes are applied correctly. If data corruption is detected Horizon will block access to endpoints serving invalid data.

We recommend to keep this security feature turned on however if it's causing problems (due to CPU usage) this can be disabled by `--ingest-disable-state-verification` CLI param or `INGEST-DISABLE-STATE-VERIFICATION` env variable.

## Reading the logs

In order to check the progress and the status of experimental ingestion you should check the logs. All logs connected to experimental ingestion are tagged with `service=expingest`.

It starts with informing you about state ingestion:
```
INFO[2019-08-29T13:04:13.473+02:00] Starting ingestion system from empty state...  pid=5965 service=expingest temp_set="*io.MemoryTempSet"
INFO[2019-08-29T13:04:15.263+02:00] Reading from History Archive Snapshot         ledger=25565887 pid=5965 service=expingest
```
During state ingestion, Horizon will log number of processed entries every 100,000 entries (there are currently around 5.5M entries in the public network):
```
INFO[2019-08-29T13:04:34.652+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=100000 pid=5965 service=expingest
INFO[2019-08-29T13:04:38.487+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=200000 pid=5965 service=expingest
INFO[2019-08-29T13:04:41.322+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=300000 pid=5965 service=expingest
INFO[2019-08-29T13:04:48.429+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=400000 pid=5965 service=expingest
INFO[2019-08-29T13:05:00.306+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=500000 pid=5965 service=expingest
```
When state ingestion is finished it will proceed to ledger ingestion starting from the next ledger after checkpoint ledger (25565887+1 in this example) to update the state using transaction meta:
```
INFO[2019-08-29T13:39:41.590+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=5300000 pid=5965 service=expingest
INFO[2019-08-29T13:39:44.518+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=5400000 pid=5965 service=expingest
INFO[2019-08-29T13:39:47.488+02:00] Processing entries from History Archive Snapshot  ledger=25565887 numEntries=5500000 pid=5965 service=expingest
INFO[2019-08-29T13:40:00.670+02:00] Processed ledger                              ledger=25565887 pid=5965 service=expingest type=state_pipeline
INFO[2019-08-29T13:40:00.670+02:00] Finished processing History Archive Snapshot  duration=2145.337575904 ledger=25565887 numEntries=5529931 pid=5965 service=expingest shutdown=false
INFO[2019-08-29T13:40:00.693+02:00] Reading new ledger                            ledger=25565888 pid=5965 service=expingest
INFO[2019-08-29T13:40:00.694+02:00] Processing ledger                             ledger=25565888 pid=5965 service=expingest type=ledger_pipeline updating_database=true
INFO[2019-08-29T13:40:00.779+02:00] Processed ledger                              ledger=25565888 pid=5965 service=expingest type=ledger_pipeline
INFO[2019-08-29T13:40:00.779+02:00] Finished processing ledger                    duration=0.086024492 ledger=25565888 pid=5965 service=expingest shutdown=false transactions=14
INFO[2019-08-29T13:40:00.815+02:00] Reading new ledger                            ledger=25565889 pid=5965 service=expingest
INFO[2019-08-29T13:40:00.816+02:00] Processing ledger                             ledger=25565889 pid=5965 service=expingest type=ledger_pipeline updating_database=true
INFO[2019-08-29T13:40:00.881+02:00] Processed ledger                              ledger=25565889 pid=5965 service=expingest type=ledger_pipeline
INFO[2019-08-29T13:40:00.881+02:00] Finished processing ledger                    duration=0.06619956 ledger=25565889 pid=5965 service=expingest shutdown=false transactions=29
INFO[2019-08-29T13:40:00.901+02:00] Reading new ledger                            ledger=25565890 pid=5965 service=expingest
INFO[2019-08-29T13:40:00.902+02:00] Processing ledger                             ledger=25565890 pid=5965 service=expingest type=ledger_pipeline updating_database=true
INFO[2019-08-29T13:40:00.972+02:00] Processed ledger                              ledger=25565890 pid=5965 service=expingest type=ledger_pipeline
INFO[2019-08-29T13:40:00.972+02:00] Finished processing ledger                    duration=0.071039012 ledger=25565890 pid=5965 service=expingest shutdown=false transactions=20
```
