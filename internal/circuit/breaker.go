package circuit

import (
	"fmt"
	"sync"
	"time"

	"github.com/gvquiroz/cell-routing-from-scratch/internal/logging"
)

// State represents the circuit breaker state
type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

// Config configures circuit breaker behavior
type Config struct {
	FailureThreshold uint32        // Number of consecutive failures before opening
	Timeout          time.Duration // How long to stay open before half-open
}

// Breaker is a per-endpoint circuit breaker
type Breaker struct {
	placementKey    string
	config          Config
	state           State
	failures        uint32
	lastStateChange time.Time
	nextRetryTime   time.Time
	mu              sync.RWMutex
	logger          *logging.Logger
}

// NewBreaker creates a new circuit breaker
func NewBreaker(placementKey string, config Config, logger *logging.Logger) *Breaker {
	return &Breaker{
		placementKey:    placementKey,
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
		logger:          logger,
	}
}

// Allow checks if a request should be allowed through
// Returns true if request can proceed, false if circuit is open
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	switch b.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if it's time to transition to half-open
		if now.After(b.nextRetryTime) {
			b.transitionTo(StateHalfOpen, "timeout_elapsed")
			return true // Allow one request through in half-open
		}
		return false

	case StateHalfOpen:
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful request
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		// Reset failure count on success in closed state
		b.failures = 0

	case StateHalfOpen:
		// Success in half-open means we can close the circuit
		b.failures = 0
		b.transitionTo(StateClosed, "recovery_successful")
	}
}

// RecordFailure records a failed request
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures++

	switch b.state {
	case StateClosed:
		// Check if we've hit the failure threshold
		if b.failures >= b.config.FailureThreshold {
			b.nextRetryTime = time.Now().Add(b.config.Timeout)
			b.transitionTo(StateOpen, fmt.Sprintf("failure_threshold_reached: %d", b.failures))
		}

	case StateHalfOpen:
		// Any failure in half-open means we go back to open
		b.nextRetryTime = time.Now().Add(b.config.Timeout)
		b.transitionTo(StateOpen, "half_open_failure")
	}
}

// GetState returns the current circuit breaker state
func (b *Breaker) GetState() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// GetFailureCount returns the current failure count
func (b *Breaker) GetFailureCount() uint32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.failures
}

// transitionTo changes the circuit breaker state (must be called with lock held)
func (b *Breaker) transitionTo(newState State, reason string) {
	if b.state == newState {
		return
	}

	oldState := b.state
	b.state = newState
	b.lastStateChange = time.Now()

	b.logger.LogInfo(fmt.Sprintf("circuit breaker state transition: %s -> %s", oldState, newState), map[string]interface{}{
		"placement": b.placementKey,
		"old_state": oldState,
		"new_state": newState,
		"failures":  b.failures,
		"reason":    reason,
		"timestamp": time.Now().Unix(),
	})
}

// Manager manages circuit breakers for multiple endpoints
type Manager struct {
	breakers map[string]*Breaker
	config   Config
	logger   *logging.Logger
	mu       sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager(config Config, logger *logging.Logger) *Manager {
	return &Manager{
		breakers: make(map[string]*Breaker),
		config:   config,
		logger:   logger,
	}
}

// GetBreaker returns the circuit breaker for a placement key
// Creates one if it doesn't exist
func (m *Manager) GetBreaker(placementKey string) *Breaker {
	m.mu.RLock()
	breaker, exists := m.breakers[placementKey]
	m.mu.RUnlock()

	if exists {
		return breaker
	}

	// Create new breaker
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[placementKey]; exists {
		return breaker
	}

	breaker = NewBreaker(placementKey, m.config, m.logger)
	m.breakers[placementKey] = breaker
	return breaker
}

// RemoveBreaker removes a circuit breaker for a placement key
func (m *Manager) RemoveBreaker(placementKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, placementKey)
}
