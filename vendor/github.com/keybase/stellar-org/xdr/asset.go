package xdr

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/stellar/go/strkey"
)

// This file contains helpers for working with xdr.Asset structs

// AssetTypeToString maps an xdr.AssetType to its string representation
var AssetTypeToString = map[AssetType]string{
	AssetTypeAssetTypeNative:           "native",
	AssetTypeAssetTypeCreditAlphanum4:  "credit_alphanum4",
	AssetTypeAssetTypeCreditAlphanum12: "credit_alphanum12",
}

// StringToAssetType maps an strings to its xdr.AssetType representation
var StringToAssetType = map[string]AssetType{
	"native":            AssetTypeAssetTypeNative,
	"credit_alphanum4":  AssetTypeAssetTypeCreditAlphanum4,
	"credit_alphanum12": AssetTypeAssetTypeCreditAlphanum12,
}

// MustNewNativeAsset returns a new native asset, panicking if it can't.
func MustNewNativeAsset() Asset {
	a := Asset{}
	err := a.SetNative()
	if err != nil {
		panic(err)
	}
	return a
}

// MustNewCreditAsset returns a new general asset, panicking if it can't.
func MustNewCreditAsset(code string, issuer string) Asset {
	a, err := NewCreditAsset(code, issuer)
	if err != nil {
		panic(err)
	}
	return a
}

// NewCreditAsset returns a new general asset, returning an error if it can't.
func NewCreditAsset(code string, issuer string) (Asset, error) {
	a := Asset{}
	accountID := AccountId{}
	if err := accountID.SetAddress(issuer); err != nil {
		return Asset{}, err
	}
	if err := a.SetCredit(code, accountID); err != nil {
		return Asset{}, err
	}
	return a, nil
}

// BuildAsset creates a new asset from a given `assetType`, `code`, and `issuer`.
//
// Valid assetTypes are:
// 		- `native`
//		- `credit_alphanum4`
//		- `credit_alphanum12`
func BuildAsset(assetType, issuer, code string) (Asset, error) {
	t, ok := StringToAssetType[assetType]

	if !ok {
		return Asset{}, errors.New("invalid asset type: was not one of 'native', 'credit_alphanum4', 'credit_alphanum12'")
	}

	var asset Asset
	switch t {
	case AssetTypeAssetTypeNative:
		if err := asset.SetNative(); err != nil {
			return Asset{}, err
		}
	default:
		issuerAccountID := AccountId{}
		if err := issuerAccountID.SetAddress(issuer); err != nil {
			return Asset{}, err
		}

		if err := asset.SetCredit(code, issuerAccountID); err != nil {
			return Asset{}, err
		}
	}

	return asset, nil
}

var ValidAssetCode = regexp.MustCompile("^[[:alnum:]]{1,12}$")

// BuildAssets parses a list of assets from a given string.
// The string is expected to be a comma separated list of assets
// encoded in the format (Code:Issuer or "native") defined by SEP-0011
// https://github.com/stellar/stellar-protocol/pull/313
// If the string is empty, BuildAssets will return an empty list of assets
func BuildAssets(s string) ([]Asset, error) {
	var assets []Asset
	if s == "" {
		return assets, nil
	}

	assetStrings := strings.Split(s, ",")
	for _, assetString := range assetStrings {
		var asset Asset

		// Technically https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0011.md allows
		// any string up to 12 characters not containing an unescaped colon to represent XLM
		// however, this function only accepts the string "native" to represent XLM
		if strings.ToLower(assetString) == "native" {
			if err := asset.SetNative(); err != nil {
				return nil, err
			}
		} else {
			parts := strings.Split(assetString, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("%s is not a valid asset", assetString)
			}

			code := parts[0]
			if !ValidAssetCode.MatchString(code) {
				return nil, fmt.Errorf(
					"%s is not a valid asset, it contains an invalid asset code",
					assetString,
				)
			}

			issuer, err := AddressToAccountId(parts[1])
			if err != nil {
				return nil, fmt.Errorf(
					"%s is not a valid asset, it contains an invalid issuer",
					assetString,
				)
			}

			if err := asset.SetCredit(code, issuer); err != nil {
				return nil, fmt.Errorf("%s is not a valid asset", assetString)
			}
		}

		assets = append(assets, asset)
	}

	return assets, nil
}

// SetCredit overwrites `a` with a credit asset using `code` and `issuer`.  The
// asset type (CreditAlphanum4 or CreditAlphanum12) is chosen automatically
// based upon the length of `code`.
func (a *Asset) SetCredit(code string, issuer AccountId) error {
	length := len(code)
	var typ AssetType
	var body interface{}

	switch {
	case length >= 1 && length <= 4:
		newbody := AssetAlphaNum4{Issuer: issuer}
		copy(newbody.AssetCode[:], []byte(code)[:length])
		typ = AssetTypeAssetTypeCreditAlphanum4
		body = newbody
	case length >= 5 && length <= 12:
		newbody := AssetAlphaNum12{Issuer: issuer}
		copy(newbody.AssetCode[:], []byte(code)[:length])
		typ = AssetTypeAssetTypeCreditAlphanum12
		body = newbody
	default:
		return errors.New("Asset code length is invalid")
	}

	newa, err := NewAsset(typ, body)
	if err != nil {
		return err
	}
	*a = newa
	return nil
}

// SetNative overwrites `a` with the native asset type
func (a *Asset) SetNative() error {
	newa, err := NewAsset(AssetTypeAssetTypeNative, nil)
	if err != nil {
		return err
	}
	*a = newa
	return nil
}

