package limits

import (
	"context"
	"fmt"
	"sync"

	"github.com/gvquiroz/cell-routing-from-scratch/internal/logging"
)

// Config defines resource limits for a placement
type Config struct {
	MaxConcurrentRequests int   // Max concurrent requests per placement
	MaxRequestBodyBytes   int64 // Max request body size in bytes
}

// Semaphore implements a counting semaphore for concurrency control
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a new semaphore with the given capacity
func NewSemaphore(capacity int) *Semaphore {
	return &Semaphore{
		ch: make(chan struct{}, capacity),
	}
}

// Acquire acquires one slot, blocking if at capacity
// Returns an error if context is canceled
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire one slot without blocking
// Returns true if acquired, false if at capacity
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases one slot
func (s *Semaphore) Release() {
	<-s.ch
}

// Manager manages concurrency limits for multiple placements
type Manager struct {
	semaphores map[string]*Semaphore
	config     map[string]Config
	logger     *logging.Logger
	mu         sync.RWMutex
}

// NewManager creates a new limits manager
func NewManager(logger *logging.Logger) *Manager {
	return &Manager{
		semaphores: make(map[string]*Semaphore),
		config:     make(map[string]Config),
		logger:     logger,
	}
}

// SetConfig sets the limit configuration for a placement
func (m *Manager) SetConfig(placementKey string, config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config[placementKey] = config

	// Create or update semaphore
	if config.MaxConcurrentRequests > 0 {
		m.semaphores[placementKey] = NewSemaphore(config.MaxConcurrentRequests)
	} else {
		// Remove semaphore if no limit set
		delete(m.semaphores, placementKey)
	}
}

// GetConfig returns the limit configuration for a placement
func (m *Manager) GetConfig(placementKey string) (Config, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, exists := m.config[placementKey]
	return config, exists
}

// TryAcquire attempts to acquire a concurrency slot for a placement
// Returns true if acquired, false if at limit
func (m *Manager) TryAcquire(placementKey string) bool {
	m.mu.RLock()
	sem, exists := m.semaphores[placementKey]
	m.mu.RUnlock()

	if !exists {
		// No limit configured, allow request
		return true
	}

	acquired := sem.TryAcquire()
	if !acquired {
		m.logger.LogInfo("concurrency limit reached", map[string]interface{}{
			"placement": placementKey,
			"action":    "rejected",
		})
	}
	return acquired
}

// Release releases a concurrency slot for a placement
func (m *Manager) Release(placementKey string) {
	m.mu.RLock()
	sem, exists := m.semaphores[placementKey]
	m.mu.RUnlock()

	if exists {
		sem.Release()
	}
}

// ValidateRequestBodySize checks if request body size is within limits
func (m *Manager) ValidateRequestBodySize(placementKey string, size int64) error {
	config, exists := m.GetConfig(placementKey)
	if !exists || config.MaxRequestBodyBytes == 0 {
		// No limit configured
		return nil
	}

	if size > config.MaxRequestBodyBytes {
		return fmt.Errorf("request body size %d exceeds limit %d", size, config.MaxRequestBodyBytes)
	}
	return nil
}

// RemoveConfig removes limit configuration for a placement
func (m *Manager) RemoveConfig(placementKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.config, placementKey)
	delete(m.semaphores, placementKey)
}
