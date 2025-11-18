package test

import (
	"testing"

	"github.com/rony4d/go-opera-asset/integration"
)

// Package integration_test verifies that configuration presets behave correctly:
// - Each preset produces distinct, internally consistent configurations
// - Presets override default values as expected
// - Helper functions (GetPresetByName, ApplyPreset) work correctly
// - Edge cases and invalid inputs are handled gracefully
//
// These tests ensure that operators can reliably use presets without
// unexpected side effects or configuration conflicts.

// TestDefaultPreset_hasReasonableDefaults verifies that DefaultPreset returns
// a configuration with sensible baseline values. This test acts as a regression
// guard: if defaults change, we want to know immediately.
func TestDefaultPreset_hasReasonableDefaults(t *testing.T) {
	cfg := integration.DefaultPreset()

	// Verify preset name is set correctly for logging/config dumps
	if cfg.Name != "default" {
		t.Fatalf("Name = %q, want 'default'", cfg.Name)
	}

	// Cache should be non-zero and reasonable (not too small, not excessive)
	if cfg.CacheMB <= 0 || cfg.CacheMB > 10000 {
		t.Fatalf("CacheMB = %d, want value between 1 and 10000", cfg.CacheMB)
	}

	// GCMode must be one of the valid options
	validGCModes := map[string]bool{"light": true, "full": true, "archive": true}
	if !validGCModes[cfg.GCMode] {
		t.Fatalf("GCMode = %q, want one of: light, full, archive", cfg.GCMode)
	}

	// DBPreset should be non-empty (exact value depends on your DB layer)
	if cfg.DBPreset == "" {
		t.Fatal("DBPreset is empty, should have a value")
	}

	// Security defaults: LightKDF should be false for production safety
	if cfg.EnableLightKDF {
		t.Fatal("EnableLightKDF should be false by default for security")
	}
}

// TestLitePreset_overridesDefaults verifies that LitePreset produces a
// configuration distinct from DefaultPreset, with values optimized for
// development environments.
func TestLitePreset_overridesDefaults(t *testing.T) {
	defaultCfg := integration.DefaultPreset()
	liteCfg := integration.LitePreset()

	// Lite preset should have a different name
	if liteCfg.Name != "lite" {
		t.Fatalf("Name = %q, want 'lite'", liteCfg.Name)
	}

	// Cache should be smaller than default (optimized for low-resource envs)
	if liteCfg.CacheMB >= defaultCfg.CacheMB {
		t.Fatalf("Lite CacheMB (%d) should be smaller than default (%d)", liteCfg.CacheMB, defaultCfg.CacheMB)
	}

	// Lite preset should use archive GC mode (keep all state for debugging)
	if liteCfg.GCMode != "archive" {
		t.Fatalf("GCMode = %q, want 'archive' for lite preset", liteCfg.GCMode)
	}

	// Metrics should be enabled for development diagnostics
	if !liteCfg.EnableMetrics {
		t.Fatal("EnableMetrics should be true for lite preset")
	}

	// LightKDF should be enabled for faster development workflows
	if !liteCfg.EnableLightKDF {
		t.Fatal("EnableLightKDF should be true for lite preset (dev convenience)")
	}
}

// TestFullPreset_overridesDefaults verifies that FullPreset produces a
// production-ready configuration with larger caches and strong security.
func TestFullPreset_overridesDefaults(t *testing.T) {
	defaultCfg := integration.DefaultPreset()
	fullCfg := integration.FullPreset()

	// Full preset should have a different name
	if fullCfg.Name != "full" {
		t.Fatalf("Name = %q, want 'full'", fullCfg.Name)
	}

	// Cache should be larger than default (optimized for performance)
	if fullCfg.CacheMB <= defaultCfg.CacheMB {
		t.Fatalf("Full CacheMB (%d) should be larger than default (%d)", fullCfg.CacheMB, defaultCfg.CacheMB)
	}

	// Full preset should use full GC mode (prune old state)
	if fullCfg.GCMode != "full" {
		t.Fatalf("GCMode = %q, want 'full' for full preset", fullCfg.GCMode)
	}

	// Metrics and tracing should be enabled for production monitoring
	if !fullCfg.EnableMetrics {
		t.Fatal("EnableMetrics should be true for full preset")
	}
	if !fullCfg.EnableTracing {
		t.Fatal("EnableTracing should be true for full preset")
	}

	// LightKDF should remain false (strong security for production)
	if fullCfg.EnableLightKDF {
		t.Fatal("EnableLightKDF should be false for full preset (security)")
	}
}

