package inter

import (
	"crypto/sha256"

	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

/*
This file defines how "Votes" are extracted and verified separately from the rest of the Event.
1. What is LLR?
LLR stands for Lachesis Light Repeater. In this consensus protocol (likely a variant of Lachesis or Opera), consensus happens in two layers:
DAG Layer: Fast, asynchronous ordering of events.
Block/Epoch Layer (LLR): A slower, heavier process where validators vote to "finalize" blocks and seal epochs.
This file handles the data structures for this second layer.


2. Embedding Votes in Events
Validators don't send separate "Vote Messages". Instead, they piggyback their votes inside the standard DAG Events they create.
An Event contains: { Transactions, MisbehaviourProofs, EpochVote, BlockVotes }.
This file allows us to slice out just the BlockVotes (or EpochVote) and pass them around as a self-contained proof.


3. The "Partial Hash" Verification Trick
The most important logic here is in CalcPayloadHash (lines 84-90).
The Goal: You receive a LlrSignedBlockVotes object. You want to know: "Did Validator X really sign these votes?"
The Problem: Validator X signed the EventLocator. The EventLocator contains the PayloadHash. The PayloadHash is a hash of everything (Txs + MPs + EpochVote + BlockVotes). But you only have the BlockVotes. You don't have the Txs.
The Solution (Merkle Proof-like): The struct LlrSignedBlockVotes includes the hashes of the missing parts (TxsAndMisbehaviourProofsHash and EpochVoteHash).
You compute Hash(YourBlockVotes).
You combine it with the provided sibling hashes.
You get a resulting CalculatedPayloadHash.
You check if CalculatedPayloadHash == Signed.Locator.PayloadHash.
If they match, the signature is valid for your votes, even though you haven't seen the transactions.


4. Batching
Notice LlrBlockVotes uses []hash.Hash for votes. This is an optimization.
Instead of creating one event per block vote, a validator waits and says "I vote for blocks 100-105 all at once."
This significantly reduces network overhead.


5. Hashing Order (CRITICAL)
When porting CalcPayloadHash, you must match the exact tree structure from event.go:
Root = Hash(    Hash(Txs, MPs),    Hash(EpochVote, BlockVotes))
In LlrSignedBlockVotes: You have BlockVotes. You are given Hash(EpochVote). You combine them as Hash(GivenEpochHash, MyBlockVotesHash).
In LlrSignedEpochVote: You have EpochVote. You are given Hash(BlockVotes). You combine them as Hash(MyEpochVoteHash, GivenBlockVotesHash).
If you swap the order in the hash function, the resulting hash will differ, and signature verification will fail.
*/

// LLR (Long-Lasting Round) votes are used to reach consensus on large-scale events
// like confirming blocks or sealing epochs. They are carried inside standard DAG events.

// LlrBlockVotes represents a batch of votes for a sequence of blocks.
// Instead of sending one message per block vote, validators batch them for efficiency.
// Example: "I confirm blocks 100 through 105 have hashes A, B, C, D, E, F".
type LlrBlockVotes struct {
	Start idx.Block   // The index of the first block in this batch
	Epoch idx.Epoch   // The epoch these blocks belong to
	Votes []hash.Hash // The proposed hashes for the blocks [Start, Start+1, ..., Start+N]
}

// LastBlock calculates the index of the final block voted for in this batch.
func (bvs LlrBlockVotes) LastBlock() idx.Block {
	if len(bvs.Votes) == 0 {
		return bvs.Start - 1
	}
	// Formula: Start + Length - 1
	return bvs.Start + idx.Block(len(bvs.Votes)) - 1
}

// LlrEpochVote represents a vote to seal an epoch.
// This is used to agree on the final state of an epoch before moving to the next one.
type LlrEpochVote struct {
	Epoch idx.Epoch // The epoch being voted on
	Vote  hash.Hash // The hash digest of the epoch's final state (usually Merkle root)
}

// LlrSignedBlockVotes is a wrapper that proves WHO cast the block votes.
// Since the votes are just data inside an Event, we need to link them back
// to the Event's signature to prove authenticity.
//
// It reconstructs the necessary parts of the Event Payload to verify the signature.
type LlrSignedBlockVotes struct {
	// Signed contains the event header (Locator) and the validator's Signature.
	Signed SignedEventLocator

	// TxsAndMisbehaviourProofsHash is a partial hash of the event payload.
	// Since we only have the BlockVotes part here, we need the hash of the *other* parts
	// (Txs, Proofs) to reconstruct the full PayloadHash and verify the signature.
	TxsAndMisbehaviourProofsHash hash.Hash

	// EpochVoteHash is the hash of the EpochVote part of the payload.
	// Required for the same reason as above: to reconstruct the full PayloadHash.
	EpochVoteHash hash.Hash

	// Val is the actual block votes being signed.
	Val LlrBlockVotes
}

// LlrSignedEpochVote is the epoch-equivalent of LlrSignedBlockVotes.
// It wraps an epoch vote with the proof of signature.
type LlrSignedEpochVote struct {
	Signed SignedEventLocator

	// TxsAndMisbehaviourProofsHash is the hash of the Txs/Proofs section.
	TxsAndMisbehaviourProofsHash hash.Hash

	// BlockVotesHash is the hash of the BlockVotes section.
	// Needed to reconstruct the full PayloadHash.
	BlockVotesHash hash.Hash

	// Val is the actual epoch vote being signed.
	Val LlrEpochVote
}

// AsSignedBlockVotes extracts a signed block vote package from a full event.
// This is useful when you want to forward just the votes to another peer
// without sending the entire event body (transactions, etc.).
func AsSignedBlockVotes(e EventPayloadI) LlrSignedBlockVotes {
	// We calculate the hash of Txs and MisbehaviourProofs combined.
	// This "summary" allows us to verify the payload hash later without needing the full Txs list.
	txsAndMps := hash.Of(
		CalcTxHash(e.Txs()).Bytes(),
		CalcMisbehaviourProofsHash(e.MisbehaviourProofs()).Bytes(),
	)

	return LlrSignedBlockVotes{
		Signed:                       AsSignedEventLocator(e),
		TxsAndMisbehaviourProofsHash: txsAndMps,
		EpochVoteHash:                e.EpochVote().Hash(), // Extract hash of the sibling data
		Val:                          e.BlockVotes(),       // Extract the data itself
	}
}

// AsSignedEpochVote extracts a signed epoch vote package from a full event.
func AsSignedEpochVote(e EventPayloadI) LlrSignedEpochVote {
	txsAndMps := hash.Of(
		CalcTxHash(e.Txs()).Bytes(),
		CalcMisbehaviourProofsHash(e.MisbehaviourProofs()).Bytes(),
	)

	return LlrSignedEpochVote{
		Signed:                       AsSignedEventLocator(e),
		TxsAndMisbehaviourProofsHash: txsAndMps,
		BlockVotesHash:               e.BlockVotes().Hash(), // Extract hash of the sibling data
		Val:                          e.EpochVote(),         // Extract the data itself
	}
}

// Size returns an estimated size in bytes for the signed locator.
// Used for bandwidth/storage estimation.
func (r SignedEventLocator) Size() uint64 {
	// Signature (65 bytes usually) + 3 hashes (32 bytes) + 4 integers (4 bytes) approx
	// Precise calc: Sig + BaseHash + PayloadHash + ...
	return uint64(len(r.Sig)) + 3*32 + 4*4
}

// Size returns estimated size for the signed block votes package.
func (bvs LlrSignedBlockVotes) Size() uint64 {
	// Signed Header + (Number of Votes * 32 bytes) + 2 Hash overheads + overheads
	return bvs.Signed.Size() + uint64(len(bvs.Val.Votes))*32 + 32*2 + 8 + 4
}

// Hash computes the canonical hash of an Epoch Vote.
// This hash is what gets included in the Event's payload merkle tree.
func (ers LlrEpochVote) Hash() hash.Hash {
	hasher := sha256.New()
	hasher.Write(ers.Epoch.Bytes())
	hasher.Write(ers.Vote.Bytes())
	return hash.BytesToHash(hasher.Sum(nil))
}

// Hash computes the canonical hash of a batch of Block Votes.
func (bvs LlrBlockVotes) Hash() hash.Hash {
	hasher := sha256.New()
	hasher.Write(bvs.Start.Bytes())
	hasher.Write(bvs.Epoch.Bytes())
	// Write length to prevent extension attacks
	hasher.Write(bigendian.Uint32ToBytes(uint32(len(bvs.Votes))))
	for _, bv := range bvs.Votes {
		hasher.Write(bv.Bytes())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

// CalcPayloadHash reconstructs the full Event PayloadHash from the partial components.
//
// An Event PayloadHash is: Hash( Hash(Txs, MPs), Hash(EpochVote, BlockVotes) )
// Here we have:
// 1. TxsAndMisbehaviourProofsHash = Hash(Txs, MPs)  [Provided]
// 2. EpochVoteHash                  [Provided]
// 3. Val                            [We have the data, so we Hash it ourselves]
//
// This allows us to check if `bvs.Signed.Locator.PayloadHash` matches the data we hold,
// ensuring the validator actually signed THESE votes.
func (bvs LlrSignedBlockVotes) CalcPayloadHash() hash.Hash {
	// Inner Hash 2: Hash(EpochVoteHash, Hash(Val))
	votesSubHash := hash.Of(bvs.EpochVoteHash.Bytes(), bvs.Val.Hash().Bytes())

	// Outer Hash: Hash(Inner Hash 1, Inner Hash 2)
	return hash.Of(bvs.TxsAndMisbehaviourProofsHash.Bytes(), votesSubHash.Bytes())
}

// CalcPayloadHash reconstructs the full Event PayloadHash for verification (Epoch version).
func (ev LlrSignedEpochVote) CalcPayloadHash() hash.Hash {
	// Inner Hash 2: Hash(Hash(Val), BlockVotesHash)
	// Note: Order matters! It must match the structure in event.go CalcPayloadHash.
	// In event.go: hash.Of(EpochVote.Hash(), BlockVotes.Hash())
	votesSubHash := hash.Of(ev.Val.Hash().Bytes(), ev.BlockVotesHash.Bytes())

	// Outer Hash
	return hash.Of(ev.TxsAndMisbehaviourProofsHash.Bytes(), votesSubHash.Bytes())
}

// Size returns estimated size for the signed epoch vote package.
func (ev LlrSignedEpochVote) Size() uint64 {
	return ev.Signed.Size() + 32 + 32*2 + 4 + 4
}
