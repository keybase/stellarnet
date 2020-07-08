package stellarnet

import (
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

// NewKeyPair creates a new random stellar keypair.
func NewKeyPair() (*keypair.Full, error) {
	return keypair.Random()
}

// SeedStr is a string representation of a private stellar key.
type SeedStr string

// AddressStr is a string representation of a public stellar key.
type AddressStr string

// NewSeedStr ensures that s is a valid stellar seed.
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

func (s SeedStr) String() string {
	return "DONOTLOGDONOTLOGDONOTLOGDONOTLOGDONOTLOGDONOTLOGDONOTLOG"
}

// SecureNoLogString returns a native string representation of SeedStr.
// It should not be logged or persisted anywhere.
func (s SeedStr) SecureNoLogString() string {
	return string(s)
}

// Address returns the public address for a seed.
func (s SeedStr) Address() (AddressStr, error) {
	kp, err := keypair.Parse(s.SecureNoLogString())
	if err != nil {
		return "", err
	}
	return AddressStr(kp.Address()), nil
}

// NewAddressStr ensures that s is a valid stellar address.
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

// AccountID converts an AddressStr into an xdr.AccountId.
func (s AddressStr) AccountID() (acctID xdr.AccountId, err error) {
	err = acctID.SetAddress(s.String())
	return acctID, err
}

// MuxedAccount converts an AddressStr into an xdr.MuxedAccount
func (s AddressStr) MuxedAccount() (mux xdr.MuxedAccount, err error) {
	err = mux.SetAddress(s.String())
	if err != nil {
		return xdr.MuxedAccount{}, err
	}
	return mux, err
}

// AddressStrFromMuxedAccount returns address encoded into MuxedAccount as
// AddressStr. If MuxedAccount had a memoID, it is dropped.
func AddressStrFromMuxedAccount(mux xdr.MuxedAccount) (addrStr AddressStr, err error) {
	acctID := mux.ToAccountId()
	addr, err := acctID.GetAddress()
	if err != nil {
		return "", err
	}
	return NewAddressStr(addr)
}

// MuxedAccountToAccountString converts MuxedAccount's AccountID to string. If
// MuxedAccount had a memoID, it is dropped.
// NOTE: Stellar at some point will introduce string encoding for Accounts that
// include MemoID. (see SEP-23). That will warrant another function, or maybe
// there will be one provided by stellar/go.
func MuxedAccountToAccountString(acc xdr.MuxedAccount) string {
	addr := acc.ToAccountId()
	return addr.Address()
}
