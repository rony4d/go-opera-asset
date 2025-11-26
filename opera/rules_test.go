package opera

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/rony4d/go-opera-asset/inter"
	"github.com/rony4d/go-opera-asset/opera/contracts/evmwriter"
)

// TestNetworkConstants verifies that network ID constants are correctly defined.
// These constants are used throughout the codebase to identify which network
// a node is running on.
func TestNetworkConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant uint64
		want     uint64
	}{
		{"MainNetworkID", MainNetworkID, 0xfa},  // 250 in decimal
		{"TestNetworkID", TestNetworkID, 0xfa2}, // 4002 in decimal
		{"FakeNetworkID", FakeNetworkID, 0xfa3}, // 4003 in decimal
		{"DefaultEventGas", DefaultEventGas, 28000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.want)
			}
		})
	}
}

// TestUpgradeBits verifies that upgrade bit flags are correctly defined.
// These bits are used to track which protocol upgrades are enabled.
func TestUpgradeBits(t *testing.T) {
	if berlinBit != 1<<0 {
		t.Errorf("berlinBit = %d, want %d", berlinBit, 1<<0)
	}
	if londonBit != 1<<1 {
		t.Errorf("londonBit = %d, want %d", londonBit, 1<<1)
	}
	if llrBit != 1<<2 {
		t.Errorf("llrBit = %d, want %d", llrBit, 1<<2)
	}
}

// TestDefaultVMConfig verifies that the default VM config includes the EVM writer precompile.
// The EVM writer contract allows writing state changes from events.
func TestDefaultVMConfig(t *testing.T) {
	if DefaultVMConfig.StatePrecompiles == nil {
		t.Fatal("DefaultVMConfig.StatePrecompiles is nil")
	}

	precompile, exists := DefaultVMConfig.StatePrecompiles[evmwriter.ContractAddress]
	if !exists {
		t.Fatalf("EVM writer precompile not found at address %s", evmwriter.ContractAddress)
	}

	if precompile == nil {
		t.Fatal("EVM writer precompile is nil")
	}

	// Verify it's the correct type
	if _, ok := precompile.(*evmwriter.PreCompiledContract); !ok {
		t.Errorf("Expected *evmwriter.PreCompiledContract, got %T", precompile)
	}
}

// TestMainNetRules verifies that MainNetRules returns the correct configuration.
// Mainnet uses conservative, production-ready parameters.
func TestMainNetRules(t *testing.T) {
	rules := MainNetRules()

	// Verify network identification
	if rules.Name != "main" {
		t.Errorf("Name = %q, want %q", rules.Name, "main")
	}
	if rules.NetworkID != MainNetworkID {
		t.Errorf("NetworkID = %d, want %d", rules.NetworkID, MainNetworkID)
	}

	// Verify blocks configuration
	if rules.Blocks.MaxBlockGas != 20500000 {
		t.Errorf("MaxBlockGas = %d, want %d", rules.Blocks.MaxBlockGas, 20500000)
	}
	if rules.Blocks.MaxEmptyBlockSkipPeriod != inter.Timestamp(1*time.Minute) {
		t.Errorf("MaxEmptyBlockSkipPeriod = %v, want %v",
			rules.Blocks.MaxEmptyBlockSkipPeriod, inter.Timestamp(1*time.Minute))
	}

	// Verify upgrades are not set (mainnet starts with no upgrades)
	if rules.Upgrades.Berlin || rules.Upgrades.London || rules.Upgrades.Llr {
		t.Errorf("Mainnet should not have upgrades enabled by default: %+v", rules.Upgrades)
	}
}

