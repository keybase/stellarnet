---
title: Overview
---

Horizon is an API server for the Stellar ecosystem.  It acts as the interface between [stellar-core](https://github.com/stellar/stellar-core) and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams, etc. See [an overview of the Stellar ecosystem](https://www.stellar.org/developers/guides/) for details of where Horizon fits in. You can also watch a [talk on Horizon](https://www.youtube.com/watch?v=AtJ-f6Ih4A4) by Stellar.org developer Scott Fleckenstein:

[![Horizon: API webserver for the Stellar network](https://img.youtube.com/vi/AtJ-f6Ih4A4/sddefault.jpg "Horizon: API webserver for the Stellar network")](https://www.youtube.com/watch?v=AtJ-f6Ih4A4)

Horizon provides a RESTful API to allow client applications to interact with the Stellar network. You can communicate with Horizon using cURL or just your web browser. However, if you're building a client application, you'll likely want to use a Stellar SDK in the language of your client.
SDF provides a [JavaScript SDK](https://www.stellar.org/developers/js-stellar-sdk/learn/index.html) for clients to use to interact with Horizon.

SDF runs a instance of Horizon that is connected to the test net: [https://horizon-testnet.stellar.org/](https://horizon-testnet.stellar.org/) and one that is connected to the public Stellar network:
[https://horizon.stellar.org/](https://horizon.stellar.org/).

## Libraries

SDF maintained libraries:<br />
- [JavaScript](https://github.com/stellar/js-stellar-sdk)
- [Go](https://github.com/stellar/go/tree/master/clients/horizonclient)
- [Java](https://github.com/stellar/java-stellar-sdk)

Community maintained libraries (in various states of completeness) for interacting with Horizon in other languages:<br>
- [Python](https://github.com/StellarCN/py-stellar-base)
- [C# .NET Core 2.x](https://github.com/elucidsoft/dotnetcore-stellar-sdk)
- [Ruby](https://github.com/bloom-solutions/ruby-stellar-sdk)
- [iOS and macOS](https://github.com/Soneso/stellar-ios-mac-sdk)
- [Scala SDK](https://github.com/synesso/scala-stellar-sdk)
- [C++ SDK](https://github.com/bnogalm/StellarQtSDK)
