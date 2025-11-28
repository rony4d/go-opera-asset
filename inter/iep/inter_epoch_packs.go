// Package iep (Inter-Epoch Packs) defines aggregate data structures for epoch processing.
// It acts as a container that bundles the full record of an epoch along with the
// necessary cryptographic proofs (votes) from validators attesting to its validity.

package iep

import (
	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/inter/ier"
)

// LlrEpochPack (Lachesis Light Repeater Epoch Pack) encapsulates all the data required to prove and reconstruct a finished epoch.
// It combines the content of the epoch (Record) with the consensus agreement (Votes).
type LlrEpochPack struct {
	// Votes is a collection of signed votes from validators.
	// These signatures collectively prove that the network reached consensus on the
	// data contained in the Record.
	Votes []inter.LlrSignedEpochVote

	// Record contains the full data of the epoch (e.g., index, state root, etc.),
	// along with its unique index.
	Record ier.LlrIdxFullEpochRecord
}