// TestTestNetRules verifies that TestNetRules returns the correct configuration.
// Testnet uses the same parameters as mainnet for realistic testing.
func TestTestNetRules(t *testing.T) {
	rules := TestNetRules()

	// Verify network identification
	if rules.Name != "test" {
		t.Errorf("Name = %q, want %q", rules.Name, "test")
	}
	if rules.NetworkID != TestNetworkID {
		t.Errorf("NetworkID = %d, want %d", rules.NetworkID, TestNetworkID)
	}

	// Verify blocks configuration (should match mainnet)
	if rules.Blocks.MaxBlockGas != 20500000 {
		t.Errorf("MaxBlockGas = %d, want %d", rules.Blocks.MaxBlockGas, 20500000)
	}
	if rules.Blocks.MaxEmptyBlockSkipPeriod != inter.Timestamp(1*time.Minute) {
		t.Errorf("MaxEmptyBlockSkipPeriod = %v, want %v",
			rules.Blocks.MaxEmptyBlockSkipPeriod, inter.Timestamp(1*time.Minute))
	}
}

// TestFakeNetRules verifies that FakeNetRules returns accelerated configuration.
// Fake networks use faster parameters for testing and development.
func TestFakeNetRules(t *testing.T) {
	rules := FakeNetRules()

	// Verify network identification
	if rules.Name != "fake" {
		t.Errorf("Name = %q, want %q", rules.Name, "fake")
	}
	if rules.NetworkID != FakeNetworkID {
		t.Errorf("NetworkID = %d, want %d", rules.NetworkID, FakeNetworkID)
	}

	// Verify accelerated blocks configuration
	if rules.Blocks.MaxBlockGas != 20500000 {
		t.Errorf("MaxBlockGas = %d, want %d", rules.Blocks.MaxBlockGas, 20500000)
	}
	// Fake network has much shorter empty block skip period
	if rules.Blocks.MaxEmptyBlockSkipPeriod != inter.Timestamp(3*time.Second) {
		t.Errorf("MaxEmptyBlockSkipPeriod = %v, want %v",
			rules.Blocks.MaxEmptyBlockSkipPeriod, inter.Timestamp(3*time.Second))
	}

	// Verify all upgrades are enabled for fake networks
	if !rules.Upgrades.Berlin {
		t.Error("Fake network should have Berlin upgrade enabled")
	}
	if !rules.Upgrades.London {
		t.Error("Fake network should have London upgrade enabled")
	}
	if !rules.Upgrades.Llr {
		t.Error("Fake network should have LLR upgrade enabled")
	}
}

// TestDefaultDagRules verifies the default DAG configuration.
// DAG rules apply to all network types.
func TestDefaultDagRules(t *testing.T) {
	rules := DefaultDagRules()

	if rules.MaxParents != 10 {
		t.Errorf("MaxParents = %d, want %d", rules.MaxParents, 10)
	}
	if rules.MaxFreeParents != 3 {
		t.Errorf("MaxFreeParents = %d, want %d", rules.MaxFreeParents, 3)
	}
	if rules.MaxExtraData != 128 {
		t.Errorf("MaxExtraData = %d, want %d", rules.MaxExtraData, 128)
	}
}

// TestDefaultEpochsRules verifies the mainnet epoch configuration.
func TestDefaultEpochsRules(t *testing.T) {
	rules := DefaultEpochsRules()

	if rules.MaxEpochGas != 1500000000 {
		t.Errorf("MaxEpochGas = %d, want %d", rules.MaxEpochGas, 1500000000)
	}
	if rules.MaxEpochDuration != inter.Timestamp(4*time.Hour) {
		t.Errorf("MaxEpochDuration = %v, want %v",
			rules.MaxEpochDuration, inter.Timestamp(4*time.Hour))
	}
}

// TestFakeNetEpochsRules verifies that fake network epochs are accelerated.
func TestFakeNetEpochsRules(t *testing.T) {
	rules := FakeNetEpochsRules()

	// Should be 1/5 of mainnet gas limit
	expectedGas := uint64(1500000000 / 5)
	if rules.MaxEpochGas != expectedGas {
		t.Errorf("MaxEpochGas = %d, want %d", rules.MaxEpochGas, expectedGas)
	}

	// Should be 10 minutes instead of 4 hours
	if rules.MaxEpochDuration != inter.Timestamp(10*time.Minute) {
		t.Errorf("MaxEpochDuration = %v, want %v",
			rules.MaxEpochDuration, inter.Timestamp(10*time.Minute))
	}
}

