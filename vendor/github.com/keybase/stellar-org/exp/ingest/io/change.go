package io

import (
	"bytes"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Change is a developer friendly representation of LedgerEntryChanges.
// It also provides some helper functions to quickly check if a given
// change has occured in an entry.
//
// If an entry is created: Pre is nil and Post is not nil.
// If an entry is updated: Pre is not nil and Post is not nil.
// If an entry is removed: Pre is not nil and Post is nil.
type Change struct {
	Type xdr.LedgerEntryType
	Pre  *xdr.LedgerEntry
	Post *xdr.LedgerEntry
}

// LedgerEntryChangeType returns type in terms of LedgerEntryChangeType.
func (c *Change) LedgerEntryChangeType() xdr.LedgerEntryChangeType {
	switch {
	case c.Pre == nil && c.Post != nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryCreated
	case c.Pre != nil && c.Post == nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryRemoved
	case c.Pre != nil && c.Post != nil:
		return xdr.LedgerEntryChangeTypeLedgerEntryUpdated
	default:
		panic("Invalid state of Change (Pre == nil && Post == nil)")
	}
}

// AccountChangedExceptSigners returns true if account has changed WITHOUT
// checking the signers (except master key weight!). In other words, if the only
// change is connected to signers, this function will return false.
func (c *Change) AccountChangedExceptSigners() (bool, error) {
	if c.Type != xdr.LedgerEntryTypeAccount {
		panic("This should not be called on changes other than Account changes")
	}

	// New account
	if c.Pre == nil {
		return true, nil
	}

	// Account merged
	// c.Pre != nil at this point.
	if c.Post == nil {
		return true, nil
	}

	// c.Pre != nil && c.Post != nil at this point.
	if c.Pre.LastModifiedLedgerSeq != c.Post.LastModifiedLedgerSeq {
		return true, nil
	}

	// Don't use short assignment statement (:=) to ensure variables below
	// are not pointers (if `xdr` package changes in the future)!
	var preAccountEntry, postAccountEntry xdr.AccountEntry
	preAccountEntry = c.Pre.Data.MustAccount()
	postAccountEntry = c.Post.Data.MustAccount()

	// preAccountEntry and postAccountEntry are copies so it's fine to
	// modify them here, EXCEPT pointers inside them!
	if preAccountEntry.Ext.V == 0 {
		preAccountEntry.Ext.V = 1
		preAccountEntry.Ext.V1 = &xdr.AccountEntryV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	preAccountEntry.Signers = nil

	if postAccountEntry.Ext.V == 0 {
		postAccountEntry.Ext.V = 1
		postAccountEntry.Ext.V1 = &xdr.AccountEntryV1{
			Liabilities: xdr.Liabilities{
				Buying:  0,
				Selling: 0,
			},
		}
	}

	postAccountEntry.Signers = nil

	preBinary, err := preAccountEntry.MarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "Error running preAccountEntry.MarshalBinary")
	}

	postBinary, err := postAccountEntry.MarshalBinary()
	if err != nil {
		return false, errors.Wrap(err, "Error running postAccountEntry.MarshalBinary")
	}

	return !bytes.Equal(preBinary, postBinary), nil
}

// AccountSignersChanged returns true if account signers have changed.
// Notice: this will return true on master key changes too!
func (c *Change) AccountSignersChanged() bool {
	if c.Type != xdr.LedgerEntryTypeAccount {
		panic("This should not be called on changes other than Account changes")
	}

	// New account so new master key (which is also a signer)
	if c.Pre == nil {
		return true
	}

	// Account merged. Account being merge can still have signers.
	// c.Pre != nil at this point.
	if c.Post == nil {
		return true
	}

	// c.Pre != nil && c.Post != nil at this point.
	preAccountEntry := c.Pre.Data.MustAccount()
	postAccountEntry := c.Post.Data.MustAccount()

	preSigners := preAccountEntry.SignerSummary()
	postSigners := postAccountEntry.SignerSummary()

	if len(preSigners) != len(postSigners) {
		return true
	}

	for postSigner, postWeight := range postSigners {
		preWeight, exist := preSigners[postSigner]
		if !exist {
			return true
		}

		if preWeight != postWeight {
			return true
		}
	}

	return false
}