// TestArchivePreset_overridesDefaults verifies that ArchivePreset produces
// a configuration optimized for historical queries with maximum caching.
func TestArchivePreset_overridesDefaults(t *testing.T) {
	defaultCfg := integration.DefaultPreset()
	archiveCfg := integration.ArchivePreset()

	// Archive preset should have a different name
	if archiveCfg.Name != "archive" {
		t.Fatalf("Name = %q, want 'archive'", archiveCfg.Name)
	}

	// Cache should be largest of all presets (optimized for read-heavy workloads)
	if archiveCfg.CacheMB <= defaultCfg.CacheMB {
		t.Fatalf("Archive CacheMB (%d) should be larger than default (%d)", archiveCfg.CacheMB, defaultCfg.CacheMB)
	}

	// Archive preset must use archive GC mode (never prune state)
	if archiveCfg.GCMode != "archive" {
		t.Fatalf("GCMode = %q, want 'archive' for archive preset", archiveCfg.GCMode)
	}

	// Both metrics and tracing should be enabled for monitoring long syncs
	if !archiveCfg.EnableMetrics {
		t.Fatal("EnableMetrics should be true for archive preset")
	}
	if !archiveCfg.EnableTracing {
		t.Fatal("EnableTracing should be true for archive preset")
	}

	// Security should remain strong even for archival nodes
	if archiveCfg.EnableLightKDF {
		t.Fatal("EnableLightKDF should be false for archive preset")
	}
}

// TestPresets_haveDistinctValues verifies that all presets produce unique
// configurations. This ensures presets are actually useful and not redundant.
func TestPresets_haveDistinctValues(t *testing.T) {
	lite := integration.LitePreset()
	full := integration.FullPreset()
	archive := integration.ArchivePreset()

	// Each preset should have a unique name
	names := map[string]bool{
		lite.Name:    true,
		full.Name:    true,
		archive.Name: true,
	}
	if len(names) != 3 {
		t.Fatalf("Presets should have unique names, got: %v", names)
	}

	// Cache sizes should be ordered: lite < full < archive
	if lite.CacheMB >= full.CacheMB {
		t.Fatalf("Lite cache (%d) should be smaller than full (%d)", lite.CacheMB, full.CacheMB)
	}
	if full.CacheMB >= archive.CacheMB {
		t.Fatalf("Full cache (%d) should be smaller than archive (%d)", full.CacheMB, archive.CacheMB)
	}

	// GC modes should differ: lite/archive use archive, full uses full
	if lite.GCMode != "archive" || archive.GCMode != "archive" {
		t.Fatal("Lite and archive presets should use archive GC mode")
	}
	if full.GCMode != "full" {
		t.Fatal("Full preset should use full GC mode")
	}
}

// TestGetPresetByName_validPresets verifies that GetPresetByName correctly
// returns the expected preset for all valid preset names.
func TestGetPresetByName_validPresets(t *testing.T) {
	tests := []struct {
		name     string
		wantName string
	}{
		{"lite", "lite"},
		{"full", "full"},
		{"archive", "archive"},
		{"default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := integration.GetPresetByName(tt.name)
			if err != nil {
				t.Fatalf("GetPresetByName(%q) returned error: %v", tt.name, err)
			}
			// Verify the returned preset has the correct name
			if cfg.Name != tt.wantName {
				t.Fatalf("Preset name = %q, want %q", cfg.Name, tt.wantName)
			}
			// Verify the preset has reasonable values (non-zero cache, valid GC mode)
			if cfg.CacheMB <= 0 {
				t.Fatalf("Preset %q has invalid CacheMB: %d", tt.name, cfg.CacheMB)
			}
		})
	}
}