// TestDefaultGasRules verifies the default gas costs for network operations.
func TestDefaultGasRules(t *testing.T) {
	rules := DefaultGasRules()

	if rules.MaxEventGas != 10000000+DefaultEventGas {
		t.Errorf("MaxEventGas = %d, want %d", rules.MaxEventGas, 10000000+DefaultEventGas)
	}
	if rules.EventGas != DefaultEventGas {
		t.Errorf("EventGas = %d, want %d", rules.EventGas, DefaultEventGas)
	}
	if rules.ParentGas != 2400 {
		t.Errorf("ParentGas = %d, want %d", rules.ParentGas, 2400)
	}
	if rules.ExtraDataGas != 25 {
		t.Errorf("ExtraDataGas = %d, want %d", rules.ExtraDataGas, 25)
	}
	if rules.BlockVotesBaseGas != 1024 {
		t.Errorf("BlockVotesBaseGas = %d, want %d", rules.BlockVotesBaseGas, 1024)
	}
	if rules.BlockVoteGas != 512 {
		t.Errorf("BlockVoteGas = %d, want %d", rules.BlockVoteGas, 512)
	}
	if rules.EpochVoteGas != 1536 {
		t.Errorf("EpochVoteGas = %d, want %d", rules.EpochVoteGas, 1536)
	}
	if rules.MisbehaviourProofGas != 71536 {
		t.Errorf("MisbehaviourProofGas = %d, want %d", rules.MisbehaviourProofGas, 71536)
	}
}

// TestDefaultLongGasPowerRules verifies the long-window gas power configuration.
func TestDefaultLongGasPowerRules(t *testing.T) {
	rules := DefaulLongGasPowerRules()

	expectedAllocPerSec := 100 * DefaultEventGas // 2.8M gas/sec
	if rules.AllocPerSec != expectedAllocPerSec {
		t.Errorf("AllocPerSec = %d, want %d", rules.AllocPerSec, expectedAllocPerSec)
	}

	if rules.MaxAllocPeriod != inter.Timestamp(60*time.Minute) {
		t.Errorf("MaxAllocPeriod = %v, want %v",
			rules.MaxAllocPeriod, inter.Timestamp(60*time.Minute))
	}

	if rules.StartupAllocPeriod != inter.Timestamp(5*time.Second) {
		t.Errorf("StartupAllocPeriod = %v, want %v",
			rules.StartupAllocPeriod, inter.Timestamp(5*time.Second))
	}

	expectedMinStartupGas := DefaultEventGas * 20 // 560K gas
	if rules.MinStartupGas != expectedMinStartupGas {
		t.Errorf("MinStartupGas = %d, want %d", rules.MinStartupGas, expectedMinStartupGas)
	}
}

// TestDefaultShortGasPowerRules verifies the short-window gas power configuration.
// Short window should have 2x faster allocation and shorter periods than long window.
func TestDefaultShortGasPowerRules(t *testing.T) {
	rules := DefaultShortGasPowerRules()
	longRules := DefaulLongGasPowerRules()

	// Should be 2x the long window allocation rate
	expectedAllocPerSec := longRules.AllocPerSec * 2
	if rules.AllocPerSec != expectedAllocPerSec {
		t.Errorf("AllocPerSec = %d, want %d", rules.AllocPerSec, expectedAllocPerSec)
	}

	// Should be half the startup period
	expectedStartupPeriod := longRules.StartupAllocPeriod / 2
	if rules.StartupAllocPeriod != expectedStartupPeriod {
		t.Errorf("StartupAllocPeriod = %v, want %v",
			rules.StartupAllocPeriod, expectedStartupPeriod)
	}

	// Should be 12x shorter max period (2 * 6)
	expectedMaxPeriod := longRules.MaxAllocPeriod / 12
	if rules.MaxAllocPeriod != expectedMaxPeriod {
		t.Errorf("MaxAllocPeriod = %v, want %v",
			rules.MaxAllocPeriod, expectedMaxPeriod)
	}
}

