package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"
)

// ConfigSource indicates where the config came from
type ConfigSource string

const (
	SourceFile         ConfigSource = "file"
	SourceControlPlane ConfigSource = "control_plane"
)

// Loader manages hot-reloading of routing configuration
type Loader struct {
	configPath   string
	activeConfig atomic.Value // stores *Config
	configSource atomic.Value // stores ConfigSource
	lastChecksum atomic.Value // stores string
	lastReload   atomic.Value // stores time.Time
	pollInterval time.Duration
	stopChan     chan struct{}
}

// NewLoader creates a new config loader
func NewLoader(configPath string, pollInterval time.Duration) *Loader {
	return &Loader{
		configPath:   configPath,
		pollInterval: pollInterval,
		stopChan:     make(chan struct{}),
	}
}

// LoadInitial loads the config file at startup
// Returns error if config is invalid or missing
func (l *Loader) LoadInitial() error {
	cfg, err := LoadFromFile(l.configPath)
	if err != nil {
		return fmt.Errorf("failed to load initial config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid initial config: %w", err)
	}

	checksum, err := fileChecksum(l.configPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	l.activeConfig.Store(cfg)
	l.configSource.Store(SourceFile)
	l.lastChecksum.Store(checksum)
	l.lastReload.Store(time.Now())

	log.Printf("Loaded initial config version: %s", cfg.Version)
	return nil
}

// ApplyConfig atomically applies a config from the control plane
func (l *Loader) ApplyConfig(cfg *Config) error {
	l.activeConfig.Store(cfg)
	l.configSource.Store(SourceControlPlane)
	l.lastReload.Store(time.Now())
	return nil
}

// GetConfigSource returns the source of the current config
func (l *Loader) GetConfigSource() interface{} {
	v := l.configSource.Load()
	if v == nil {
		return SourceFile
	}
	return v.(ConfigSource)
}

// GetConfig returns the current active config (atomic read)
func (l *Loader) GetConfig() *Config {
	return l.activeConfig.Load().(*Config)
}

// GetConfigVersion returns the current config version for debug endpoint
func (l *Loader) GetConfigVersion() string {
	return l.GetConfig().Version
}

// GetRoutingTable implements routing.ConfigProvider
func (l *Loader) GetRoutingTable() map[string]string {
	return l.GetConfig().RoutingTable
}

// GetCellEndpoints implements routing.ConfigProvider
func (l *Loader) GetCellEndpoints() map[string]string {
	return l.GetConfig().CellEndpoints
}

// GetDefaultPlacement implements routing.ConfigProvider
func (l *Loader) GetDefaultPlacement() string {
	return l.GetConfig().DefaultPlacement
}

// LastReloadTime returns the timestamp of the last successful reload
func (l *Loader) LastReloadTime() time.Time {
	v := l.lastReload.Load()
	if v == nil {
		return time.Time{}
	}
	return v.(time.Time)
}

// StartReloadLoop starts a background goroutine that polls for config changes
func (l *Loader) StartReloadLoop() {
	go l.reloadLoop()
}

// Stop stops the reload loop
func (l *Loader) Stop() {
	close(l.stopChan)
}

// reloadLoop polls the config file for changes and reloads if needed
func (l *Loader) reloadLoop() {
	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.tryReload()
		case <-l.stopChan:
			return
		}
	}
}

// tryReload attempts to reload the config if it has changed
func (l *Loader) tryReload() {
	// Check if file has changed
	currentChecksum, err := fileChecksum(l.configPath)
	if err != nil {
		log.Printf("Config reload: failed to compute checksum: %v", err)
		return
	}

	lastChecksum := l.lastChecksum.Load().(string)
	if currentChecksum == lastChecksum {
		// No changes
		return
	}

	// File changed, attempt to load and validate
	cfg, err := LoadFromFile(l.configPath)
	if err != nil {
		log.Printf("Config reload failed: %v (keeping last-known-good config)", err)
		return
	}

	if err := cfg.Validate(); err != nil {
		log.Printf("Config reload failed: validation error: %v (keeping last-known-good config)", err)
		return
	}

	// Atomically swap to new config
	l.activeConfig.Store(cfg)
	l.configSource.Store(SourceFile)
	l.lastChecksum.Store(currentChecksum)
	l.lastReload.Store(time.Now())

	log.Printf("Config reloaded successfully: version %s", cfg.Version)
}

// fileChecksum computes SHA256 checksum of a file
func fileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
