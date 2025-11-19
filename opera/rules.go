// Package opera defines the network rules and configuration parameters for the Opera blockchain network.
//
// This package provides:
//   - Network identification constants (MainNet, TestNet, FakeNet)
//   - DAG (Directed Acyclic Graph) rules for event ordering
//   - Epoch management rules
//   - Block production rules and limits
//   - Economic parameters including gas pricing and gas power allocation
//   - Protocol upgrade configuration (Berlin, London, LLR)
//
// The Rules type serves as the central configuration structure that defines
// all consensus-critical parameters for a given Opera network deployment.

package opera

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/opera/contracts/evmwriter"

	ethparams "github.com/ethereum/go-ethereum/params"
)

// Network identification constants
const (
	// MainNetworkID is the chain ID for the Opera mainnet (0xfa = 250 in decimal)
	MainNetworkID uint64 = 0xfa

	// TestNetworkID is the chain ID for the Opera testnet (0xfa2 = 4002 in decimal)
	TestNetworkID uint64 = 0xfa2

	// FakeNetworkID is the chain ID for local/fake networks used in testing (0xfa3 = 4003 in decimal)
	FakeNetworkID uint64 = 0xfa3

	// DefaultEventGas is the base gas cost for creating an event in the DAG
	// This is the minimum gas required to publish an event to the network
	DefaultEventGas uint64 = 28000

	// Upgrade flags (bit positions for upgrade tracking)
	berlinBit = 1 << 0 // Berlin upgrade flag
	londonBit = 1 << 1 // London upgrade flag
	llrBit    = 1 << 2 // LLR (Low Latency Records) upgrade flag
)

// DefaultVMConfig provides the default EVM configuration with precompiled contracts.
// This includes the EVM writer contract which allows writing state changes from events.
var DefaultVMConfig = vm.Config{
	StatePrecompiles: map[common.Address]vm.PrecompiledStateContract{
		evmwriter.ContractAddress: &evmwriter.PreCompiledContract{},
	},
}

// RulesRLP (RLP stands for Recursive Length Prefix. It's Ethereum's serialization format) is the RLP-serializable version of Rules.
// It contains all network configuration parameters that need to be persisted
// or transmitted over the network. The Upgrades field is excluded from RLP encoding.
type RulesRLP struct {
	Name      string // Network name identifier (e.g., "main", "test", "fake")
	NetworkID uint64 // Chain ID for transaction signing and network identification

	// Graph options - DAG (Directed Acyclic Graph) configuration
	Dag DagRules

	// Epochs options - Epoch duration and gas limits
	Epochs EpochsRules

	// Blockchain options - Block production rules
	Blocks BlocksRules

	// Economy options - Gas pricing and economic parameters
	Economy EconomyRules

	// Upgrades - Protocol upgrade flags (not RLP-encoded)
	Upgrades Upgrades `rlp:"-"`
}

// Rules describes the complete configuration for an Opera network.
// This is the main type used throughout the codebase to access network parameters.
//
// Note: When implementing Copy(), ensure all non-copiable variables (like *big.Int)
// are properly deep-copied to avoid shared state issues.
type Rules RulesRLP

// GasPowerRules defines the gas power allocation rules for the consensus mechanism.
// Gas power determines how much gas a validator can use when creating events.
// There are two windows: short (for immediate needs) and long (for sustained operations).
type GasPowerRules struct {
	// AllocPerSec is the rate at which gas power is allocated per second
	// This determines how quickly validators accumulate gas power
	AllocPerSec uint64

	// MaxAllocPeriod is the maximum time window for accumulating gas power
	// Gas power cannot accumulate beyond this period
	MaxAllocPeriod inter.Timestamp

	// StartupAllocPeriod is the initial period where validators get extra gas power
	// This helps new validators start producing events immediately
	StartupAllocPeriod inter.Timestamp

	// MinStartupGas is the minimum gas power given to validators during startup
	// Ensures validators can always create at least one event
	MinStartupGas uint64
}

// GasRulesRLPV1 defines gas costs for various operations in the network.
// This is version 1 of the gas rules structure, supporting post-LLR features.
type GasRulesRLPV1 struct {
	// MaxEventGas is the maximum gas allowed per event
	// Events exceeding this limit are rejected
	MaxEventGas uint64

	// EventGas is the base gas cost for creating an event
	// This is charged for every event published to the network
	EventGas uint64

	// ParentGas is the gas cost per parent reference in an event
	// Events can reference multiple parent events, each parent costs this amount
	ParentGas uint64

	// ExtraDataGas is the gas cost per byte of extra data in an event
	// Additional data beyond the standard event structure incurs this cost
	ExtraDataGas uint64

	// Post-LLR fields (Low Latency Records upgrade)

	// BlockVotesBaseGas is the base gas cost for block voting operations
	BlockVotesBaseGas uint64

	// BlockVoteGas is the gas cost per individual block vote
	BlockVoteGas uint64

	// EpochVoteGas is the gas cost per epoch vote
	EpochVoteGas uint64

	// MisbehaviourProofGas is the gas cost for submitting a misbehaviour proof
	// This incentivizes reporting validator misbehavior
	MisbehaviourProofGas uint64
}

