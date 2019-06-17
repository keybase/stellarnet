# Horizon
[![Build Status](https://circleci.com/gh/stellar/go.svg?style=shield)](https://circleci.com/gh/stellar/go)

Horizon is the client facing API server for the [Stellar ecosystem](https://www.stellar.org/developers/guides/get-started/).  It acts as the interface between [Stellar Core](https://www.stellar.org/developers/stellar-core/software/admin.html) and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.

## Try it out
See Horizon in action by running your own Stellar node as part of the Stellar [testnet](https://www.stellar.org/developers/guides/concepts/test-net.html). With our Docker quick-start image, you can be running your own fully functional node in around 20 minutes. See the [Quickstart Guide](internal/docs/quickstart.md) to get up and running.

## Run a production server
If you're an administrator planning to run a production instance of Horizon as part of the public Stellar network, check out the detailed [Administration Guide](internal/docs/admin.md). It covers installation, monitoring, error scenarios and more.

## Contributing
As an open source project, development of Horizon is public, and you can help! We welcome new issue reports, documentation and bug fixes, and contributions that further the project roadmap. The [Development Guide](internal/docs/developing.md) will show you how to build Horizon, see what's going on behind the scenes, and set up an effective develop-test-push cycle so that you can get your work incorporated quickly.
