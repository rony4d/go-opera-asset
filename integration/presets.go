package integration

import "fmt"

// Package integration provides configuration presets and assembly helpers for
// building the Opera node runtime. Presets bundle common settings (cache sizes,
// DB layouts, GC modes) into named profiles (Lite, Full, Archive) so operators
// can quickly spin up nodes optimized for different workloads without tweaking
// dozens of flags.
//
// Usage:
//   cfg := integration.LitePreset()  // for development
//   cfg := integration.FullPreset()  // for production validators
//   cfg := integration.ArchivePreset() // for chain explorers
//
// Each preset returns a PresetConfig struct that can be merged into the
// launcher's main config during node initialization.

// PresetConfig captures the tunable parameters that vary across preset profiles.
// It intentionally excludes fields that are always the same (like network IDs
// or RPC ports) so presets focus on performance and resource trade-offs.
type PresetConfig struct {
	Name           string // human-readable identifier (e.g., "lite", "full")
	CacheMB        int    // total memory allocated to internal caches (DB, state, etc.)
	GCMode         string // garbage collection strategy: "light", "full", "archive"
	DBPreset       string // database layout identifier (e.g., "ldb-1", "pbl-1")
	EnableMetrics  bool   // whether to expose Prometheus-style metrics endpoints
	EnableTracing  bool   // whether to enable distributed tracing (Jaeger, etc.)
	EnableLightKDF bool   // use faster (weaker) key derivation for keystore passwords
}

func DefaultPreset() PresetConfig {

	return PresetConfig{
		Name:           "default",
		CacheMB:        1024,    // 1GB cache: enough for moderate workloads
		GCMode:         "full",  // full pruning: reclaim disk space while keeping recent state
		DBPreset:       "ldb-1", // LevelDB-based layout optimized for write-heavy workloads
		EnableMetrics:  false,   // metrics disabled by default to reduce overhead
		EnableTracing:  false,   // tracing disabled by default (adds latency)
		EnableLightKDF: false,   // strong key derivation for production security
	}
}

// LitePreset returns a lightweight configuration optimized for development,
// testing, and low-resource environments. It trades durability and security
// for faster startup times and lower memory footprint.
//
// Use cases:
//   - Local development on laptops
//   - CI/CD pipelines with limited resources
//   - Quick network testing with disposable nodes
//
// Trade-offs:
//   - Smaller caches may slow down sync on large chains
//   - Light KDF weakens keystore security (never use for production keys)
//   - Archive GC mode keeps all state (useful for debugging, but uses more disk)
func LitePreset() PresetConfig {
	cfg := DefaultPreset()    // start with balanced defaults
	cfg.Name = "lite"         // set preset identifier for logging/config dumps
	cfg.CacheMB = 256         // reduce cache to 256MB so node fits in constrained environments
	cfg.GCMode = "archive"    // disable pruning: keep all historical state for debugging
	cfg.DBPreset = "lite"     // use minimal DB schema optimized for small datasets
	cfg.EnableMetrics = true  // enable metrics to help diagnose issues during development
	cfg.EnableLightKDF = true // faster key derivation speeds up account unlock during testing
	return cfg
}

// FullPreset returns a production-ready configuration optimized for validator
// nodes and high-performance full nodes. It maximizes caching and enables
// monitoring while maintaining strong security defaults.
//
// Use cases:
//   - Mainnet validator nodes
//   - Public RPC endpoints
//   - Nodes requiring maximum throughput
//
// Trade-offs:
//   - Large caches require significant RAM (4GB+ recommended)
//   - Full GC mode prunes old state to save disk (not suitable for archival queries)
//   - Metrics and tracing add small overhead but provide essential observability
func FullPreset() PresetConfig {
	cfg := DefaultPreset()
	cfg.Name = "full"
	cfg.CacheMB = 4096         // 4GB cache: large enough to keep hot state in memory
	cfg.GCMode = "full"        // aggressive pruning: reclaim disk space by removing old state
	cfg.DBPreset = "ldb-1"     // LevelDB layout tuned for durability and write performance
	cfg.EnableMetrics = true   // expose metrics for Prometheus/Grafana dashboards
	cfg.EnableTracing = true   // enable distributed tracing for production debugging
	cfg.EnableLightKDF = false // strong key derivation: critical for validator key security
	return cfg
}

// ArchivePreset returns a configuration optimized for chain explorers, analytics
// platforms, and nodes that need to query historical state. It disables pruning
// and maximizes caching to support fast lookups across the entire chain history.
//
// Use cases:
//   - Block explorers (Etherscan-style services)
//   - Analytics and reporting tools
//   - Nodes serving historical RPC queries
//
// Trade-offs:
//   - Very large caches require substantial RAM (8GB+ recommended)
//   - Archive GC mode never prunes state (disk usage grows linearly with chain length)
//   - Higher memory and disk costs compared to Full preset
func ArchivePreset() PresetConfig {
	cfg := DefaultPreset()
	cfg.Name = "archive"
	cfg.CacheMB = 8192         // 8GB cache: large enough to keep significant state in memory
	cfg.GCMode = "archive"     // never prune: retain complete state history for queries
	cfg.DBPreset = "pbl-1"     // PebbleDB layout optimized for read-heavy analytical workloads
	cfg.EnableMetrics = true   // metrics help monitor long-running archival sync jobs
	cfg.EnableTracing = true   // tracing aids debugging complex historical queries
	cfg.EnableLightKDF = false // maintain strong security even for archival nodes
	return cfg
}

// GetPresetByName looks up a preset by its string identifier and returns the
// corresponding PresetConfig. Returns an error if the name is unrecognized.
// This helper enables CLI flags like --preset=full to select configurations
// dynamically.
//
// Example:
//
//	preset, err := integration.GetPresetByName("lite")
//	if err != nil {
//	    log.Fatal(err)
//	}
func GetPresetByName(name string) (PresetConfig, error) {
	switch name {
	case "lite":
		return LitePreset(), nil
	case "full":
		return FullPreset(), nil
	case "archive":
		return ArchivePreset(), nil
	case "default":
		return DefaultPreset(), nil
	default:
		return PresetConfig{}, fmt.Errorf("unknown preset: %q (valid: lite, full, archive, default)", name)
	}
}

// ApplyPreset merges a preset configuration into an existing config struct.
// Fields set in the preset override the corresponding values in the target.
// This allows presets to be applied incrementally on top of CLI/config-file
// overrides without clobbering unrelated settings.
//
// Example:
//
//	cfg := launcher.DefaultConfig()
//	preset := integration.FullPreset()
//	integration.ApplyPreset(&cfg, preset)
func ApplyPreset(target *PresetConfig, preset PresetConfig) {
	if preset.CacheMB > 0 {
		target.CacheMB = preset.CacheMB
	}
	if preset.GCMode != "" {
		target.GCMode = preset.GCMode
	}
	if preset.DBPreset != "" {
		target.DBPreset = preset.DBPreset
	}
	// boolean flags are always applied (no zero-value check needed)
	target.EnableMetrics = preset.EnableMetrics
	target.EnableTracing = preset.EnableTracing
	target.EnableLightKDF = preset.EnableLightKDF
	if preset.Name != "" {
		target.Name = preset.Name
	}
}
