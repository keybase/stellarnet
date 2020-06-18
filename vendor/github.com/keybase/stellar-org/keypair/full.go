package keypair

import (
	"bytes"
	"encoding/base64"

	"github.com/stellar/go/strkey"
	"github.com/stellar/go/xdr"

	"golang.org/x/crypto/ed25519"
)

type Full struct {
	seed string
}

func (kp *Full) Address() string {
	return strkey.MustEncode(strkey.VersionByteAccountID, kp.publicKey()[:])
}

// FromAddress gets the address-only representation, or public key, of this
// Full keypair.
func (kp *Full) FromAddress() *FromAddress {
	return &FromAddress{address: kp.Address()}
}

func (kp *Full) Hint() (r [4]byte) {
	copy(r[:], kp.publicKey()[28:])
	return
}

func (kp *Full) Seed() string {
	return kp.seed
}

func (kp *Full) Verify(input []byte, sig []byte) error {
	if len(sig) != 64 {
		return ErrInvalidSignature
	}
	if !ed25519.Verify(kp.publicKey(), input, sig) {
		return ErrInvalidSignature
	}
	return nil
}

func (kp *Full) Sign(input []byte) ([]byte, error) {
	_, priv := kp.keys()
	return ed25519.Sign(priv, input), nil
}

// SignBase64 signs the input data and returns a base64 encoded string, the
// common format in which signatures are exchanged.
func (kp *Full) SignBase64(input []byte) (string, error) {
	sig, err := kp.Sign(input)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (kp *Full) SignDecorated(input []byte) (xdr.DecoratedSignature, error) {
	sig, err := kp.Sign(input)
	if err != nil {
		return xdr.DecoratedSignature{}, err
	}

	return xdr.DecoratedSignature{
		Hint:      xdr.SignatureHint(kp.Hint()),
		Signature: xdr.Signature(sig),
	}, nil
}

func (kp *Full) publicKey() ed25519.PublicKey {
	pub, _ := kp.keys()
	return pub
}

func (kp *Full) keys() (ed25519.PublicKey, ed25519.PrivateKey) {
	reader := bytes.NewReader(kp.rawSeed())
	pub, priv, err := ed25519.GenerateKey(reader)
	if err != nil {
		panic(err)
	}
	return pub, priv
}

func (kp *Full) rawSeed() []byte {
	return strkey.MustDecode(strkey.VersionByteSeed, kp.seed)
}
