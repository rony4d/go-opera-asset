// Package ier (Inter-Epoch Records) defines the data structures for recording epoch transitions.
// It captures the complete state of the network at the end of an epoch, including both
// the final block state and the finalized epoch state. This ensures that all necessary
// data for state verification and transition is bundled together.
package ier

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/rony4d/go-opera-asset/inter/iblockproc"
)

// LlrFullEpochRecord contains the complete state snapshots required to close an epoch.
// It aggregates the state of the last block in the epoch and the summarized state of the epoch itself.
type LlrFullEpochRecord struct {
	// BlockState is the state of the network after applying the very last block of the epoch.
	BlockState iblockproc.BlockState

	// EpochState is the finalized state summary for the entire epoch, containing
	// validator rewards, new validator sets, and other epoch-level metadata.
	EpochState iblockproc.EpochState
}

// LlrIdxFullEpochRecord wraps LlrFullEpochRecord with the specific epoch index.
// This associates the state record with its sequence number in the chain's history.
type LlrIdxFullEpochRecord struct {
	LlrFullEpochRecord
	// Idx is the unique sequential identifier (number) of the epoch this record belongs to.
	Idx idx.Epoch
}

// Hash calculates a deterministic hash of the full epoch record.
// It combines the hashes of the BlockState and EpochState.
// This hash serves as a unique fingerprint for the entire epoch's outcome and is likely
// what validators sign to reach consensus on the epoch transition.
func (er LlrFullEpochRecord) Hash() hash.Hash {
	return hash.Of(er.BlockState.Hash().Bytes(), er.EpochState.Hash().Bytes())
}
