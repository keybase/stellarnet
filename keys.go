package stellarnet

import "github.com/stellar/go/keypair"

func NewKeyPair() (*keypair.Full, error) {
	return keypair.Random()
}

type SeedStr string
type AddressStr string

func NewSeedStr(s string) (SeedStr, error) {
	// parse s to make sure it is a valid seed
	kp, err := keypair.Parse(s)
	if err != nil {
		return "", err
	}

	switch kp.(type) {
	case *keypair.Full:
		return SeedStr(s), nil
	case *keypair.FromAddress:
		return "", ErrAddressNotSeed
	}

	return "", ErrUnknownKeypairType
}

func (s SeedStr) String() string { return string(s) }

func NewAddressStr(s string) (AddressStr, error) {
	// parse s to make sure it is a valid address
	kp, err := keypair.Parse(s)
	if err != nil {
		return "", err
	}

	switch kp.(type) {
	case *keypair.FromAddress:
		return AddressStr(s), nil
	case *keypair.Full:
		return "", ErrSeedNotAddress
	}

	return "", ErrUnknownKeypairType
}

func (s AddressStr) String() string { return string(s) }