// TestFakeLongGasPowerRules verifies that fake network long gas power is accelerated.
func TestFakeLongGasPowerRules(t *testing.T) {
	rules := FakeLongGasPowerRules()
	defaultRules := DefaulLongGasPowerRules()

	// Should be 1000x faster allocation
	expectedAllocPerSec := defaultRules.AllocPerSec * 1000
	if rules.AllocPerSec != expectedAllocPerSec {
		t.Errorf("AllocPerSec = %d, want %d", rules.AllocPerSec, expectedAllocPerSec)
	}

	// Other fields should remain the same
	if rules.MaxAllocPeriod != defaultRules.MaxAllocPeriod {
		t.Errorf("MaxAllocPeriod should remain unchanged: got %v, want %v",
			rules.MaxAllocPeriod, defaultRules.MaxAllocPeriod)
	}
	if rules.StartupAllocPeriod != defaultRules.StartupAllocPeriod {
		t.Errorf("StartupAllocPeriod should remain unchanged: got %v, want %v",
			rules.StartupAllocPeriod, defaultRules.StartupAllocPeriod)
	}
	if rules.MinStartupGas != defaultRules.MinStartupGas {
		t.Errorf("MinStartupGas should remain unchanged: got %d, want %d",
			rules.MinStartupGas, defaultRules.MinStartupGas)
	}
}

// TestFakeShortGasPowerRules verifies that fake network short gas power is accelerated.
func TestFakeShortGasPowerRules(t *testing.T) {
	rules := FakeShortGasPowerRules()
	defaultRules := DefaultShortGasPowerRules()

	// Should be 1000x faster allocation
	expectedAllocPerSec := defaultRules.AllocPerSec * 1000
	if rules.AllocPerSec != expectedAllocPerSec {
		t.Errorf("AllocPerSec = %d, want %d", rules.AllocPerSec, expectedAllocPerSec)
	}

	// Other fields should remain the same
	if rules.MaxAllocPeriod != defaultRules.MaxAllocPeriod {
		t.Errorf("MaxAllocPeriod should remain unchanged: got %v, want %v",
			rules.MaxAllocPeriod, defaultRules.MaxAllocPeriod)
	}
	if rules.StartupAllocPeriod != defaultRules.StartupAllocPeriod {
		t.Errorf("StartupAllocPeriod should remain unchanged: got %v, want %v",
			rules.StartupAllocPeriod, defaultRules.StartupAllocPeriod)
	}
	if rules.MinStartupGas != defaultRules.MinStartupGas {
		t.Errorf("MinStartupGas should remain unchanged: got %d, want %d",
			rules.MinStartupGas, defaultRules.MinStartupGas)
	}
}

// TestDefaultEconomyRules verifies the mainnet economy configuration.
func TestDefaultEconomyRules(t *testing.T) {
	rules := DefaultEconomyRules()

	if rules.BlockMissedSlack != 50 {
		t.Errorf("BlockMissedSlack = %d, want %d", rules.BlockMissedSlack, 50)
	}

	// Verify MinGasPrice is 1 Gwei
	expectedMinGasPrice := big.NewInt(1e9)
	if rules.MinGasPrice.Cmp(expectedMinGasPrice) != 0 {
		t.Errorf("MinGasPrice = %s, want %s", rules.MinGasPrice.String(), expectedMinGasPrice.String())
	}

	// Verify gas rules are set
	if rules.Gas.MaxEventGas == 0 {
		t.Error("Gas rules should be set")
	}

	// Verify gas power rules are set
	if rules.ShortGasPower.AllocPerSec == 0 {
		t.Error("ShortGasPower should be set")
	}
	if rules.LongGasPower.AllocPerSec == 0 {
		t.Error("LongGasPower should be set")
	}
}

