package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"time"
)

// HealthCheckConfig configures health checking for an endpoint
type HealthCheckConfig struct {
	Path     string `json:"path"`
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
}

// ParsedHealthCheckConfig contains parsed duration values
type ParsedHealthCheckConfig struct {
	Path     string
	Interval time.Duration
	Timeout  time.Duration
}

// Parse converts string durations to time.Duration
func (h *HealthCheckConfig) Parse() (*ParsedHealthCheckConfig, error) {
	interval, err := time.ParseDuration(h.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid health check interval: %w", err)
	}

	timeout, err := time.ParseDuration(h.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid health check timeout: %w", err)
	}

	return &ParsedHealthCheckConfig{
		Path:     h.Path,
		Interval: interval,
		Timeout:  timeout,
	}, nil
}

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	FailureThreshold int    `json:"failure_threshold"`
	Timeout          string `json:"timeout"`
}

// ParsedCircuitBreakerConfig contains parsed duration values
type ParsedCircuitBreakerConfig struct {
	FailureThreshold uint32
	Timeout          time.Duration
}

// Parse converts string duration to time.Duration
func (c *CircuitBreakerConfig) Parse() (*ParsedCircuitBreakerConfig, error) {
	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid circuit breaker timeout: %w", err)
	}

	return &ParsedCircuitBreakerConfig{
		FailureThreshold: uint32(c.FailureThreshold),
		Timeout:          timeout,
	}, nil
}

// PlacementConfig contains resilience configuration for a placement
type PlacementConfig struct {
	URL                 string                `json:"url"`
	Fallback            string                `json:"fallback,omitempty"`
	HealthCheck         *HealthCheckConfig    `json:"health_check,omitempty"`
	CircuitBreaker      *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	ConcurrencyLimit    int                   `json:"concurrency_limit,omitempty"`
	MaxRequestBodyBytes int64                 `json:"max_request_body_bytes,omitempty"`
}

// Config represents the routing configuration
type Config struct {
	Version          string                      `json:"version"`
	RoutingTable     map[string]string           `json:"routingTable"`
	CellEndpoints    map[string]string           `json:"cellEndpoints,omitempty"` // Legacy format
	Placements       map[string]*PlacementConfig `json:"placements,omitempty"`    // New format
	DefaultPlacement string                      `json:"defaultPlacement"`
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
// Supports both legacy and new formats
func (c *Config) GetCellEndpoints() map[string]string {
	if len(c.CellEndpoints) > 0 {
		// Legacy format
		return c.CellEndpoints
	}

	// New format: extract URLs from placements
	endpoints := make(map[string]string)
	for key, placement := range c.Placements {
		endpoints[key] = placement.URL
	}
	return endpoints
}

// GetDefaultPlacement implements routing.ConfigProvider
func (c *Config) GetDefaultPlacement() string {
	return c.DefaultPlacement
}

// GetPlacementConfig returns the placement configuration
func (c *Config) GetPlacementConfig(placementKey string) (*PlacementConfig, bool) {
	if c.Placements == nil {
		return nil, false
	}
	placement, exists := c.Placements[placementKey]
	return placement, exists
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

	// Get endpoints (supports both formats)
	endpoints := c.GetCellEndpoints()

	// DefaultPlacement must exist in endpoints
	if _, exists := endpoints[c.DefaultPlacement]; !exists {
		return fmt.Errorf("defaultPlacement '%s' not found in endpoints", c.DefaultPlacement)
	}

	// All placements in routingTable must exist in endpoints
	for routingKey, placementKey := range c.RoutingTable {
		if _, exists := endpoints[placementKey]; !exists {
			return fmt.Errorf("routingTable[%s] references unknown placement '%s'", routingKey, placementKey)
		}
	}

	// All endpoint URLs must be valid
	for placement, endpointURL := range endpoints {
		if _, err := url.Parse(endpointURL); err != nil {
			return fmt.Errorf("invalid URL for placement '%s': %w", placement, err)
		}
	}

	// Validate fallback references
	if c.Placements != nil {
		for placementKey, placement := range c.Placements {
			if placement.Fallback != "" {
				if _, exists := endpoints[placement.Fallback]; !exists {
					return fmt.Errorf("placement '%s' references unknown fallback '%s'", placementKey, placement.Fallback)
				}
			}

			// Validate health check config
			if placement.HealthCheck != nil {
				if _, err := placement.HealthCheck.Parse(); err != nil {
					return fmt.Errorf("placement '%s': %w", placementKey, err)
				}
			}

			// Validate circuit breaker config
			if placement.CircuitBreaker != nil {
				if _, err := placement.CircuitBreaker.Parse(); err != nil {
					return fmt.Errorf("placement '%s': %w", placementKey, err)
				}
			}
		}
	}

	return nil
}
