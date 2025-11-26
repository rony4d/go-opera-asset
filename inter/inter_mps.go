package inter

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

/*
	This file, inter_mps.go (likely short for Interface Misbehaviour Proofs), defines the data structures used to penalize bad actors in the network.
	1. The Philosophy of "Proofs"

	In this consensus engine, you cannot simply accuse someone of cheating; you must provide cryptographic proof.
	"Cryptographic proofs are mathematical algorithms and protocols that use advanced cryptography to validate a piece of information or a claim while maintaining privacy and security.
	Essentially, they allow one party (the Prover) to demonstrate the truth of a statement to another party (the Verifier) in a way that
	is computationally infeasible to fake or tamper with, often without revealing the underlying sensitive data."

	A proof in this context of Misbehaviour Proofs typically consists of Signed Messages that contradict each other or contradict the finalized chain.

	2. Types of Misbehaviour
	There are two main categories of misbehaviour handled here:

	A. Double Signing (Equivocation)
	This is when a validator says two different things at the same time. It is essentially "lying" or being "two-faced".
	EventsDoublesign: The validator released two DAG events with the same sequence number. This forks the DAG and attacks the ordering protocol.
	BlockVoteDoublesign: The validator voted "Hash A" for Block 10 and also voted "Hash B" for Block 10.
	EpochVoteDoublesign: The validator voted to seal Epoch 5 with "Hash X" and also with "Hash Y".

	B. Wrong Voting (Contradicting Consensus)
	This is when a validator votes for something that is objectively false according to the rest of the network (e.g., voting for a block that was never proposed or fails validation).
	WrongBlockVote: Voting for a bad block.
	WrongEpochVote: Voting for a bad epoch data.


	3. The "Accomplice" Rule (MinAccomplicesForProof)
	This is a unique feature of this protocol explained in the WrongBlockVote comments.
	Problem: If a validator's computer has a bit-flip in RAM, it might sign a random garbage hash. If we slash them immediately, we punish honest hardware failures.
	Solution: We only punish "Wrong Votes" if two or more validators sign the same wrong value. It is statistically impossible for two independent hardware failures to produce the exact same random garbage hash. Therefore, if two nodes sign the same wrong hash, they are running modified software (colluding).


	4. Struct Structure
	Pair [2]...: Used for double-signs. It always holds exactly two items: Evidence A and Evidence B.
	Pals [MinAccomplicesForProof]...: Used for wrong-votes. "Pals" implies the accomplices. It holds the array of signatures proving the collusion.

	5. Helper Functions (GetVote)
	The vote structures (likely LlrSignedBlockVotes) often contain a batch of votes (e.g., "I vote for blocks 100 to 110").
	The GetVote(i int) function is a utility to index into that batch and pull out the specific hash for the block being disputed (p.Block).
	Formula: index = target_block - start_block_of_batch

	6. The MisbehaviourProof Container
	This is a "Union" struct. In Go RLP (Recursive Length Prefix) serialization, pointers with the tag `` rlp:"nil" `` indicate optional fields.
	When this struct is sent over the network, only one of the 5 fields will be populated.
	When porting to a language with proper Enum/Union types (like Rust or TypeScript), you would likely represent this as an Enum with variants rather than a struct with optional pointers.
*/

// Constants related to proof validation.

// MinAccomplicesForProof defines the threshold for proving a "Wrong Vote".
// In distributed systems, a single validator might cast a wrong vote due to
// hardware failure, cosmic rays, or software bugs (non-malicious).
//
// To prevent slashing honest nodes for accidental faults, the protocol requires
// at least 2 validators (the culprit + 1 accomplice) to sign the same invalid
// vote to consider it a coordinated attack or significant protocol violation.
const (
	MinAccomplicesForProof = 2
)

