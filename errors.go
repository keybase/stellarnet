package stellarnet

import "errors"

var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrAddressNotSeed     = errors.New("string provided is an address not a seed")
	ErrSeedNotAddress     = errors.New("string provided is a seed not an address")
	ErrUnknownKeypairType = errors.New("unknown keypair type")
)
