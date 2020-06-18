---
title: Horizon
---

Horizon is the server for the client facing API for the Stellar ecosystem.  It acts as the interface between [stellar-core](https://www.stellar.org/developers/software/#stellar-core) and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams, etc. See [an overview of the Stellar ecosystem](https://www.stellar.org/developers/guides/) for more details.

You can interact directly with horizon via curl or a web browser but SDF provides a [JavaScript SDK](https://www.stellar.org/developers/js-stellar-sdk/reference/) for clients to use to interact with Horizon.

SDF runs a instance of Horizon that is connected to the test net [https://horizon-testnet.stellar.org/](https://horizon-testnet.stellar.org/).

## Libraries

SDF maintained libraries:<br />
- [JavaScript](https://github.com/stellar/js-stellar-sdk)
- [Go](https://github.com/stellar/go/tree/master/clients/horizonclient)
- [Java](https://github.com/stellar/java-stellar-sdk)

Community maintained libraries for interacting with Horizon in other languages:<br>
- [Python](https://github.com/StellarCN/py-stellar-base)
- [C# .NET Core 2.x](https://github.com/elucidsoft/dotnetcore-stellar-sdk)
- [Ruby](https://github.com/astroband/ruby-stellar-sdk)
- [iOS and macOS](https://github.com/Soneso/stellar-ios-mac-sdk)
- [Scala SDK](https://github.com/synesso/scala-stellar-sdk)
- [C++ SDK](https://github.com/bnogalm/StellarQtSDK)