// GasRules is the current version of gas rules (aliased to V1)
type GasRules GasRulesRLPV1

// EpochsRules defines the rules for epoch management.
// Epochs are time-based periods that group events together for finalization.
type EpochsRules struct {
	// MaxEpochGas is the maximum total gas allowed in a single epoch
	// Once this limit is reached, the epoch must be finalized
	MaxEpochGas uint64

	// MaxEpochDuration is the maximum time an epoch can last
	// Epochs are finalized when either gas limit or time limit is reached
	MaxEpochDuration inter.Timestamp
}

// DagRules defines the rules for the Lachesis DAG (Directed Acyclic Graph).
// The DAG structure allows events to reference multiple parent events,
// enabling parallel event processing while maintaining ordering.
type DagRules struct {
	// MaxParents is the maximum number of parent events an event can reference
	// This limits the branching factor of the DAG
	MaxParents idx.Event

	// MaxFreeParents is the maximum number of parents that don't incur gas cost
	// Parents beyond this count require additional gas (ParentGas per parent)
	MaxFreeParents idx.Event

	// MaxExtraData is the maximum size (in bytes) of extra data in an event
	// Extra data beyond this limit is rejected
	MaxExtraData uint32
}

// BlocksMissed tracks information about blocks missed by a validator.
// This is used for slashing and validator reputation tracking.
type BlocksMissed struct {
	BlocksNum idx.Block       // Number of blocks missed
	Period    inter.Timestamp // Time period over which blocks were missed
}

// EconomyRules contains all economic parameters for the network.
// These rules govern gas pricing, validator incentives, and economic security.
type EconomyRules struct {
	// BlockMissedSlack is the number of blocks a validator can miss before penalties
	// This provides tolerance for temporary network issues
	BlockMissedSlack idx.Block

	// Gas contains all gas-related rules and costs
	Gas GasRules

	// MinGasPrice is the minimum gas price (in wei) for transactions
	// Transactions with lower gas prices are rejected
	MinGasPrice *big.Int

	// ShortGasPower is the gas power allocation for short-term operations
	// Used for immediate event creation needs
	ShortGasPower GasPowerRules

	// LongGasPower is the gas power allocation for long-term operations
	// Used for sustained validator operations over longer periods
	LongGasPower GasPowerRules
}

// BlocksRules contains rules for block production and validation.
type BlocksRules struct {
	// MaxBlockGas is the technical hard limit for gas per block
	// Note: Actual block gas is mostly governed by gas power allocation,
	// this is just a safety limit
	MaxBlockGas uint64

	// MaxEmptyBlockSkipPeriod is the maximum time validators can skip empty blocks
	// Validators must produce blocks even if empty, unless within this period
	MaxEmptyBlockSkipPeriod inter.Timestamp
}

// Upgrades tracks which protocol upgrades are enabled for a network.
// These flags control feature availability and compatibility.
type Upgrades struct {
	Berlin bool // Berlin upgrade (EIP-2565, EIP-2929, EIP-2718, EIP-2930)
	London bool // London upgrade (EIP-1559, EIP-3198, EIP-3529, EIP-3541)
	Llr    bool // LLR (Low Latency Records) upgrade - Opera-specific feature
}

// UpgradeHeight specifies at which block height an upgrade becomes active.
// This allows for scheduled protocol upgrades.
type UpgradeHeight struct {
	Upgrades Upgrades  // Which upgrades are activated
	Height   idx.Block // Block height at which upgrades take effect
}