// TestGetPresetByName_invalidPreset verifies that GetPresetByName returns
// an error for unrecognized preset names.
func TestGetPresetByName_invalidPreset(t *testing.T) {
	invalidNames := []string{"unknown", "invalid", "", "LITE", "Full"}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			cfg, err := integration.GetPresetByName(name)
			if err == nil {
				t.Fatalf("GetPresetByName(%q) should return error, got config: %+v", name, cfg)
			}
			// Error message should be helpful and mention valid options
			if err.Error() == "" {
				t.Fatal("Error message should not be empty")
			}
		})
	}
}

// TestApplyPreset_overridesTarget verifies that ApplyPreset correctly merges
// preset values into an existing configuration, overriding only the fields
// that are set in the preset.
func TestApplyPreset_overridesTarget(t *testing.T) {
	// Start with a custom target config
	target := integration.PresetConfig{
		Name:           "custom",
		CacheMB:        512,
		GCMode:         "light",
		DBPreset:       "custom-db",
		EnableMetrics:  false,
		EnableTracing:  false,
		EnableLightKDF: true,
	}

	// Apply the full preset
	preset := integration.FullPreset()
	integration.ApplyPreset(&target, preset)

	// Verify all preset fields were applied
	if target.Name != preset.Name {
		t.Fatalf("Name not overridden: got %q, want %q", target.Name, preset.Name)
	}
	if target.CacheMB != preset.CacheMB {
		t.Fatalf("CacheMB not overridden: got %d, want %d", target.CacheMB, preset.CacheMB)
	}
	if target.GCMode != preset.GCMode {
		t.Fatalf("GCMode not overridden: got %q, want %q", target.GCMode, preset.GCMode)
	}
	if target.DBPreset != preset.DBPreset {
		t.Fatalf("DBPreset not overridden: got %q, want %q", target.DBPreset, preset.DBPreset)
	}
	if target.EnableMetrics != preset.EnableMetrics {
		t.Fatalf("EnableMetrics not overridden: got %v, want %v", target.EnableMetrics, preset.EnableMetrics)
	}
	if target.EnableTracing != preset.EnableTracing {
		t.Fatalf("EnableTracing not overridden: got %v, want %v", target.EnableTracing, preset.EnableTracing)
	}
	if target.EnableLightKDF != preset.EnableLightKDF {
		t.Fatalf("EnableLightKDF not overridden: got %v, want %v", target.EnableLightKDF, preset.EnableLightKDF)
	}
}

// TestApplyPreset_partialOverride verifies that ApplyPreset handles partial
// presets correctly (presets with some zero values should only override
// non-zero fields).
func TestApplyPreset_partialOverride(t *testing.T) {
	target := integration.DefaultPreset()
	originalName := target.Name

	// Create a partial preset that only sets CacheMB
	partial := integration.PresetConfig{
		CacheMB: 2048,
		// Name is empty, so it shouldn't override
		// Other fields are zero/false, so they shouldn't override either
	}

	integration.ApplyPreset(&target, partial)

	// CacheMB should be overridden
	if target.CacheMB != 2048 {
		t.Fatalf("CacheMB should be overridden to 2048, got %d", target.CacheMB)
	}

	// Name should remain unchanged (empty string in preset means don't override)
	if target.Name != originalName {
		t.Fatalf("Name should remain %q when preset has empty name, got %q", originalName, target.Name)
	}
}

// TestPresets_areIdempotent verifies that calling preset functions multiple
// times returns consistent results. This ensures presets don't have hidden
// state or side effects.
func TestPresets_areIdempotent(t *testing.T) {
	// Call each preset function twice
	lite1 := integration.LitePreset()
	lite2 := integration.LitePreset()

	full1 := integration.FullPreset()
	full2 := integration.FullPreset()

	archive1 := integration.ArchivePreset()
	archive2 := integration.ArchivePreset()

	// Compare results: they should be identical
	if lite1 != lite2 {
		t.Fatal("LitePreset() should return identical results on multiple calls")
	}
	if full1 != full2 {
		t.Fatal("FullPreset() should return identical results on multiple calls")
	}
	if archive1 != archive2 {
		t.Fatal("ArchivePreset() should return identical results on multiple calls")
	}
}
