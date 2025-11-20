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

// Package evmcore provides EVM block and transaction processing functionality.
// This file specifically handles fake genesis block creation for testing and development.

package evmcore

import (
	"crypto/ecdsa"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/rony4d/go-opera-asset/inter"
)

// FakeGenesisTime is the default timestamp used for fake genesis blocks.
// Timestamp: 1608600000 seconds since Unix epoch (December 22, 2020)
// This provides a consistent reference point for fake network initialization.
var FakeGenesisTime = inter.Timestamp(1608600000 * time.Second)

// ApplyFakeGenesis initializes a fake genesis block with the specified account balances.
//
// This function is used for testing, development, and fake network initialization.
// It creates a genesis block (block number 0) with pre-funded accounts, allowing
// for rapid testing without needing real network interactions.
//
// Process:
//  1. Sets initial balances for all specified accounts
//  2. Commits the state to the database and computes the state root
//  3. Creates a genesis block with the computed state root
//
// Parameters:
//   - statedb: The state database where account balances will be set
//   - time: The timestamp for the genesis block (typically FakeGenesisTime)
//   - balances: Map of account addresses to their initial balances (in wei)
//
// Returns:
//   - *EvmBlock: The created genesis block (block number 0)
//   - error: Any error encountered during state commit or block creation
//
// Example:
//
//	balances := map[common.Address]*big.Int{
//	    common.HexToAddress("0x123..."): big.NewInt(1000000000000000000), // 1 ETH
//	}
//	block, err := ApplyFakeGenesis(statedb, FakeGenesisTime, balances)
func ApplyFakeGenesis(statedb *state.StateDB, time inter.Timestamp, balances map[common.Address]*big.Int) (*EvmBlock, error) {
	// Set initial balances for all accounts specified in the map
	// This pre-funds accounts for testing and development purposes
	for acc, balance := range balances {
		statedb.SetBalance(acc, balance)
	}

	// Commit the state changes and compute the state root hash
	// The 'true' parameter indicates a clean commit (no intermediate state caching)
	root, err := flush(statedb, true)
	if err != nil {
		return nil, err
	}

	// Create the genesis block with the computed state root
	block := genesisBlock(time, root)

	return block, nil
}

// flush commits state changes to the database and returns the state root hash.
//
// This function performs a two-phase commit:
//  1. Commits pending state changes to the state trie
//  2. Commits the trie to the underlying database
//
// The 'clean' parameter controls whether to perform a clean commit:
//   - clean=true: Full commit, used for genesis initialization
//   - clean=false: Incremental commit with trie capping for memory management
//
// Parameters:
//   - statedb: The state database containing pending changes
//   - clean: Whether to perform a clean commit (true for genesis, false for regular commits)
//
// Returns:
//   - root: The Merkle root hash of the state trie after commit
//   - err: Any error encountered during the commit process
func flush(statedb *state.StateDB, clean bool) (root common.Hash, err error) {
	// Phase 1: Commit pending state changes to the state trie
	// This computes the Merkle root hash of all account states
	root, err = statedb.Commit(clean)
	if err != nil {
		return
	}

	// Phase 2: Commit the trie nodes to the underlying database
	// This persists the trie structure to disk
	// The 'false' parameter means don't force a full trie write
	err = statedb.Database().TrieDB().Commit(root, false, nil)
	if err != nil {
		return
	}

	// For non-clean commits, cap the trie cache to manage memory usage
	// Cap(0) removes all nodes that are not referenced by the current root
	if !clean {
		err = statedb.Database().TrieDB().Cap(0)
	}

	return
}

// genesisBlock creates a genesis block (block number 0) with the specified parameters.
//
// Genesis blocks are special: they have no parent block and initialize the blockchain.
// This function creates a minimal genesis block suitable for fake/test networks.
//
// Genesis Block Properties:
//   - Number: 0 (genesis block)
//   - Time: Specified timestamp
//   - GasLimit: Maximum uint64 (allows unlimited gas for testing)
//   - Root: State root hash from committed state
//   - TxHash: Empty root hash (no transactions in genesis)
//
// Parameters:
//   - time: The timestamp for the genesis block
//   - root: The state root hash from the committed state database
//
// Returns:
//   - *EvmBlock: A pointer to the created genesis block
func genesisBlock(time inter.Timestamp, root common.Hash) *EvmBlock {
	block := &EvmBlock{
		EvmHeader: EvmHeader{
			Number:   big.NewInt(0),       // Genesis block is always block 0
			Time:     time,                // Block timestamp
			GasLimit: math.MaxUint64,      // Maximum gas limit for testing flexibility
			Root:     root,                // State root hash (Merkle root of all accounts)
			TxHash:   types.EmptyRootHash, // No transactions in genesis block
		},
	}

	return block
}

// MustApplyFakeGenesis is a convenience wrapper around ApplyFakeGenesis that panics on error.
//
// This function is useful in test code or initialization code where a failure
// to create the genesis block should immediately terminate the program.
// It logs a critical error and panics if genesis creation fails.
//
// Parameters:
//   - statedb: The state database where account balances will be set
//   - time: The timestamp for the genesis block
//   - balances: Map of account addresses to their initial balances
//
// Returns:
//   - *EvmBlock: The created genesis block (never nil, panics on error)
//
// Panics:
//   - If ApplyFakeGenesis returns an error, logs a critical error and panics
func MustApplyFakeGenesis(statedb *state.StateDB, time inter.Timestamp, balances map[common.Address]*big.Int) *EvmBlock {
	block, err := ApplyFakeGenesis(statedb, time, balances)
	if err != nil {
		// Log critical error and panic - genesis creation failure is fatal
		log.Crit("ApplyFakeGenesis", "err", err)
	}
	return block
}

// FakeKey generates a deterministic fake private key for testing purposes.
//
// This function uses a seeded random number generator to produce deterministic
// private keys. Given the same input 'n', it will always generate the same key.
// This is useful for creating consistent test accounts and validators.
//
// Use Cases:
//   - Creating test accounts with known keys
//   - Generating validator keys for fake networks
//   - Reproducible testing scenarios
//
// Parameters:
//   - n: The seed/index for key generation (deterministic: same n = same key)
//
// Returns:
//   - *ecdsa.PrivateKey: A deterministic ECDSA private key using secp256k1 curve
//
// Panics:
//   - If key generation fails (should never happen in practice)
//
// Example:
//
//	key0 := FakeKey(0)  // First fake key
//	key1 := FakeKey(1)  // Second fake key (different from key0)
//	key0Again := FakeKey(0)  // Same as key0 (deterministic)
func FakeKey(n int) *ecdsa.PrivateKey {
	// Create a seeded random number generator
	// Using the index 'n' as the seed ensures deterministic key generation
	reader := rand.New(rand.NewSource(int64(n)))

	// Generate an ECDSA key pair using the secp256k1 curve (Bitcoin/Ethereum curve)
	// The seeded reader ensures the same seed produces the same key
	key, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		// Key generation should never fail, but panic if it does
		panic(err)
	}

	return key
}
