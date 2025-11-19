package genesis

// Package genesis defines the configuration structures and validation logic
// for network genesis blocks. The genesis block is the first block in a
// blockchain and establishes the initial state, validator set, and network
// parameters that all nodes must agree on.
//
// Key concepts:
//   - Rules: Chain-specific parameters (block times, gas limits, validator rotation)
//   - Network: Network-level settings (chain ID, name, bootnodes)
//   - Genesis: Complete genesis block definition combining rules, network, and initial state
//
// Usage:
//   rules := genesis.NewRules("mainnet", 250)
//   network := genesis.NewNetwork(250, "Fantom Opera")
//   gen := genesis.Genesis{Rules: rules, Network: network, ...}
//
// The genesis configuration is typically loaded from a file (TOML/JSON) or
// generated programmatically for test networks (fakenet).

import (
	"math/big"
	"time"
)

// Rules defines the consensus and execution parameters that govern how blocks
// are produced and validated on this chain. These rules are immutable once
// the chain launches (changes require a hard fork).
type Rules struct {
	// Chain identification
	Name      string // Human-readable network name (e.g., "mainnet", "testnet")
	NetworkID uint64 // Unique numeric identifier for this network (prevents cross-chain replay attacks)

	// Block production timing
	BlockPeriod time.Duration // Target time between blocks (e.g., 1 second for fast chains)
	EpochLength uint64        // Number of blocks per epoch (epochs trigger validator set updates)

	// Gas and fee economics
	MinGasPrice    *big.Int // Minimum gas price (in wei) that transactions must pay
	MaxGasLimit    uint64   // Maximum gas allowed per block (prevents DoS via oversized blocks)
	GasPowerPerSec uint64   // Gas power regeneration rate per second for validators

	// Validator management
	MaxValidators     uint64        // Maximum number of validators allowed in the set
	ValidatorStakeMin *big.Int      // Minimum stake required to become a validator
	ValidatorStakeMax *big.Int      // Maximum stake per validator (prevents centralization)
	DelegationMin     *big.Int      // Minimum delegation amount
	EpochDuration     time.Duration // How long an epoch lasts before validator rotation

	// Economic parameters
	InflationRate      *big.Int            // Annual inflation rate (as a fraction, e.g., 0.05 for 5%)
	RewardDistribution map[string]*big.Int // How rewards are split (validators, delegators, treasury)

	// Upgrade and fork management
	UpgradeHeight      map[string]uint64 // Block heights at which protocol upgrades activate
	ForkID             uint16            // Fork identifier (incremented on hard forks)
	CompatibleVersions []string          // Node versions compatible with this genesis

	// EVM compatibility
	ChainID *big.Int // Ethereum-compatible chain ID (for EIP-155 transaction signing)
}
