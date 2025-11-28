// Package iblockproc defines the structures and logic for processing inter-block state.
// This file (decided_state.go) contains the core state definitions that the consensus engine
// maintains and transitions. It defines two main levels of state:
// 1. BlockState: State that changes with every decided block (e.g., validator uptime, gas power).
// 2. EpochState: State that is finalized at the end of an epoch (e.g., validator set changes, accumulated rewards).
// It also includes methods for hashing, copying, and accessing these states safely.
package iblockproc

import (
	"crypto/sha256"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/opera"
)

// ValidatorBlockState tracks the dynamic state of a validator that changes with every block.
// This includes uptime metrics and gas power tracking which are updated frequently.
type ValidatorBlockState struct {
	// LastEvent contains info about the last event confirmed from this validator.
	LastEvent EventInfo
	// Uptime is the total duration the validator has been online/active in the current epoch.
	Uptime inter.Timestamp
	// LastOnlineTime is the timestamp of the last proof of liveness.
	LastOnlineTime inter.Timestamp
	// LastGasPowerLeft tracks the remaining gas power allocation for the validator.
	// This is used for rate-limiting or throughput control.
	LastGasPowerLeft inter.GasPowerLeft
	// LastBlock is the index of the last block where this validator was active.
	LastBlock idx.Block
	// DirtyGasRefund is the gas refund accumulated in the current block/short-term.
	DirtyGasRefund uint64
	// Originated tracks the amount of gas/resources originated by this validator.
	Originated *big.Int
}

// EventInfo is a compact representation of an event, storing only what's needed for state tracking.
type EventInfo struct {
	ID           hash.Event
	GasPowerLeft inter.GasPowerLeft
	Time         inter.Timestamp
}

// ValidatorEpochState tracks validator information that is summarized at the epoch level.
// These values persist or are reset at epoch boundaries.
type ValidatorEpochState struct {
	// GasRefund is the total gas refund accumulated for the validator over the epoch.
	GasRefund uint64
	// PrevEpochEvent is the last event from the previous epoch, linking the event chains.
	PrevEpochEvent EventInfo
}

// BlockCtx contains metadata about a specific block.
type BlockCtx struct {
	Idx  idx.Block
	Time inter.Timestamp
	// Atropos is the hash of the event that finalized this block (the "deciding" event).
	Atropos hash.Event
}

// BlockState represents the global state of the chain at a specific block height.
// It aggregates all validator states, cheater information, and pending rule changes.
type BlockState struct {
	LastBlock          BlockCtx
	FinalizedStateRoot hash.Hash

	// EpochGas captures total gas used in the epoch so far.
	EpochGas uint64
	// EpochCheaters maintains a list of validators caught cheating in this epoch.
	EpochCheaters lachesis.Cheaters
	// CheatersWritten tracks how many cheaters have been written to the state (for efficient updates).
	CheatersWritten uint32

	// ValidatorStates holds the block-level state for each validator.
	ValidatorStates []ValidatorBlockState
	// NextValidatorProfiles contains the validator set that will become active in the *next* epoch.
	NextValidatorProfiles ValidatorProfiles

	// DirtyRules stores any rule changes (like network upgrades) pending for this epoch.
	// nil means no changes relative to the epoch start.
	DirtyRules *opera.Rules `rlp:"nil"`

	// AdvanceEpochs indicates if/how many epochs should be advanced.
	AdvanceEpochs idx.Epoch
}

// Copy creates a deep copy of the BlockState to ensure thread safety and prevent side effects
// when the state is modified in different contexts.
func (bs BlockState) Copy() BlockState {
	cp := bs
	// Deep copy slices
	cp.EpochCheaters = make(lachesis.Cheaters, len(bs.EpochCheaters))
	copy(cp.EpochCheaters, bs.EpochCheaters)
	cp.ValidatorStates = make([]ValidatorBlockState, len(bs.ValidatorStates))
	copy(cp.ValidatorStates, bs.ValidatorStates)
	// Deep copy big.Int pointers within struct slice
	for i := range cp.ValidatorStates {
		cp.ValidatorStates[i].Originated = new(big.Int).Set(cp.ValidatorStates[i].Originated)
	}
	// Deep copy maps/complex structures
	cp.NextValidatorProfiles = bs.NextValidatorProfiles.Copy()
	if bs.DirtyRules != nil {
		rules := bs.DirtyRules.Copy()
		cp.DirtyRules = &rules
	}
	return cp
}

