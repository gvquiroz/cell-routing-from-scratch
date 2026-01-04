package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gvquiroz/cell-routing-from-scratch/internal/logging"
)

// State represents the health state of an endpoint
type State string

const (
	StateHealthy   State = "healthy"
	StateUnhealthy State = "unhealthy"
)

// CheckConfig configures health checking for an endpoint
type CheckConfig struct {
	Path     string
	Interval time.Duration
	Timeout  time.Duration
}

// EndpointHealth tracks the health of a single endpoint
type EndpointHealth struct {
	URL       string
	State     State
	LastCheck time.Time
	mu        sync.RWMutex
}

// GetState returns the current health state thread-safely
func (e *EndpointHealth) GetState() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.State
}

// setState updates the health state thread-safely
func (e *EndpointHealth) setState(state State) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.State = state
	e.LastCheck = time.Now()
}

// Checker manages health checks for multiple endpoints
type Checker struct {
	endpoints map[string]*EndpointHealth
	config    CheckConfig
	logger    *logging.Logger
	client    *http.Client
	mu        sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewChecker creates a new health checker
func NewChecker(config CheckConfig, logger *logging.Logger) *Checker {
	return &Checker{
		endpoints: make(map[string]*EndpointHealth),
		config:    config,
		logger:    logger,
		client: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		stopCh: make(chan struct{}),
	}
}

// RegisterEndpoint adds an endpoint to be health checked
func (c *Checker) RegisterEndpoint(placementKey, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.endpoints[placementKey]; exists {
		return
	}

	endpoint := &EndpointHealth{
		URL:   url,
		State: StateHealthy, // Start as healthy
	}
	c.endpoints[placementKey] = endpoint

	// Start health checking goroutine
	c.wg.Add(1)
	go c.checkLoop(placementKey, endpoint)
}

// UnregisterEndpoint removes an endpoint from health checking
func (c *Checker) UnregisterEndpoint(placementKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.endpoints, placementKey)
}

// IsHealthy returns whether an endpoint is healthy
func (c *Checker) IsHealthy(placementKey string) bool {
	c.mu.RLock()
	endpoint, exists := c.endpoints[placementKey]
	c.mu.RUnlock()

	if !exists {
		// Unknown endpoints are assumed healthy (fail-open)
		return true
	}

	return endpoint.GetState() == StateHealthy
}

// GetState returns the health state of an endpoint
func (c *Checker) GetState(placementKey string) State {
	c.mu.RLock()
	endpoint, exists := c.endpoints[placementKey]
	c.mu.RUnlock()

	if !exists {
		return StateHealthy
	}

	return endpoint.GetState()
}

// Stop stops all health checking goroutines
func (c *Checker) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// checkLoop runs periodic health checks for an endpoint
func (c *Checker) checkLoop(placementKey string, endpoint *EndpointHealth) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()

	// Perform initial check immediately
	c.performCheck(placementKey, endpoint)

	for {
		select {
		case <-ticker.C:
			c.performCheck(placementKey, endpoint)
		case <-c.stopCh:
			return
		}
	}
}

// performCheck executes a single health check
func (c *Checker) performCheck(placementKey string, endpoint *EndpointHealth) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	healthURL := endpoint.URL + c.config.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		c.transitionState(placementKey, endpoint, StateUnhealthy, fmt.Sprintf("request_creation_failed: %v", err))
		return
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.transitionState(placementKey, endpoint, StateUnhealthy, fmt.Sprintf("request_failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Consider 2xx as healthy, anything else as unhealthy
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.transitionState(placementKey, endpoint, StateHealthy, "")
	} else {
		c.transitionState(placementKey, endpoint, StateUnhealthy, fmt.Sprintf("status_code_%d", resp.StatusCode))
	}
}

// transitionState updates endpoint state and logs transitions
func (c *Checker) transitionState(placementKey string, endpoint *EndpointHealth, newState State, reason string) {
	oldState := endpoint.GetState()

	if oldState != newState {
		endpoint.setState(newState)
		c.logger.LogInfo(fmt.Sprintf("health state transition: %s -> %s", oldState, newState), map[string]interface{}{
			"placement": placementKey,
			"url":       endpoint.URL,
			"old_state": oldState,
			"new_state": newState,
			"reason":    reason,
			"timestamp": time.Now().Unix(),
		})
	} else {
		// State unchanged, just update last check time
		endpoint.setState(newState)
	}
}
