package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
)

// Config represents the routing configuration
type Config struct {
	Version          string            `json:"version"`
	RoutingTable     map[string]string `json:"routingTable"`
	CellEndpoints    map[string]string `json:"cellEndpoints"`
	DefaultPlacement string            `json:"defaultPlacement"`
}

// GetVersion returns the config version
func (c *Config) GetVersion() string {
	return c.Version
}

// GetRoutingTable implements routing.ConfigProvider
func (c *Config) GetRoutingTable() map[string]string {
	return c.RoutingTable
}

// GetCellEndpoints implements routing.ConfigProvider
func (c *Config) GetCellEndpoints() map[string]string {
	return c.CellEndpoints
}

// GetDefaultPlacement implements routing.ConfigProvider
func (c *Config) GetDefaultPlacement() string {
	return c.DefaultPlacement
}

// LoadFromFile reads and parses a config file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	// Version must be present
	if c.Version == "" {
		return fmt.Errorf("version must be non-empty")
	}

	// DefaultPlacement must exist in cellEndpoints
	if _, exists := c.CellEndpoints[c.DefaultPlacement]; !exists {
		return fmt.Errorf("defaultPlacement '%s' not found in cellEndpoints", c.DefaultPlacement)
	}

	// All placements in routingTable must exist in cellEndpoints
	for routingKey, placementKey := range c.RoutingTable {
		if _, exists := c.CellEndpoints[placementKey]; !exists {
			return fmt.Errorf("routingTable[%s] references unknown placement '%s'", routingKey, placementKey)
		}
	}

	// All endpoint URLs must be valid
	for placement, endpointURL := range c.CellEndpoints {
		if _, err := url.Parse(endpointURL); err != nil {
			return fmt.Errorf("invalid URL for placement '%s': %w", placement, err)
		}
	}

	return nil
}
