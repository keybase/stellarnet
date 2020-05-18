module github.com/keybase/stellarnet

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-errors/errors v1.0.2-0.20180813162953-d98b870cc4e0 // indirect
	github.com/keybase/vcr v0.0.0-20191017153547-a32d93056205
	github.com/lib/pq v1.2.1-0.20190919160911-931b5ae4c24e // indirect
	github.com/pkg/errors v0.8.2-0.20190227000051-27936f6d90f9
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/shopspring/decimal v1.1.1-0.20191009025716-f1972eb1d1f5
	github.com/stellar/go v0.0.0-20191010205648-0fc3bfe3dfa7
	github.com/stretchr/testify v1.4.0
)

replace github.com/stellar/go => github.com/keybase/stellar-org v0.0.0-20191010205648-0fc3bfe3dfa7
