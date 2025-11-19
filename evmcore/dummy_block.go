// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package evmcore provides adapters between Opera's consensus block format
// (Lachesis DAG events/blocks) and Ethereum's EVM-compatible block format.
// This file implements "dummy" block structures that bridge the gap between
// Opera's event-based consensus and Ethereum's block-based execution model.
//
// Key concepts:
//   - EvmHeader/EvmBlock: Opera's EVM-compatible block representation
//   - Conversion functions: Transform between Opera blocks and Ethereum blocks
//   - Gas limits: Opera uses MaxUint64 (unlimited) since gas is managed per-event
//
// Usage:
//   operaBlock := inter.Block{...}
//   evmHeader := ToEvmHeader(&operaBlock, blockIndex, prevHash, rules)
//   ethBlock := evmHeader.EthHeader() // convert to Ethereum format for EVM execution
//
// The "dummy" name refers to the fact that these blocks don't follow Ethereum's
// proof-of-work model (no difficulty, no mining). Instead, they're produced by
// Opera's Lachesis consensus and converted to EVM format for execution.

package evmcore

import (
	"math"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/opera"
)

// EvmHeader represents an EVM-compatible block header in Opera's format.
// It contains the essential fields needed for EVM execution while maintaining
// compatibility with Opera's consensus model (Lachesis DAG).
//
// Key differences from Ethereum headers:
//   - GasLimit is MaxUint64 (unlimited) because Opera manages gas per-event, not per-block
//   - Hash is derived from Opera's consensus event hash (Atropos), not PoW
//   - Time uses Opera's Timestamp type (higher precision than Ethereum's uint64)
//   - BaseFee is optional (only set if London upgrade is active)
type EvmHeader struct {
	Number     *big.Int        // Block number (height in the chain)
	Hash       common.Hash     // Block hash (derived from Opera's consensus event hash)
	ParentHash common.Hash     // Hash of the parent block
	Root       common.Hash     // State root (Merkle root of account/storage state)
	TxHash     common.Hash     // Transactions root (Merkle root of transaction trie)
	Time       inter.Timestamp // Block timestamp (Opera's high-precision timestamp)
	Coinbase   common.Address  // Validator address that produced this block

	GasLimit uint64 // Gas limit per block (always MaxUint64 in Opera - unlimited)
	GasUsed  uint64 // Total gas consumed by transactions in this block

	BaseFee *big.Int // Base fee per gas (EIP-1559, nil if London upgrade not active)
}

// EvmBlock represents a complete EVM-compatible block containing a header
// and list of transactions. This structure bridges Opera's consensus blocks
// with Ethereum's block format for EVM execution.
type EvmBlock struct {
	EvmHeader                       // Embedded header (contains block metadata)
	Transactions types.Transactions // List of transactions included in this block
}

// NewEvmBlock constructs a new EvmBlock from a header and transaction list.
// It automatically computes the transaction root hash (TxHash) using a Merkle trie.
//
// Parameters:
//   - h: Block header (will be copied into the new block)
//   - txs: List of transactions to include in the block
//
// Returns:
//   - Pointer to a new EvmBlock with TxHash computed
//
// The TxHash is set to EmptyRootHash if there are no transactions, otherwise
// it's computed using Ethereum's DeriveSha function (Merkle trie root).

func NewEvmBlock(h *EvmHeader, txs types.Transactions) *EvmBlock {
	b := &EvmBlock{
		EvmHeader:    *h,  // copy header struct
		Transactions: txs, // store transaction list
	}

	// Compute transaction root hash
	if len(txs) == 0 {
		// Empty block: use empty root hash (standard Ethereum convention)
		b.EvmHeader.TxHash = types.EmptyRootHash
	} else {
		// Non-empty block: compute Merkle root of transaction trie
		// StackTrie is a memory-efficient trie implementation for one-time hashing
		b.EvmHeader.TxHash = types.DeriveSha(txs, trie.NewStackTrie(nil))
	}

	return b
}

// ToEvmHeader converts an Opera consensus block (inter.Block) into an EVM-compatible
// header format. This is the primary conversion function used when Opera's consensus
// produces a new block and it needs to be executed by the EVM.
//
// Parameters:
//   - block: Opera's internal block structure (from Lachesis consensus)
//   - index: Block number/index in the chain
//   - prevHash: Hash of the previous block (for ParentHash)
//   - rules: Chain rules (determines BaseFee based on upgrade status)
//
// Returns:
//   - Pointer to EvmHeader ready for EVM execution
//
// Key conversions:
//   - block.Atropos (consensus event hash) -> Hash
//   - block.Root (state root) -> Root
//   - block.Time (Opera timestamp) -> Time
//   - GasLimit always set to MaxUint64 (Opera doesn't limit gas per-block)
//   - BaseFee only set if London upgrade (EIP-1559) is active
func ToEvmHeader(block *inter.Block, index idx.Block, prevHash hash.Event, rules opera.Rules) *EvmHeader {
	// Determine base fee: only set if London upgrade is active
	baseFee := rules.Economy.MinGasPrice
	if !rules.Upgrades.London {
		baseFee = nil // London upgrade not active, no base fee
	}

	return &EvmHeader{
		Hash:       common.Hash(block.Atropos), // Consensus event hash becomes block hash
		ParentHash: common.Hash(prevHash),      // Previous block's hash
		Root:       common.Hash(block.Root),    // State root from consensus
		Number:     big.NewInt(int64(index)),   // Block number (height)
		Time:       block.Time,                 // Timestamp (Opera's high-precision type)
		GasLimit:   math.MaxUint64,             // Unlimited gas (Opera manages gas per-event)
		GasUsed:    block.GasUsed,              // Actual gas consumed by transactions
		BaseFee:    baseFee,                    // Base fee (nil if London not active)
	}
}

// ConvertFromEthHeader converts an Ethereum-formatted header (types.Header) into
// Opera's EvmHeader format. This is used when importing blocks from Ethereum-compatible
// chains or when interfacing with Ethereum tooling.
//
// Parameters:
//   - h: Ethereum header (from go-ethereum types package)
//
// Returns:
//   - Pointer to EvmHeader in Opera's format
//
// NOTE: This conversion is incomplete - some fields may not map perfectly between
// formats. The Hash is stored in Extra field, and GasLimit is set to MaxUint64
// (Opera's convention) regardless of the Ethereum header's value.
func ConvertFromEthHeader(h *types.Header) *EvmHeader {
	// NOTE: incomplete conversion - some fields may not map perfectly
	return &EvmHeader{
		Number:     h.Number,                      // Block number (direct copy)
		Coinbase:   h.Coinbase,                    // Miner/validator address
		GasLimit:   math.MaxUint64,                // Always unlimited in Opera (ignore Ethereum's limit)
		GasUsed:    h.GasUsed,                     // Gas consumed
		Root:       h.Root,                        // State root
		TxHash:     h.TxHash,                      // Transaction root
		ParentHash: h.ParentHash,                  // Parent block hash
		Time:       inter.FromUnix(int64(h.Time)), // Convert Unix timestamp to Opera timestamp
		Hash:       common.BytesToHash(h.Extra),   // Store Opera hash in Extra field (hack for compatibility)
		BaseFee:    h.BaseFee,                     // Base fee (EIP-1559)
	}
}