// TestFakeEconomyRules verifies that fake network economy uses accelerated gas power.
func TestFakeEconomyRules(t *testing.T) {
	rules := FakeEconomyRules()
	defaultRules := DefaultEconomyRules()

	// BlockMissedSlack should remain the same
	if rules.BlockMissedSlack != defaultRules.BlockMissedSlack {
		t.Errorf("BlockMissedSlack should remain unchanged: got %d, want %d",
			rules.BlockMissedSlack, defaultRules.BlockMissedSlack)
	}

	// MinGasPrice should remain the same
	if rules.MinGasPrice.Cmp(defaultRules.MinGasPrice) != 0 {
		t.Errorf("MinGasPrice should remain unchanged: got %s, want %s",
			rules.MinGasPrice.String(), defaultRules.MinGasPrice.String())
	}

	// Gas rules should remain the same
	if !reflect.DeepEqual(rules.Gas, defaultRules.Gas) {
		t.Error("Gas rules should remain unchanged")
	}

	// ShortGasPower should be accelerated (1000x)
	expectedShortAlloc := defaultRules.ShortGasPower.AllocPerSec * 1000
	if rules.ShortGasPower.AllocPerSec != expectedShortAlloc {
		t.Errorf("ShortGasPower.AllocPerSec = %d, want %d",
			rules.ShortGasPower.AllocPerSec, expectedShortAlloc)
	}

	// LongGasPower should be accelerated (1000x)
	expectedLongAlloc := defaultRules.LongGasPower.AllocPerSec * 1000
	if rules.LongGasPower.AllocPerSec != expectedLongAlloc {
		t.Errorf("LongGasPower.AllocPerSec = %d, want %d",
			rules.LongGasPower.AllocPerSec, expectedLongAlloc)
	}
}

// TestRulesCopy verifies that Copy() creates a deep copy, especially for pointer types.
// This is critical because Rules contains *big.Int which would be shared in a shallow copy.
func TestRulesCopy(t *testing.T) {
	original := MainNetRules()

	// Modify the original's MinGasPrice
	original.Economy.MinGasPrice.Set(big.NewInt(999999))

	// Create a copy
	copied := original.Copy()

	// Modify the copy's MinGasPrice
	copied.Economy.MinGasPrice.Set(big.NewInt(123456))

	// Original should not be affected (deep copy)
	if original.Economy.MinGasPrice.Cmp(big.NewInt(999999)) != 0 {
		t.Errorf("Original MinGasPrice was modified: got %s, want 999999",
			original.Economy.MinGasPrice.String())
	}

	// Copy should have the new value
	if copied.Economy.MinGasPrice.Cmp(big.NewInt(123456)) != 0 {
		t.Errorf("Copied MinGasPrice = %s, want 123456",
			copied.Economy.MinGasPrice.String())
	}

	// Verify they are different pointers
	if original.Economy.MinGasPrice == copied.Economy.MinGasPrice {
		t.Error("MinGasPrice pointers should be different (deep copy)")
	}
}

// TestRulesString verifies that String() returns valid JSON.
func TestRulesString(t *testing.T) {
	rules := MainNetRules()
	jsonStr := rules.String()

	// Verify it's valid JSON by unmarshaling
	var unmarshaled Rules
	if err := json.Unmarshal([]byte(jsonStr), &unmarshaled); err != nil {
		t.Fatalf("String() returned invalid JSON: %v\nJSON: %s", err, jsonStr)
	}

	// Verify key fields are present
	if unmarshaled.Name != rules.Name {
		t.Errorf("Unmarshaled Name = %q, want %q", unmarshaled.Name, rules.Name)
	}
	if unmarshaled.NetworkID != rules.NetworkID {
		t.Errorf("Unmarshaled NetworkID = %d, want %d", unmarshaled.NetworkID, rules.NetworkID)
	}
}

// TestEvmChainConfig verifies that EvmChainConfig correctly converts Rules to Ethereum ChainConfig.
func TestEvmChainConfig(t *testing.T) {
	rules := MainNetRules()

	// Test with no upgrades (empty upgrade heights)
	cfg := rules.EvmChainConfig([]UpgradeHeight{})

	if cfg.ChainID.Cmp(big.NewInt(int64(MainNetworkID))) != 0 {
		t.Errorf("ChainID = %s, want %d", cfg.ChainID.String(), MainNetworkID)
	}

	if cfg.BerlinBlock != nil {
		t.Error("BerlinBlock should be nil when no upgrades specified")
	}
	if cfg.LondonBlock != nil {
		t.Error("LondonBlock should be nil when no upgrades specified")
	}
}

