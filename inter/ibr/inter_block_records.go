// Package ibr (Inter-Block Records) defines data structures used for Lachesis Light Repeater (LLR)
// or similar inter-block voting/recording mechanisms. These structures capture essential block data
// required for consensus voting and verifying block integrity without necessarily needing the full block body at all stages.

package ibr

import (
	"github.com/Fantom-foundation/lachesis-base/common/bigendian"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rony4d/go-opera-asset/inter"
)

// LlrBlockVote represents a summary or "vote" for a specific block's content.
// Instead of containing the full transactions and receipts, it contains their hashes.
// This is lightweight and suitable for passing around in consensus votes where bandwidth is a concern.
type LlrBlockVote struct {
	// Atropos is the hash of the event that finalized the block (the "deciding" event).
	Atropos hash.Event
	// Root is the state root hash after applying the block's transactions.
	Root hash.Hash
	// TxHash is the Merkle root hash of the transactions included in the block.
	TxHash hash.Hash
	// ReceiptsHash is the Merkle root hash of the transaction receipts.
	ReceiptsHash hash.Hash
	// Time is the timestamp of the block.
	Time inter.Timestamp
	// GasUsed is the total gas consumed by transactions in this block.
	GasUsed uint64
}

// LlrFullBlockRecord contains the complete data for a block record.
// Unlike LlrBlockVote, this includes the actual list of transactions and receipts.
// This structure is likely used for storage or when full validation is required.
type LlrFullBlockRecord struct {
	Atropos  hash.Event
	Root     hash.Hash
	Txs      types.Transactions
	Receipts []*types.ReceiptForStorage
	Time     inter.Timestamp
	GasUsed  uint64
}

// LlrIdxFullBlockRecord wraps LlrFullBlockRecord with its sequential block index (number).
// This is useful when the record needs to be associated with its specific height in the chain.
type LlrIdxFullBlockRecord struct {
	LlrFullBlockRecord
	Idx idx.Block
}

// Hash calculates a deterministic hash of the LlrBlockVote.
// It combines all fields (Atropos, Root, TxHash, ReceiptsHash, Time, GasUsed) into a single hash.
// This hash identifies the specific combination of block data being voted on.
func (bv LlrBlockVote) Hash() hash.Hash {
	return hash.Of(
		bv.Atropos.Bytes(),
		bv.Root.Bytes(),
		bv.TxHash.Bytes(),
		bv.ReceiptsHash.Bytes(),
		bv.Time.Bytes(),
		bigendian.Uint64ToBytes(bv.GasUsed),
	)
}

// Hash calculates the hash of the LlrFullBlockRecord.
// It first reduces the full record to a lightweight LlrBlockVote by calculating
// the transaction and receipt root hashes, and then calls Hash() on that vote.
// This ensures that a full record and its corresponding vote produce the same hash.
func (br LlrFullBlockRecord) Hash() hash.Hash {
	return LlrBlockVote{
		Atropos:      br.Atropos,
		Root:         br.Root,
		TxHash:       inter.CalcTxHash(br.Txs),
		ReceiptsHash: inter.CalcReceiptsHash(br.Receipts),
		Time:         br.Time,
		GasUsed:      br.GasUsed,
	}.Hash()
}