// EvmChainConfig converts Opera Rules to Ethereum ChainConfig format.
// This is used for transaction signing and EVM execution compatibility.
//
// Parameters:
//   - hh: Array of upgrade heights, ordered by block height
//
// Returns:
//   - *ethparams.ChainConfig: Ethereum-compatible chain configuration
//
// The function processes upgrade heights sequentially and sets BerlinBlock
// and LondonBlock based on the first occurrence of each upgrade flag.
func (r Rules) EvmChainConfig(hh []UpgradeHeight) *ethparams.ChainConfig {
	// Start with all Ethereum protocol changes as base
	cfg := *ethparams.AllEthashProtocolChanges

	// Set the chain ID from network ID
	cfg.ChainID = new(big.Int).SetUint64(r.NetworkID)

	// Initialize upgrade blocks as nil (not activated)
	cfg.BerlinBlock = nil
	cfg.LondonBlock = nil

	// Process each upgrade height in order
	for i, h := range hh {
		height := new(big.Int)

		// For the first entry (i == 0), height remains 0 (genesis)
		// For subsequent entries, set the actual block height
		if i > 0 {
			height.SetUint64(uint64(h.Height))
		}

		// Handle Berlin upgrade activation
		// Set BerlinBlock on first occurrence, clear it if disabled later
		if cfg.BerlinBlock == nil && h.Upgrades.Berlin {
			cfg.BerlinBlock = height
		}
		if !h.Upgrades.Berlin {
			cfg.BerlinBlock = nil
		}

		// Handle London upgrade activation
		// Set LondonBlock on first occurrence, clear it if disabled later
		if cfg.LondonBlock == nil && h.Upgrades.London {
			cfg.LondonBlock = height
		}
		if !h.Upgrades.London {
			cfg.LondonBlock = nil
		}
	}

	return &cfg
}

// MainNetRules returns the configuration rules for Opera mainnet.
// This is the production network configuration with conservative parameters.
func MainNetRules() Rules {
	return Rules{
		Name:      "main",
		NetworkID: MainNetworkID,
		Dag:       DefaultDagRules(),
		Epochs:    DefaultEpochsRules(),
		Economy:   DefaultEconomyRules(),
		Blocks: BlocksRules{
			MaxBlockGas:             20500000, // 20.5M gas per block
			MaxEmptyBlockSkipPeriod: inter.Timestamp(1 * time.Minute),
		},
	}
}

// TestNetRules returns the configuration rules for Opera testnet.
// Testnet uses the same parameters as mainnet for realistic testing.
func TestNetRules() Rules {
	return Rules{
		Name:      "test",
		NetworkID: TestNetworkID,
		Dag:       DefaultDagRules(),
		Epochs:    DefaultEpochsRules(),
		Economy:   DefaultEconomyRules(),
		Blocks: BlocksRules{
			MaxBlockGas:             20500000, // Same as mainnet
			MaxEmptyBlockSkipPeriod: inter.Timestamp(1 * time.Minute),
		},
	}
}

// FakeNetRules returns the configuration rules for fake/local networks.
// Fake networks use accelerated parameters for faster testing and development:
//   - Shorter epoch durations (10 minutes vs 4 hours)
//   - Reduced epoch gas limits (1/5 of mainnet)
//   - Faster gas power allocation (1000x multiplier)
//   - Shorter empty block skip period (3 seconds vs 1 minute)
//   - All upgrades enabled by default
func FakeNetRules() Rules {
	return Rules{
		Name:      "fake",
		NetworkID: FakeNetworkID,
		Dag:       DefaultDagRules(),
		Epochs:    FakeNetEpochsRules(), // Accelerated epochs
		Economy:   FakeEconomyRules(),   // Accelerated gas power
		Blocks: BlocksRules{
			MaxBlockGas:             20500000,
			MaxEmptyBlockSkipPeriod: inter.Timestamp(3 * time.Second), // Much shorter for testing
		},
		Upgrades: Upgrades{
			Berlin: true, // All upgrades enabled for testing
			London: true,
			Llr:    true,
		},
	}
}

// DefaultEconomyRules returns the mainnet economy configuration.
// This defines gas pricing and gas power allocation for production use.
func DefaultEconomyRules() EconomyRules {
	return EconomyRules{
		BlockMissedSlack: 50, // Allow 50 missed blocks before penalty
		Gas:              DefaultGasRules(),
		MinGasPrice:      big.NewInt(1e9), // 1 Gwei minimum gas price
		ShortGasPower:    DefaultShortGasPowerRules(),
		LongGasPower:     DefaulLongGasPowerRules(),
	}
}

// FakeEconomyRules returns the fake network economy configuration.
// Uses accelerated gas power allocation for faster testing cycles.
func FakeEconomyRules() EconomyRules {
	cfg := DefaultEconomyRules()
	// Override with accelerated gas power rules (1000x faster)
	cfg.ShortGasPower = FakeShortGasPowerRules()
	cfg.LongGasPower = FakeLongGasPowerRules()
	return cfg
}