// TestEvmChainConfig_WithUpgrades verifies EvmChainConfig with upgrade heights.
func TestEvmChainConfig_WithUpgrades(t *testing.T) {
	rules := MainNetRules()

	tests := []struct {
		name           string
		upgradeHeights []UpgradeHeight
		wantBerlin     *big.Int
		wantLondon     *big.Int
	}{
		{
			name: "Berlin at genesis",
			upgradeHeights: []UpgradeHeight{
				{Upgrades: Upgrades{Berlin: true}, Height: 0},
			},
			wantBerlin: big.NewInt(0),
			wantLondon: nil,
		},
		{
			name: "Berlin and London at different heights",
			upgradeHeights: []UpgradeHeight{
				{Upgrades: Upgrades{Berlin: true}, Height: 0},
				{Upgrades: Upgrades{Berlin: true, London: true}, Height: 1000},
			},
			wantBerlin: big.NewInt(0),
			wantLondon: big.NewInt(1000),
		},
		{
			name: "London only at height 5000",
			upgradeHeights: []UpgradeHeight{
				{Upgrades: Upgrades{}, Height: 0},
				{Upgrades: Upgrades{London: true}, Height: 5000},
			},
			wantBerlin: nil,
			wantLondon: big.NewInt(5000),
		},
		{
			name: "Upgrades disabled after being enabled",
			upgradeHeights: []UpgradeHeight{
				{Upgrades: Upgrades{Berlin: true}, Height: 0},
				{Upgrades: Upgrades{Berlin: false}, Height: 1000},
			},
			wantBerlin: nil, // Should be cleared when disabled
			wantLondon: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := rules.EvmChainConfig(tt.upgradeHeights)

			// Verify Berlin block
			if tt.wantBerlin == nil {
				if cfg.BerlinBlock != nil {
					t.Errorf("BerlinBlock = %s, want nil", cfg.BerlinBlock.String())
				}
			} else {
				if cfg.BerlinBlock == nil {
					t.Fatal("BerlinBlock is nil, but should be set")
				}
				if cfg.BerlinBlock.Cmp(tt.wantBerlin) != 0 {
					t.Errorf("BerlinBlock = %s, want %s",
						cfg.BerlinBlock.String(), tt.wantBerlin.String())
				}
			}

			// Verify London block
			if tt.wantLondon == nil {
				if cfg.LondonBlock != nil {
					t.Errorf("LondonBlock = %s, want nil", cfg.LondonBlock.String())
				}
			} else {
				if cfg.LondonBlock == nil {
					t.Fatal("LondonBlock is nil, but should be set")
				}
				if cfg.LondonBlock.Cmp(tt.wantLondon) != 0 {
					t.Errorf("LondonBlock = %s, want %s",
						cfg.LondonBlock.String(), tt.wantLondon.String())
				}
			}
		})
	}
}

// TestEvmChainConfig_NetworkIDs verifies that different network IDs produce correct chain IDs.
func TestEvmChainConfig_NetworkIDs(t *testing.T) {
	tests := []struct {
		name      string
		rulesFunc func() Rules
		wantID    uint64
	}{
		{"MainNet", MainNetRules, MainNetworkID},
		{"TestNet", TestNetRules, TestNetworkID},
		{"FakeNet", FakeNetRules, FakeNetworkID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := tt.rulesFunc()
			cfg := rules.EvmChainConfig([]UpgradeHeight{})

			if cfg.ChainID.Cmp(big.NewInt(int64(tt.wantID))) != 0 {
				t.Errorf("ChainID = %s, want %d", cfg.ChainID.String(), tt.wantID)
			}
		})
	}
}