// EventsDoublesign proves that a validator created two different events
// at the same logical height (Epoch + Lamport + Seq).
// This is a classic "equivocation" or "forking" attack in DAG-based consensus.
type EventsDoublesign struct {
	// Pair contains the headers (locators) and signatures of the two conflicting events.
	// Both events must be from the same Creator and have the same Seq/Epoch.
	Pair [2]SignedEventLocator
}

// BlockVoteDoublesign proves that a validator cast two contradictory votes
// for the same block index.
// Example: Voting "Yes" for Block 100 and later voting "No" (or a different hash) for Block 100.
type BlockVoteDoublesign struct {
	// Block is the index of the block being voted on.
	Block idx.Block
	// Pair contains the two signed vote packages containing the conflicting votes.
	Pair [2]LlrSignedBlockVotes
}

// GetVote is a helper to extract the specific vote hash for the disputed block
// from the batch of votes in the proof.
func (p BlockVoteDoublesign) GetVote(i int) hash.Hash {
	// The vote package (LlrSignedBlockVotes) contains a range of votes.
	// We calculate the offset: (Target Block - Start Block of the batch).
	return p.Pair[i].Val.Votes[p.Block-p.Pair[i].Val.Start]
}

// WrongBlockVote proves that a validator voted for a block that contradicts
// the canonical chain (e.g., voting for a block hash that doesn't exist or
// conflicts with finality).
//
// Unlike doublesigning (which is self-contradiction), this is contradicting reality.
// It requires 'MinAccomplicesForProof' signatures to be valid (see constant doc).
type WrongBlockVote struct {
	// Block is the index of the invalid block vote.
	Block idx.Block
	// Pals (Accomplices) are the signed vote packages from the validators involved.
	// Pals[0] is usually the primary target, and Pals[1:] are the accomplices.
	Pals [MinAccomplicesForProof]LlrSignedBlockVotes
	// WrongEpoch indicates if the vote was for the wrong epoch context entirely.
	WrongEpoch bool
}

// GetVote extracts the specific invalid hash voted for by the i-th accomplice.
func (p WrongBlockVote) GetVote(i int) hash.Hash {
	// Calculate offset in the vote batch to find the specific vote hash.
	return p.Pals[i].Val.Votes[p.Block-p.Pals[i].Val.Start]
}

// EpochVoteDoublesign proves that a validator cast two contradictory votes
// regarding the sealing of an epoch.
// Similar to BlockVoteDoublesign but for the higher-level Epoch structure.
type EpochVoteDoublesign struct {
	// Pair contains the two conflicting signed epoch votes.
	Pair [2]LlrSignedEpochVote
}

// WrongEpochVote proves that a validator voted for an epoch sealing that
// contradicts the canonical history (e.g., wrong root hash for the epoch).
// Like WrongBlockVote, this requires accomplices to prove it wasn't a glitch.
type WrongEpochVote struct {
	// Pals are the signed votes from the validators involved (culprit + accomplice).
	Pals [MinAccomplicesForProof]LlrSignedEpochVote
}

// MisbehaviourProof is a union container (sum type) that holds exactly one
// specific type of proof.
//
// When serializing/deserializing (RLP), pointers are used to make fields optional.
// Only one field should be non-nil.
type MisbehaviourProof struct {
	// 1. Event Equivocation (Forking the DAG)
	EventsDoublesign *EventsDoublesign `rlp:"nil"`

	// 2. Block Equivocation (Conflicting votes for a block)
	BlockVoteDoublesign *BlockVoteDoublesign `rlp:"nil"`

	// 3. Invalid Block Vote (Voting against consensus)
	WrongBlockVote *WrongBlockVote `rlp:"nil"`

	// 4. Epoch Equivocation (Conflicting votes for an epoch)
	EpochVoteDoublesign *EpochVoteDoublesign `rlp:"nil"`

	// 5. Invalid Epoch Vote (Voting against consensus epoch)
	WrongEpochVote *WrongEpochVote `rlp:"nil"`
}