// GetValidatorState returns a pointer to the block-level state of a specific validator by ID.
// It uses the validators collection to map the ID to an index.
func (bs *BlockState) GetValidatorState(id idx.ValidatorID, validators *pos.Validators) *ValidatorBlockState {
	validatorIdx := validators.GetIdx(id)
	return &bs.ValidatorStates[validatorIdx]
}

// Hash calculates the SHA256 hash of the RLP-encoded BlockState.
// This hash effectively fingerprints the entire consensus state at this block.
func (bs BlockState) Hash() hash.Hash {
	hasher := sha256.New()
	err := rlp.Encode(hasher, &bs)
	if err != nil {
		panic("can't hash: " + err.Error())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

// EpochStateV1 represents the state definition for an epoch in the current version.
type EpochStateV1 struct {
	Epoch          idx.Epoch
	EpochStart     inter.Timestamp
	PrevEpochStart inter.Timestamp

	EpochStateRoot hash.Hash

	Validators        *pos.Validators
	ValidatorStates   []ValidatorEpochState
	ValidatorProfiles ValidatorProfiles

	Rules opera.Rules
}

// EpochState is the current alias for EpochStateV1.
type EpochState EpochStateV1

// GetValidatorState returns a pointer to the epoch-level state of a specific validator by ID.
func (es *EpochState) GetValidatorState(id idx.ValidatorID, validators *pos.Validators) *ValidatorEpochState {
	validatorIdx := validators.GetIdx(id)
	return &es.ValidatorStates[validatorIdx]
}

// Duration calculates the length of the epoch in time units.
func (es EpochState) Duration() inter.Timestamp {
	return es.EpochStart - es.PrevEpochStart
}

// Hash calculates the hash of the EpochState.
// It handles backward compatibility: if the "London" upgrade is not active,
// it hashes the state using the V0 structure (legacy format) to ensure hash consistency across upgrades.
func (es EpochState) Hash() hash.Hash {
	var hashed interface{}
	if es.Rules.Upgrades.London {
		hashed = &es
	} else {
		// Convert to V0 structure for legacy hashing compatibility
		es0 := EpochStateV0{
			Epoch:             es.Epoch,
			EpochStart:        es.EpochStart,
			PrevEpochStart:    es.PrevEpochStart,
			EpochStateRoot:    es.EpochStateRoot,
			Validators:        es.Validators,
			ValidatorStates:   make([]ValidatorEpochStateV0, len(es.ValidatorStates)),
			ValidatorProfiles: es.ValidatorProfiles,
			Rules:             es.Rules,
		}
		// Map V1 fields back to V0 fields
		for i, v := range es.ValidatorStates {
			es0.ValidatorStates[i].GasRefund = v.GasRefund
			es0.ValidatorStates[i].PrevEpochEvent = v.PrevEpochEvent.ID
		}
		hashed = &es0
	}

	hasher := sha256.New()
	err := rlp.Encode(hasher, hashed)
	if err != nil {
		panic("can't hash: " + err.Error())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

// Copy creates a deep copy of the EpochState.
func (es EpochState) Copy() EpochState {
	cp := es
	cp.ValidatorStates = make([]ValidatorEpochState, len(es.ValidatorStates))
	copy(cp.ValidatorStates, es.ValidatorStates)
	cp.ValidatorProfiles = es.ValidatorProfiles.Copy()
	if es.Rules != (opera.Rules{}) {
		cp.Rules = es.Rules.Copy()
	}
	return cp
}
