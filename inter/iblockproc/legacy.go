// Package iblockproc defines the structures and logic for processing inter-block state.
// This specific file (legacy.go) defines legacy versions of epoch state structures (V0).
// These are typically preserved to maintain backward compatibility, allowing the node to
// read or migrate data from older database formats or earlier network versions.

package iblockproc

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"

	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/opera"
)

// ValidatorEpochStateV0 represents the state of a single validator within a specific epoch
// in the V0 legacy format. It tracks individual validator metrics that persist or reset across epochs.
type ValidatorEpochStateV0 struct {
	// GasRefund tracks the accumulated gas refund for the validator.
	// This is often used in incentive mechanisms where validators might be reimbursed for certain actions.
	GasRefund uint64

	// PrevEpochEvent is the hash of the last event emitted by this validator in the *previous* epoch.
	// This acts as a cryptographic link ensuring the continuity of the validator's event stream across epoch boundaries.
	PrevEpochEvent hash.Event
}

// EpochStateV0 represents the global state of the network for a specific epoch in the V0 legacy format.
// It contains all necessary information to validate transitions and manage the validator set for that epoch.
type EpochStateV0 struct {
	// Epoch is the unique sequential identifier (number) of the current epoch.
	Epoch idx.Epoch

	// EpochStart is the timestamp marking the official beginning of this epoch.
	EpochStart inter.Timestamp

	// PrevEpochStart is the timestamp when the immediately preceding epoch began.
	// This is useful for calculating epoch duration and time-based rewards.
	PrevEpochStart inter.Timestamp

	// EpochStateRoot is the Merkle root hash of the global state trie at the snapshot point of this epoch.
	// It ensures integrity of the state data associated with this epoch.
	EpochStateRoot hash.Hash

	// Validators is the set of validators (and their weights) active for this epoch.
	// This defines the committee responsible for consensus during this period.
	Validators *pos.Validators

	// ValidatorStates contains the specific state data (like gas refunds and previous events)
	// for each validator in the 'Validators' set.
	ValidatorStates []ValidatorEpochStateV0

	// ValidatorProfiles stores static or semi-static metadata about validators,
	// often used for driver-specific logic or reputation tracking.
	ValidatorProfiles ValidatorProfiles

	// Rules defines the protocol rules (forks, upgrades, parameters) that are active during this epoch.
	Rules opera.Rules
}
