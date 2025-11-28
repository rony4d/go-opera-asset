// Package drivertype defines the fundamental data structures and constants for representing validators
// within the consensus driver. It serves as a bridge between the consensus engine and the
// node implementation, defining how validator identities, weights, and statuses are structured.

package drivertype

import (
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/rony4d/go-opera-asset/inter/validatorpk"
)

var (
	// DoublesignBit is a bitmask flag used to mark a validator's status if they have
	// been caught double-signing (equivocating). This usually results in slashing or
	// eviction from the validator set.
	// 1 << 7 means the 8th bit is set (binary 10000000, decimal 128).
	DoublesignBit = uint64(1 << 7)

	// OkStatus represents the clean state of a validator with no adverse status bits set.
	OkStatus = uint64(0)
)

// Validator is the node-side representation of a consensus validator.
// It encapsulates the cryptographic identity and the protocol weight of a single participant.
type Validator struct {
	// Weight represents the voting power of the validator.
	// In Proof-of-Stake, this typically corresponds to the amount of staked tokens.
	Weight *big.Int

	// PubKey is the public key used to verify the digital signatures of events
	// and blocks created by this validator.
	PubKey validatorpk.PubKey
}

// ValidatorAndID is a convenience structure that pairs a validator's definition
// with their unique numeric index (ID). This is often used when iterating over
// lists of validators where both the ID and the full details are needed.
type ValidatorAndID struct {
	// ValidatorID is the unique numeric identifier (index) for the validator.
	// This is used for efficient lookups and bitmask operations in the consensus engine.
	ValidatorID idx.ValidatorID

	// Validator holds the detailed information (Weight, PubKey) for this ID.
	Validator Validator
}
