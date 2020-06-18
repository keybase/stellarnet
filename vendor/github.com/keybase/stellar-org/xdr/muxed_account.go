package xdr

import (
	"errors"
	"fmt"

	"github.com/stellar/go/strkey"
)

// SetAddress modifies the receiver, setting it's value to the MuxedAccount form
// of the provided address.
func (m *MuxedAccount) SetAddress(address string) error {
	if m == nil {
		return nil
	}

	switch len(address) {
	case 56:
		raw, err := strkey.Decode(strkey.VersionByteAccountID, address)
		if err != nil {
			return err
		}
		if len(raw) != 32 {
			return errors.New("invalid address")
		}
		var ui Uint256
		copy(ui[:], raw)
		*m, err = NewMuxedAccount(CryptoKeyTypeKeyTypeEd25519, ui)
		return err
	default:
		return errors.New("invalid address")
	}

}

// ToAccountId transforms a MuxedAccount to an AccountId, dropping the
// memo Id if necessary
func (m MuxedAccount) ToAccountId() AccountId {
	result := AccountId{Type: PublicKeyTypePublicKeyTypeEd25519}
	switch m.Type {
	case CryptoKeyTypeKeyTypeEd25519:
		ed := m.MustEd25519()
		result.Ed25519 = &ed
	case CryptoKeyTypeKeyTypeMuxedEd25519:
		ed := m.MustMed25519().Ed25519
		result.Ed25519 = &ed
	default:
		panic(fmt.Errorf("Unknown muxed account type: %v", m.Type))
	}
	return result
}
