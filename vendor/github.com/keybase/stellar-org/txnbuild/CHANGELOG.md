# Changelog

All notable changes to this project will be documented in this
file.  This project adheres to [Semantic Versioning](http://semver.org/).

## [v1.2.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.2.0) - 2019-05-16

* In addition to account responses from horizon, transactions and operations can now be built with txnbuild.SimpleAccount structs constructed locally ([#1266](https://github.com/stellar/go/issues/1266)). 
* Added `MaxTrustlineLimit` which represents the maximum value for a trustline limit ([#1265](https://github.com/stellar/go/issues/1265)).
* ChangeTrust operation with no `Limit` field set now defaults to `MaxTrustlineLimit` ([#1265](https://github.com/stellar/go/issues/1265)).
* Add support for building `ManageBuyOffer` operation ([#1165](https://github.com/stellar/go/issues/1165)).
* Fix bug in ChangeTrust operation builder ([1296](https://github.com/stellar/go/issues/1296)).

## [v1.1.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.1.0) - 2019-02-02

* Support for multiple signatures ([#1198](https://github.com/stellar/go/pull/1198))

## [v1.0.0](https://github.com/stellar/go/releases/tag/horizonclient-v1.0) - 2019-04-26

* Initial release