// ToAllowTrustOpAsset for Asset converts the Asset to a corresponding XDR
// "allow trust" asset, used by the XDR allow trust operation.
func (a *Asset) ToAllowTrustOpAsset(code string) (AllowTrustOpAsset, error) {
	length := len(code)

	switch {
	case length >= 1 && length <= 4:
		var bytecode AssetCode4
		byteArray := []byte(code)
		copy(bytecode[:], byteArray[0:length])
		return NewAllowTrustOpAsset(AssetTypeAssetTypeCreditAlphanum4, bytecode)
	case length >= 5 && length <= 12:
		var bytecode AssetCode12
		byteArray := []byte(code)
		copy(bytecode[:], byteArray[0:length])
		return NewAllowTrustOpAsset(AssetTypeAssetTypeCreditAlphanum12, bytecode)
	default:
		return AllowTrustOpAsset{}, errors.New("Asset code length is invalid")
	}
}

// String returns a display friendly form of the asset
func (a Asset) String() string {
	var t, c, i string

	a.MustExtract(&t, &c, &i)

	if a.Type == AssetTypeAssetTypeNative {
		return t
	}

	return fmt.Sprintf("%s/%s/%s", t, c, i)
}

// MarshalBinaryCompress marshals Asset to []byte but unlike
// MarshalBinary() it removes all unnecessary bytes, exploting the fact
// that XDR is padding data to 4 bytes in union discriminants etc.
// It's primary use is in ingest/io.StateReader that keep LedgerKeys in
// memory so this function decrease memory requirements.
//
// Warning, do not use UnmarshalBinary() on data encoded using this method!
func (a Asset) MarshalBinaryCompress() ([]byte, error) {
	m := []byte{byte(a.Type)}

	var err error
	var code []byte
	var issuer []byte

	switch a.Type {
	case AssetTypeAssetTypeNative:
		return m, nil
	case AssetTypeAssetTypeCreditAlphanum4:
		code = []byte(strings.TrimRight(string(a.AlphaNum4.AssetCode[:]), "\x00"))
		issuer, err = a.AlphaNum4.Issuer.MarshalBinary()
		if err != nil {
			return nil, err
		}
	case AssetTypeAssetTypeCreditAlphanum12:
		code = []byte(strings.TrimRight(string(a.AlphaNum12.AssetCode[:]), "\x00"))
		issuer, err = a.AlphaNum12.Issuer.MarshalBinary()
		if err != nil {
			panic(err)
		}
	default:
		panic(fmt.Errorf("Unknown asset type: %v", a.Type))
	}

	m = append(m, code...)
	m = append(m, issuer...)
	return m, nil
}

// Equals returns true if `other` is equivalent to `a`
func (a Asset) Equals(other Asset) bool {
	if a.Type != other.Type {
		return false
	}
	switch a.Type {
	case AssetTypeAssetTypeNative:
		return true
	case AssetTypeAssetTypeCreditAlphanum4:
		l := a.MustAlphaNum4()
		r := other.MustAlphaNum4()
		return l.AssetCode == r.AssetCode && l.Issuer.Equals(r.Issuer)
	case AssetTypeAssetTypeCreditAlphanum12:
		l := a.MustAlphaNum12()
		r := other.MustAlphaNum12()
		return l.AssetCode == r.AssetCode && l.Issuer.Equals(r.Issuer)
	default:
		panic(fmt.Errorf("Unknown asset type: %v", a.Type))
	}
}

// Extract is a helper function to extract information from an xdr.Asset
// structure.  It extracts the asset's type to the `typ` input parameter (which
// must be either a *string or *xdr.AssetType).  It also extracts the asset's
// code and issuer to `code` and `issuer` respectively if they are of type
// *string and the asset is non-native
func (a Asset) Extract(typ interface{}, code interface{}, issuer interface{}) error {
	switch typ := typ.(type) {
	case *AssetType:
		*typ = a.Type
	case *string:
		*typ = AssetTypeToString[a.Type]
	default:
		return errors.New("can't extract type")
	}

	if code != nil {
		switch code := code.(type) {
		case *string:
			switch a.Type {
			case AssetTypeAssetTypeCreditAlphanum4:
				an := a.MustAlphaNum4()
				*code = strings.TrimRight(string(an.AssetCode[:]), "\x00")
			case AssetTypeAssetTypeCreditAlphanum12:
				an := a.MustAlphaNum12()
				*code = strings.TrimRight(string(an.AssetCode[:]), "\x00")
			}
		default:
			return errors.New("can't extract code")
		}
	}

	if issuer != nil {
		switch issuer := issuer.(type) {
		case *string:
			switch a.Type {
			case AssetTypeAssetTypeCreditAlphanum4:
				an := a.MustAlphaNum4()
				raw := an.Issuer.MustEd25519()
				*issuer = strkey.MustEncode(strkey.VersionByteAccountID, raw[:])
			case AssetTypeAssetTypeCreditAlphanum12:
				an := a.MustAlphaNum12()
				raw := an.Issuer.MustEd25519()
				*issuer = strkey.MustEncode(strkey.VersionByteAccountID, raw[:])
			}
		default:
			return errors.New("can't extract issuer")
		}
	}

	return nil
}

// MustExtract behaves as Extract, but panics if an error occurs.
func (a Asset) MustExtract(typ interface{}, code interface{}, issuer interface{}) {
	err := a.Extract(typ, code, issuer)

	if err != nil {
		panic(err)
	}
}
