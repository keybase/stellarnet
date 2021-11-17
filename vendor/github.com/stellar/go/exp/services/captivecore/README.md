# captivecore

The Captive Stellar-Core Server allows you to run a dedicated Stellar-Core instance
for the purpose of ingestion. The server must be bundled with a Stellar Core binary.

If you run Horizon with Captive Stellar-Core ingestion enabled Horizon will spawn a Stellar-Core
subprocess. Horizon's ingestion system will then stream ledgers from the subprocess via
a filesystem pipe. The disadvantage of running both Horizon and the Stellar-Core subprocess
on the same machine is it requires detailed per-process monitoring to be able to attribute
potential issues (like memory leaks) to a specific service.

Now you can run Horizon and pair it with a remote Captive Stellar-Core instance. The
Captive Stellar-Core Server can run on a separate machine from Horizon. The server
will manage Stellar-Core as a subprocess and provide an HTTP API which Horizon
can use remotely to stream ledgers for the purpose of ingestion.

Note that, currently, a single Captive Stellar-Core Server cannot be shared by
multiple Horizon instances.

## API

### `GET /latest-sequence`

Fetches the latest ledger sequence available on the captive core instance.

Response:

```json
{
	"sequence": 12345
}
```


### `GET /ledger/<sequence>`

Fetches the ledger with the given sequence number from the captive core instance.

Response:


```json
{
    "present": true,
    "ledger": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
}
```

### `POST /prepare-range`

Preloads the given range of ledgers in the captive core instance.

Bounded request:
```json
{
    "from": 123,
    "to":   150,
    "bounded": true
}
```

Unbounded request:
```json
{
    "from": 123,
    "bounded": false
}
```

Response:
```json
{
    "ledgerRange": {"from":  123, "bounded":  false},
    "startTime": "2020-08-31T13:29:09Z",
    "ready": true,
    "readyDuration": 1000
}
```

## Usage

```
$ captivecore --help
Run the Captive Stellar-Core Server

Usage:
  captivecore [flags]

Flags:
      --db-url                             Horizon Postgres URL (optional) used to lookup the ledger hash for sequence numbers
      --stellar-core-binary-path           Path to stellar core binary
      --stellar-core-config-path           Path to stellar core config file
      --history-archive-urls               Comma-separated list of stellar history archives to connect with
      --log-level                          Minimum log severity (debug, info, warn, error) to log (default info)
      --network-passphrase string          Network passphrase of the Stellar network transactions should be signed for (NETWORK_PASSPHRASE) (default "Test SDF Network ; September 2015")
      --port int                           Port to listen and serve on (PORT) (default 8000)
```