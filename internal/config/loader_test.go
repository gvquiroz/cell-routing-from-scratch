package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoader_LoadInitial_Success(t *testing.T) {
	tmpFile := t.TempDir() + "/config.json"
	validConfig := `{
		"version": "v1",
		"routingTable": {"acme": "tier1"},
		"cellEndpoints": {"tier1": "http://cell-tier1:9001"},
		"defaultPlacement": "tier1"
	}`

	if err := os.WriteFile(tmpFile, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tmpFile, 1*time.Second)
	if err := loader.LoadInitial(); err != nil {
		t.Fatalf("LoadInitial failed: %v", err)
	}

	cfg := loader.GetConfig()
	if cfg.Version != "v1" {
		t.Errorf("Version = %v, want v1", cfg.Version)
	}
}

func TestLoader_LoadInitial_InvalidConfig(t *testing.T) {
	tmpFile := t.TempDir() + "/config.json"
	invalidConfig := `{
		"version": "",
		"routingTable": {"acme": "tier1"},
		"cellEndpoints": {"tier1": "http://cell-tier1:9001"},
		"defaultPlacement": "tier1"
	}`

	if err := os.WriteFile(tmpFile, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tmpFile, 1*time.Second)
	err := loader.LoadInitial()
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("Error should mention 'version', got: %v", err)
	}
}

func TestLoader_LoadInitial_MissingFile(t *testing.T) {
	loader := NewLoader("/nonexistent/path/config.json", 1*time.Second)
	err := loader.LoadInitial()
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestLoader_HotReload(t *testing.T) {
	tmpFile := t.TempDir() + "/config.json"
	initialConfig := `{
		"version": "v1",
		"routingTable": {"acme": "tier1"},
		"cellEndpoints": {"tier1": "http://cell-tier1:9001"},
		"defaultPlacement": "tier1"
	}`

	if err := os.WriteFile(tmpFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tmpFile, 100*time.Millisecond)
	if err := loader.LoadInitial(); err != nil {
		t.Fatalf("LoadInitial failed: %v", err)
	}

	loader.StartReloadLoop()
	defer loader.Stop()

	// Verify initial config
	cfg := loader.GetConfig()
	if cfg.Version != "v1" {
		t.Errorf("Initial version = %v, want v1", cfg.Version)
	}

	// Update config file
	updatedConfig := `{
		"version": "v2",
		"routingTable": {"acme": "tier1", "globex": "tier2"},
		"cellEndpoints": {
			"tier1": "http://cell-tier1:9001",
			"tier2": "http://cell-tier2:9002"
		},
		"defaultPlacement": "tier1"
	}`

	// Wait a bit to ensure different timestamp
	time.Sleep(50 * time.Millisecond)

	if err := os.WriteFile(tmpFile, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Wait for reload to happen
	time.Sleep(300 * time.Millisecond)

	// Verify config was reloaded
	cfg = loader.GetConfig()
	if cfg.Version != "v2" {
		t.Errorf("After reload, version = %v, want v2", cfg.Version)
	}
	if cfg.RoutingTable["globex"] != "tier2" {
		t.Errorf("After reload, routingTable[globex] = %v, want tier2", cfg.RoutingTable["globex"])
	}
}

func TestLoader_HotReload_KeepsLastKnownGood(t *testing.T) {
	tmpFile := t.TempDir() + "/config.json"
	initialConfig := `{
		"version": "v1",
		"routingTable": {"acme": "tier1"},
		"cellEndpoints": {"tier1": "http://cell-tier1:9001"},
		"defaultPlacement": "tier1"
	}`

	if err := os.WriteFile(tmpFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tmpFile, 100*time.Millisecond)
	if err := loader.LoadInitial(); err != nil {
		t.Fatalf("LoadInitial failed: %v", err)
	}

	loader.StartReloadLoop()
	defer loader.Stop()

	// Verify initial config
	cfg := loader.GetConfig()
	if cfg.Version != "v1" {
		t.Errorf("Initial version = %v, want v1", cfg.Version)
	}

	// Write invalid config
	invalidConfig := `{
		"version": "",
		"routingTable": {"acme": "tier1"},
		"cellEndpoints": {"tier1": "http://cell-tier1:9001"},
		"defaultPlacement": "tier1"
	}`

	time.Sleep(50 * time.Millisecond)

	if err := os.WriteFile(tmpFile, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Wait for reload attempt
	time.Sleep(300 * time.Millisecond)

	// Verify config was NOT updated (kept last-known-good)
	cfg = loader.GetConfig()
	if cfg.Version != "v1" {
		t.Errorf("After invalid reload, version = %v, want v1 (last-known-good)", cfg.Version)
	}
}
