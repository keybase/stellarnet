---
title: Metrics
---

The metrics endpoint returns a host of [Prometheus](https://prometheus.io/) metrics for monitoring the health of the underlying Horizon process.

There is an [official Grafana Dashboard](https://grafana.com/grafana/dashboards/13793) to easily visualize those metrics.

Since Horizon 1.0.0 this endpoint is not part of the public API. It's available in the internal server (listening on the internal port set via `ADMIN_PORT` env variable or `--admin-port` CLI param).

## Request

```
GET /metrics
```

### curl Example Request

Assuming a local Horizon instance is running with an admin port of 9090 (i.e. `ADMIN_PORT=9090` env variable or `--admin-port=9090`)

```sh
curl "https://localhost:9090/metrics"
```


## Response

The `/metrics` endpoint returns a [Prometheus text-formated](https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format) response. It is meant to be scraped by Prometheus.

Below, each section of related data points are grouped together and annotated (***note**: this endpoint returns ALL this data in one response*).


#### Goroutines

Horizon utilizes Go's built in concurrency primitives ([goroutines](https://gobyexample.com/goroutines) and [channels](https://gobyexample.com/channels)). The `goroutine` metric monitors the number of currently running goroutines on this Horizon's process.


#### History

Horizon maintains its own database (postgres), a verbose and user friendly account of activity on the Stellar network.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| history.elder_ledger     | The sequence number of the oldest ledger recorded in Horizon's database. |
| history.latest_ledger    | The sequence number of the youngest (most recent) ledger recorded in Horizon's database.  |
| history.open_connections | The number of open connections to the Horizon database. |


#### Ingester

Ingester represents metrics specific to Horizon's [ingestion](https://github.com/stellar/go/blob/master/services/horizon/internal/docs/reference/admin.md#ingesting-stellar-core-data) process, or the process by which Horizon consumes transaction results from a connected Stellar Core instance.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| ingester.clear_ledger |  The count and rate of clearing (per ledger) for this Horizon process.  |
| ingester.ingest_ledger | The count and rate of ingestion (per ledger)  for this Horizon process. |

These metrics contain useful [sub metrics](#sub-metrics).


#### Logging

Horizon utilizes the standard `debug`, `error`, etc. levels of logging. This metric outputs stats for each level of log message produced, useful for a high-level monitoring of "is my Horizon instance functioning properly?" In order of increasing severity:

* logging.debug
* logging.info
* logging.warning
* logging.error
* logging.panic

These metrics contain useful [sub metrics](#sub-metrics).

#### Requests

Requests represent an overview of Horizon's incoming traffic.

These metrics contain useful [sub metrics](#sub-metrics).

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| requests.failed | Failed requests are those that return a status code in [400, 600). |
| requests.succeeded | Successful requests are those that return a status code in [200, 400). |
| requests.total | Total number of received requests.  |

#### Stellar Core
As noted above, Horizon relies on Stellar Core to stay in sync with the Stellar network. These metrics are specific to the underlying Stellar Core instance.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| stellar_core.latest_ledger    | The sequence number of the latest (most recent) ledger recorded in Stellar Core's database.  |
| stellar_core.open_connections | The number of open connections to the Stellar Core postgres database.  |

#### Transaction Submission

Horizon does not submit transactions directly to the Stellar network. Instead, it sequences transactions and sends the base64 encoded, XDR serialized blob to its connected Stellar Core instance. 

##### Horizon Transaction Sequencing and Submission

The following is a simplified version of the transaction submission process that glosses over the finer details. To dive deeper, check out the [source code](https://github.com/stellar/go/tree/master/services/horizon/internal/txsub).

Horizon's sequencing mechanism consists of a [manager](https://github.com/stellar/go/blob/master/services/horizon/internal/txsub/sequence/manager.go) that keeps track of [submission queues](https://github.com/stellar/go/blob/master/services/horizon/internal/txsub/sequence/queue.go) for a set of addresses. A submission queue is a  priority queue, prioritized by minimum transaction sequence number, that holds a set of pending transactions for an account. A pending transaction is represented as an object with a sequence number and a channel. Periodically, this queue is updated, popping off finished transactions, sending down the transaction's channel a successful/failure response.

These metrics contain useful [sub metrics](#sub-metrics).


|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| txsub.buffered | The count of submissions buffered behind this Horizon's submission queue.  |
| txsub.failed | The rate of failed transactions that have been submitted to this Horizon.  |
| txsub.open | The count of "open" submissions (i.e.) submissions whose transactions haven't been confirmed successful or failed.  |
| txsub.succeeded | The rate of successful transactions that have been submitted to this Horizon.  |
| txsub.total | Both the rate and count of all transactions submitted to this Horizon. |

### Sub Metrics
Various sub metrics related to a certain metric's performance.

|    Metric     |  Description                                                                                                                               |
| ---------------- |  ------------------------------------------------------------------------------------------------------------------------------ |
| `1m.rate`, `5min.rate`, `etc.` | The per-minute moving average rate of events per second at the given time interval.  |
| `75%`, `95%`, `etc.` | Counts at different percentiles.  |
| `count` | Sum total of a certain metric value.  |
| `max`, `mean`, `etc.` |  Common statistic calculations. |