// TestRulesComparison verifies that different network rules have expected differences.
func TestRulesComparison(t *testing.T) {
	mainRules := MainNetRules()
	testRules := TestNetRules()
	fakeRules := FakeNetRules()

	// MainNet and TestNet should have same block config
	if mainRules.Blocks.MaxBlockGas != testRules.Blocks.MaxBlockGas {
		t.Error("MainNet and TestNet should have same MaxBlockGas")
	}

	// FakeNet should have shorter empty block skip period
	if fakeRules.Blocks.MaxEmptyBlockSkipPeriod >= mainRules.Blocks.MaxEmptyBlockSkipPeriod {
		t.Error("FakeNet should have shorter MaxEmptyBlockSkipPeriod than MainNet")
	}

	// FakeNet should have accelerated epochs
	fakeEpochs := FakeNetEpochsRules()
	mainEpochs := DefaultEpochsRules()
	if fakeEpochs.MaxEpochGas >= mainEpochs.MaxEpochGas {
		t.Error("FakeNet should have lower MaxEpochGas than MainNet")
	}
	if fakeEpochs.MaxEpochDuration >= mainEpochs.MaxEpochDuration {
		t.Error("FakeNet should have shorter MaxEpochDuration than MainNet")
	}

	// FakeNet should have accelerated gas power
	fakeEconomy := FakeEconomyRules()
	mainEconomy := DefaultEconomyRules()
	if fakeEconomy.ShortGasPower.AllocPerSec <= mainEconomy.ShortGasPower.AllocPerSec {
		t.Error("FakeNet should have faster ShortGasPower.AllocPerSec than MainNet")
	}
	if fakeEconomy.LongGasPower.AllocPerSec <= mainEconomy.LongGasPower.AllocPerSec {
		t.Error("FakeNet should have faster LongGasPower.AllocPerSec than MainNet")
	}
}

// TestRulesRLPStructure verifies that RulesRLP can be used as Rules (type alias).
func TestRulesRLPStructure(t *testing.T) {
	// Rules is a type alias for RulesRLP, so they should be interchangeable
	rulesRLP := RulesRLP{
		Name:      "test",
		NetworkID: 12345,
		Dag:       DefaultDagRules(),
		Epochs:    DefaultEpochsRules(),
		Blocks:    BlocksRules{MaxBlockGas: 1000000},
		Economy:   DefaultEconomyRules(),
		Upgrades:  Upgrades{Berlin: true},
	}
	rules := Rules(rulesRLP)

	if rules.Name != "test" {
		t.Errorf("Name = %q, want %q", rules.Name, "test")
	}
	if rules.NetworkID != 12345 {
		t.Errorf("NetworkID = %d, want %d", rules.NetworkID, 12345)
	}
	if rules.Upgrades.Berlin != true {
		t.Error("Berlin upgrade should be enabled")
	}
}

// TestGasRulesTypeAlias verifies that GasRules is correctly aliased to GasRulesRLPV1.
func TestGasRulesTypeAlias(t *testing.T) {
	// GasRules should have all fields from GasRulesRLPV1
	rules := DefaultGasRules()

	// Verify all V1 fields are accessible
	_ = rules.MaxEventGas
	_ = rules.EventGas
	_ = rules.ParentGas
	_ = rules.ExtraDataGas
	_ = rules.BlockVotesBaseGas
	_ = rules.BlockVoteGas
	_ = rules.EpochVoteGas
	_ = rules.MisbehaviourProofGas
}

// TestUpgradesStructure verifies that Upgrades struct works correctly.
func TestUpgradesStructure(t *testing.T) {
	upgrades := Upgrades{
		Berlin: true,
		London: false,
		Llr:    true,
	}

	if !upgrades.Berlin {
		t.Error("Berlin should be true")
	}
	if upgrades.London {
		t.Error("London should be false")
	}
	if !upgrades.Llr {
		t.Error("Llr should be true")
	}
}

// TestUpgradeHeightStructure verifies that UpgradeHeight struct works correctly.
func TestUpgradeHeightStructure(t *testing.T) {
	height := UpgradeHeight{
		Upgrades: Upgrades{Berlin: true, London: true},
		Height:   1000,
	}

	if !height.Upgrades.Berlin {
		t.Error("Berlin should be enabled")
	}
	if !height.Upgrades.London {
		t.Error("London should be enabled")
	}
	if height.Height != 1000 {
		t.Errorf("Height = %d, want %d", height.Height, 1000)
	}
}