// DefaultDagRules returns the default DAG configuration.
// These rules apply to all network types (mainnet, testnet, fake).
func DefaultDagRules() DagRules {
	return DagRules{
		MaxParents:     10,  // Events can reference up to 10 parent events
		MaxFreeParents: 3,   // First 3 parents are free, rest cost gas
		MaxExtraData:   128, // Maximum 128 bytes of extra data per event
	}
}

// DefaultEpochsRules returns the mainnet epoch configuration.
// Epochs finalize when either gas limit or time limit is reached.
func DefaultEpochsRules() EpochsRules {
	return EpochsRules{
		MaxEpochGas:      1500000000,                     // 1.5B gas per epoch
		MaxEpochDuration: inter.Timestamp(4 * time.Hour), // 4 hour maximum epoch duration
	}
}

// DefaultGasRules returns the default gas costs for network operations.
// These costs apply to mainnet and testnet.
func DefaultGasRules() GasRules {
	return GasRules{
		MaxEventGas:          10000000 + DefaultEventGas, // 10M + base event gas
		EventGas:             DefaultEventGas,            // 28,000 base gas per event
		ParentGas:            2400,                       // 2,400 gas per parent reference
		ExtraDataGas:         25,                         // 25 gas per byte of extra data
		BlockVotesBaseGas:    1024,                       // Base cost for block voting
		BlockVoteGas:         512,                        // Per-block vote cost
		EpochVoteGas:         1536,                       // Per-epoch vote cost
		MisbehaviourProofGas: 71536,                      // Cost to submit misbehaviour proof
	}
}

// FakeNetEpochsRules returns accelerated epoch rules for fake networks.
// Epochs finalize much faster to speed up testing.
func FakeNetEpochsRules() EpochsRules {
	cfg := DefaultEpochsRules()
	cfg.MaxEpochGas /= 5                                     // 1/5 of mainnet gas limit
	cfg.MaxEpochDuration = inter.Timestamp(10 * time.Minute) // 10 minutes vs 4 hours
	return cfg
}

// DefaulLongGasPowerRules returns the long-window gas power configuration.
// Long window is used for sustained validator operations over extended periods.
func DefaulLongGasPowerRules() GasPowerRules {
	return GasPowerRules{
		AllocPerSec:        100 * DefaultEventGas,             // 2.8M gas/sec allocation rate
		MaxAllocPeriod:     inter.Timestamp(60 * time.Minute), // 60 minute accumulation window
		StartupAllocPeriod: inter.Timestamp(5 * time.Second),  // 5 second startup boost
		MinStartupGas:      DefaultEventGas * 20,              // 560K gas minimum at startup
	}
}

// DefaultShortGasPowerRules returns the short-window gas power configuration.
// Short window provides faster allocation for immediate event creation needs.
// Compared to long window:
//   - 2x faster allocation rate
//   - 6x lower maximum accumulated gas power
//   - 2x shorter startup period
func DefaultShortGasPowerRules() GasPowerRules {
	cfg := DefaulLongGasPowerRules()
	cfg.AllocPerSec *= 2        // Double the allocation rate
	cfg.StartupAllocPeriod /= 2 // Half the startup period
	cfg.MaxAllocPeriod /= 2 * 6 // 12x shorter max period (2 * 6)
	return cfg
}

// FakeLongGasPowerRules returns accelerated long-window gas power for fake networks.
// Uses 1000x faster allocation to speed up testing.
func FakeLongGasPowerRules() GasPowerRules {
	config := DefaulLongGasPowerRules()
	config.AllocPerSec *= 1000 // 1000x faster for testing
	return config
}

// FakeShortGasPowerRules returns accelerated short-window gas power for fake networks.
// Uses 1000x faster allocation to speed up testing.
func FakeShortGasPowerRules() GasPowerRules {
	config := DefaultShortGasPowerRules()
	config.AllocPerSec *= 1000 // 1000x faster for testing
	return config
}

// Copy creates a deep copy of Rules.
// This is necessary because Rules contains pointer types (*big.Int) that
// would be shared in a shallow copy, leading to unintended mutations.
//
// Returns:
//   - Rules: A new Rules instance with all fields properly copied
func (r Rules) Copy() Rules {
	cp := r
	// Deep copy MinGasPrice to avoid shared state
	cp.Economy.MinGasPrice = new(big.Int).Set(r.Economy.MinGasPrice)
	return cp
}

// String returns a JSON representation of Rules for debugging and logging.
// This is useful for inspecting network configuration at runtime.
//
// Returns:
//   - string: JSON-formatted representation of the Rules
func (r Rules) String() string {
	b, _ := json.Marshal(&r)
	return string(b)
}
