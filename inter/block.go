// Package inter defines Opera's core consensus data structures that bridge
// Lachesis DAG consensus with Ethereum-compatible execution. This file
// contains the Block structure, which represents a finalized block produced
// by Opera's consensus algorithm.
//
// Key concepts:
//   - Block: A finalized block containing events, transactions, and state root
//   - Atropos: The consensus event hash that determines block finality
//   - Events: List of event hashes included in this block
//   - SkippedTxs: Indexes of transactions that were invalid/rejected
//
// Usage:
//   block := inter.Block{
//       Time: timestamp,
//       Atropos: consensusEventHash,
//       Events: eventHashes,
//       Root: stateRoot,
//   }
//   size := block.EstimateSize()
//   filtered := FilterSkippedTxs(allTxs, block.SkippedTxs)
//
// The Block structure is produced by Opera's consensus engine (Lachesis) and
// contains all information needed to execute transactions and update state.

package inter

import (
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Block represents a finalized block in Opera's consensus model. Unlike
// Ethereum blocks which are produced by miners, Opera blocks are produced
// by the Lachesis consensus algorithm based on a DAG (Directed Acyclic Graph)
// of events from validators.
//
// The block contains:
//   - Consensus metadata (Time, Atropos, Events)
//   - Transaction references (Txs, InternalTxs, SkippedTxs)
//   - Execution results (GasUsed, Root)
//
// This structure is the output of consensus and serves as input to the EVM
// execution layer. It's converted to EvmBlock format for EVM compatibility.

type Block struct {
	// Time is the timestamp when this block was finalized by consensus.
	// This timestamp is derived from the median time of validator events
	// included in the block, ensuring Byzantine fault tolerance.
	Time Timestamp

	// Atropos is the hash of the consensus event that determines this block's
	// finality. In Lachesis terminology, "Atropos" refers to the event that
	// establishes the block's position in the DAG. This hash becomes the
	// block's unique identifier and is used as the block hash in EvmHeader.
	Atropos hash.Event

	// Events is a list of event hashes from validators that are included in
	// this block. Each event represents a validator's contribution to consensus
	// (containing transactions, votes, etc.). The block aggregates multiple
	// events to form a complete block.
	Events hash.Events

	// Txs contains hashes of transactions that are not embedded in events.
	// These transactions come from two sources:
	//   1. Genesis transactions (pre-deployed contracts, initial allocations)
	//   2. LLR (Low Latency Records) transactions (fast-tracked transactions
	//      that bypass the normal event-based flow)
	//
	// Note: The actual transaction data is stored separately; this field
	// only contains references (hashes) to locate the transactions.
	Txs []common.Hash

	// InternalTxs contains hashes of internal transactions (contract-to-contract
	// calls, self-destructs, etc.). This field is DEPRECATED and should not
	// be used in new code. Use Txs field with internal.IsInternal() method
	// to distinguish internal transactions instead.
	//
	// DEPRECATED: Use Txs field with internal.IsInternal() method
	InternalTxs []common.Hash

	// SkippedTxs contains zero-indexed positions of transactions that were
	// skipped (rejected) during block processing. The indexes reference
	// transactions in the order they appear when all events are flattened:
	// starting from the first transaction of the first event, through the
	// last transaction of the last event.
	//
	// Example: If SkippedTxs = [2, 5], it means the 3rd and 6th transactions
	// (0-indexed) were skipped due to validation failures (invalid signature,
	// insufficient gas, etc.).
	//
	// This allows the execution layer to filter out invalid transactions
	// without re-processing the entire block.
	SkippedTxs []uint32

	// GasUsed is the total amount of gas consumed by all transactions executed
	// in this block. This includes gas for successful transactions and gas
	// consumed by failed transactions (up to the point of failure).
	GasUsed uint64

	// Root is the Merkle root hash of the state trie after executing all
	// transactions in this block. This represents the complete state of all
	// accounts, contract storage, and balances after block execution.
	//
	// The state root is computed by the EVM execution layer and serves as
	// a commitment to the entire state, allowing efficient state verification
	// without storing the full state data.
	Root hash.Hash
}

// EstimateSize returns an approximate size estimate of the block in bytes.
// This is used for memory management, network transfer size estimation, and
// database storage planning.
//
// Returns:
//   - Estimated size in bytes
//
// The estimate includes:
//   - Event hashes: len(Events) * 32 bytes (each hash is 32 bytes)
//   - InternalTxs hashes: len(InternalTxs) * 32 bytes (deprecated but counted)
//   - Txs hashes: len(Txs) * 32 bytes
//   - Atropos hash: 1 * 32 bytes
//   - Root hash: 1 * 32 bytes
//   - SkippedTxs indexes: len(SkippedTxs) * 4 bytes (each uint32 is 4 bytes)
//   - GasUsed: 8 bytes (uint64)
//   - Time: 8 bytes (Timestamp is uint64 internally)
//
// This is a rough estimate and may not match the actual serialized size
// exactly due to RLP encoding overhead, but it's accurate enough for
// memory allocation and network planning purposes.
func (b *Block) EstimateSize() int {
	// Calculate hash storage: Events + InternalTxs + Txs + Atropos + Root
	// Each hash is 32 bytes
	hashCount := len(b.Events) + len(b.InternalTxs) + len(b.Txs) + 1 + 1
	hashBytes := hashCount * 32

	// Calculate SkippedTxs storage: each uint32 index is 4 bytes
	skippedBytes := len(b.SkippedTxs) * 4

	// Calculate fixed-size fields: GasUsed (8 bytes) + Time (8 bytes)
	fixedBytes := 8 + 8

	return hashBytes + skippedBytes + fixedBytes
}

// FilterSkippedTxs removes transactions from a list based on skip indexes.
// This function is used during block execution to filter out transactions
// that were marked as invalid during consensus validation.
//
// Parameters:
//   - txs: Complete list of transactions (including ones to be skipped)
//   - skippedTxs: Zero-indexed positions of transactions to remove
//
// Returns:
//   - Filtered transaction list with skipped transactions removed
//
// The skippedTxs array must be sorted in ascending order for correct behavior.
// The function assumes transactions are indexed starting from 0, where index 0
// is the first transaction of the first event, and the last index is the
// last transaction of the last event.
//
// Example:
//
//	allTxs := [tx0, tx1, tx2, tx3, tx4]
//	skipped := [1, 3]
//	filtered := FilterSkippedTxs(allTxs, skipped)
//	// Result: [tx0, tx2, tx4]
//
// Performance: O(n) where n is the number of transactions. The function
// short-circuits if skippedTxs is empty to avoid unnecessary allocation.
func FilterSkippedTxs(txs types.Transactions, skippedTxs []uint32) types.Transactions {
	// Short-circuit optimization: if no transactions are skipped, return
	// the original list without allocation
	if len(skippedTxs) == 0 {
		return txs
	}

	// Track current position in skippedTxs array
	skipCount := 0

	// Pre-allocate filtered list with capacity equal to original size
	// (worst case: no skips, best case: some skips reduce size)
	filteredTxs := make(types.Transactions, 0, len(txs))

	// Iterate through all transactions
	for i, tx := range txs {
		// Check if current transaction index matches next skip index
		// The skippedTxs array is sorted, so we can process it sequentially
		if skipCount < len(skippedTxs) && skippedTxs[skipCount] == uint32(i) {
			// This transaction should be skipped: advance skip counter
			// and don't add it to filtered list
			skipCount++
		} else {
			// This transaction should be included: add to filtered list
			filteredTxs = append(filteredTxs, tx)
		}
	}

	return filteredTxs
}
