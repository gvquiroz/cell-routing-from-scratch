package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	tmpFile := t.TempDir() + "/config.json"
	validConfig := `{
		"version": "v1",
		"routingTable": {
			"acme": "tier1",
			"visa": "visa"
		},
		"cellEndpoints": {
			"tier1": "http://cell-tier1:9001",
			"visa": "http://cell-visa:9004"
		},
		"defaultPlacement": "tier1"
	}`

	if err := os.WriteFile(tmpFile, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.Version != "v1" {
		t.Errorf("Version = %v, want v1", cfg.Version)
	}
	if cfg.RoutingTable["acme"] != "tier1" {
		t.Errorf("RoutingTable[acme] = %v, want tier1", cfg.RoutingTable["acme"])
	}
	if cfg.DefaultPlacement != "tier1" {
		t.Errorf("DefaultPlacement = %v, want tier1", cfg.DefaultPlacement)
	}
}

func TestLoadFromFile_Missing(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tmpFile := t.TempDir() + "/config.json"
	if err := os.WriteFile(tmpFile, []byte("invalid json{"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadFromFile(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		Version: "v1",
		RoutingTable: map[string]string{
			"acme": "tier1",
			"visa": "visa",
		},
		CellEndpoints: map[string]string{
			"tier1": "http://cell-tier1:9001",
			"visa":  "http://cell-visa:9004",
		},
		DefaultPlacement: "tier1",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	cfg := &Config{
		Version: "",
		RoutingTable: map[string]string{
			"acme": "tier1",
		},
		CellEndpoints: map[string]string{
			"tier1": "http://cell-tier1:9001",
		},
		DefaultPlacement: "tier1",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for missing version, got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("Error should mention 'version', got: %v", err)
	}
}

func TestValidate_DefaultPlacementNotInEndpoints(t *testing.T) {
	cfg := &Config{
		Version:          "v1",
		RoutingTable:     map[string]string{"acme": "tier1"},
		CellEndpoints:    map[string]string{"tier1": "http://cell-tier1:9001"},
		DefaultPlacement: "tier3",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for missing defaultPlacement, got nil")
	}
	if !strings.Contains(err.Error(), "defaultPlacement") {
		t.Errorf("Error should mention 'defaultPlacement', got: %v", err)
	}
}

func TestValidate_RoutingTableReferencesUnknownPlacement(t *testing.T) {
	cfg := &Config{
		Version: "v1",
		RoutingTable: map[string]string{
			"acme":   "tier1",
			"orphan": "unknown-placement",
		},
		CellEndpoints: map[string]string{
			"tier1": "http://cell-tier1:9001",
		},
		DefaultPlacement: "tier1",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for unknown placement, got nil")
	}
	if !strings.Contains(err.Error(), "unknown placement") {
		t.Errorf("Error should mention 'unknown placement', got: %v", err)
	}
}

func TestValidate_InvalidURL(t *testing.T) {
	cfg := &Config{
		Version:      "v1",
		RoutingTable: map[string]string{"acme": "tier1"},
		CellEndpoints: map[string]string{
			"tier1": "://invalid-url",
		},
		DefaultPlacement: "tier1",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("Error should mention 'invalid URL', got: %v", err)
	}
}
