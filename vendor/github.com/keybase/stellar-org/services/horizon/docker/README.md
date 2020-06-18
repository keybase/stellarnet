# Running Stellar with Docker Compose

## Dependencies

The only dependency you will need to install is [Docker](https://www.docker.com/products/docker-desktop).

## Start script

[start.sh](./start.sh) will setup the env file and run docker-compose to start the Stellar docker containers. Feel free to use this script, otherwise continue with the next two steps.

The script takes one optional parameter which configures the Stellar network used by the docker containers. If no parameter is supplied, the containers will run on the Stellar test network.

`./start.sh pubnet` will run the containers on the Stellar public network.

`./start.sh standalone` will run the containers on a private standalone Stellar network.

## Set up a .env file

Mac OS X and Windows users should create an [`.env`](https://docs.docker.com/compose/environment-variables/#the-env_file-configuration-option) file which consists of:

`NETWORK_MODE=bridge`

Linux users should also create an `.env` file. However, the contents of the file should look like:

`NETWORK_MODE=host`

Additionally, you will need to add `127.0.0.1 host.docker.internal` to the `/etc/hosts` file on your linux machine.

If https://github.com/docker/for-linux/issues/264 is ever fixed then it won't be necessary to alias `host.docker.internal` to localhost and there won't be any differences between the Linux and Mac OS X / Windows configurations.


## Run docker-compose

Run the following command to start all the Stellar docker containers:

```
docker-compose up -d --build
```

Horizon will be exposed on port 8000. Stellar Core will be exposed on port 11626. The Stellar Core postgres instance will be exposed on port 5641.
The Horizon postgres instance will be exposed on port 5432.

## Swapping in a local service

If you're developing a service locally you may want to run that service locally while also being able to interact with the other Stellar components running in Docker. You can do that by stopping the container corresponding to the service you're developing.

For example, to run Horizon locally from source, you would perform the following steps:

```
# stop horizon in docker-compose
docker-compose stop horizon
```

Now you can run horizon locally in vscode using the following configuration:
```
    {
        "name": "Launch",
        "type": "go",
        "request": "launch",
        "mode": "debug",
        "remotePath": "",
        "port": 2345,
        "host": "127.0.0.1",
        "program": "${workspaceRoot}/services/horizon/main.go",
        "env": {
            "DATABASE_URL": "postgres://postgres@localhost:5432/horizon?sslmode=disable",
            "STELLAR_CORE_DATABASE_URL": "postgres://postgres:mysecretpassword@localhost:5641/stellar?sslmode=disable",
            "NETWORK_PASSPHRASE": "Test SDF Network ; September 2015",
            "STELLAR_CORE_URL": "http://localhost:11626",
            "INGEST": "true",
        },
        "args": []
    }
```

Similarly, to run Stellar core locally from source and have it interact with Horizon in docker, all you need to do is run `docker-compose stop core` before running Stellar core from source.

## Connecting to the Stellar Public Network

By default, the Docker Compose file configures Stellar Core to connect to the Stellar test network. If you would like to run the docker containers on the
Stellar public network, run `docker-compose -f docker-compose.yml -f docker-compose.pubnet.yml up -d --build`. 

To run the containers on a private stand-alone network, run `docker-compose -f docker-compose.yml -f docker-compose.standalone.yml up -d --build`.
When you run Stellar Core on a private stand-alone network, an account will be created which will hold 100 billion Lumens.
The seed for the account will be emitted in the Stellar Core logs:

```
2020-04-22T18:39:19.248 GD5KD [Ledger INFO] Root account seed: SC5O7VZUXDJ6JBDSZ74DSERXL7W3Y5LTOAMRF7RQRL3TAGAPS7LUVG3L
```

When running Horizon on a private stand-alone network, Horizon will not start ingesting until Stellar Core creates its first history archive snapshot. Stellar Core creates snapshots every 64 ledgers, which means ingestion will be delayed until ledger 64.

When you switch between different networks you will need to clear the Stellar Core and Stellar Horizon databases. You can wipe out the databases by running `docker-compose down --remove-orphans -v`.

## Using a specific version of Stellar Core

By default the Docker Compose file is configured to use the latest version of Stellar Core. To use a specific version, you can edit [docker-compose.yml](./docker-compose.yml) and set the appropriate [tag](https://hub.docker.com/r/stellar/stellar-core/tags) on the Stellar Core docker image
