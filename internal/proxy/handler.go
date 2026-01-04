package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gvquiroz/cell-routing-from-scratch/internal/circuit"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/config"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/health"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/limits"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/logging"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/routing"
)

const (
	headerRoutingKey     = "X-Routing-Key"
	headerRequestID      = "X-Request-Id"
	headerForwardedFor   = "X-Forwarded-For"
	headerForwardedProto = "X-Forwarded-Proto"
	headerRoutedTo       = "X-Routed-To"
	headerRouteReason    = "X-Route-Reason"
	headerFailoverReason = "X-Failover-Reason"
	headerCircuitState   = "X-Circuit-State"
)

// Handler handles incoming HTTP requests and proxies them to cells
type Handler struct {
	router         *routing.Router
	config         *config.Config
	logger         *logging.Logger
	transport      *http.Transport
	healthChecker  *health.Checker
	circuitManager *circuit.Manager
	limitsManager  *limits.Manager
}

// NewHandler creates a new proxy handler
func NewHandler(router *routing.Router, cfg *config.Config, logger *logging.Logger) *Handler {
	// Configure transport with reasonable timeouts
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	// Initialize health checker with default config
	healthChecker := health.NewChecker(health.CheckConfig{
		Path:     "/health",
		Interval: 10 * time.Second,
		Timeout:  2 * time.Second,
	}, logger)

	// Initialize circuit breaker manager with default config
	circuitManager := circuit.NewManager(circuit.Config{
		FailureThreshold: 5,
		Timeout:          30 * time.Second,
	}, logger)

	// Initialize limits manager
	limitsManager := limits.NewManager(logger)

	h := &Handler{
		router:         router,
		config:         cfg,
		logger:         logger,
		transport:      transport,
		healthChecker:  healthChecker,
		circuitManager: circuitManager,
		limitsManager:  limitsManager,
	}

	// Register endpoints for health checking and configure limits
	h.configureResilienceMechanisms(cfg)

	return h
}

// configureResilienceMechanisms sets up health checks and limits based on config
func (h *Handler) configureResilienceMechanisms(cfg *config.Config) {
	endpoints := cfg.GetCellEndpoints()

	for placementKey, endpointURL := range endpoints {
		// Get placement-specific config if available
		placementCfg, exists := cfg.GetPlacementConfig(placementKey)

		if exists && placementCfg != nil {
			// Configure health checking
			if placementCfg.HealthCheck != nil {
				parsedHealthCheck, err := placementCfg.HealthCheck.Parse()
				if err == nil {
					checker := health.NewChecker(health.CheckConfig{
						Path:     parsedHealthCheck.Path,
						Interval: parsedHealthCheck.Interval,
						Timeout:  parsedHealthCheck.Timeout,
					}, h.logger)
					checker.RegisterEndpoint(placementKey, endpointURL)
					h.healthChecker = checker // Update with placement-specific checker
				}
			} else {
				// Use default health checker
				h.healthChecker.RegisterEndpoint(placementKey, endpointURL)
			}

			// Configure limits
			if placementCfg.ConcurrencyLimit > 0 || placementCfg.MaxRequestBodyBytes > 0 {
				h.limitsManager.SetConfig(placementKey, limits.Config{
					MaxConcurrentRequests: placementCfg.ConcurrencyLimit,
					MaxRequestBodyBytes:   placementCfg.MaxRequestBodyBytes,
				})
			}
		} else {
			// Use default health checker for legacy configs
			h.healthChecker.RegisterEndpoint(placementKey, endpointURL)
		}
	}
}

// Stop gracefully shuts down the handler
func (h *Handler) Stop() {
	if h.healthChecker != nil {
		h.healthChecker.Stop()
	}
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Generate or extract request ID
	requestID := r.Header.Get(headerRequestID)
	if requestID == "" {
		requestID = generateRequestID()
	}

	// Extract routing key - it's required
	routingKey := r.Header.Get(headerRoutingKey)
	if routingKey == "" {
		h.logger.LogError("missing routing key", nil, map[string]interface{}{
			"request_id": requestID,
		})
		http.Error(w, "Bad Request: X-Routing-Key header is required", http.StatusBadRequest)
		h.logRequest(requestID, r, routingKey, "", "", "", http.StatusBadRequest, time.Since(startTime), "")
		return
	}

	// Make routing decision
	decision, err := h.router.Route(routingKey)
	if err != nil {
		h.logger.LogError("routing error", err, map[string]interface{}{
			"request_id":  requestID,
			"routing_key": routingKey,
		})
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		h.logRequest(requestID, r, routingKey, "", "", "", http.StatusInternalServerError, time.Since(startTime), "")
		return
	}

	placementKey := decision.PlacementKey
	failoverReason := ""

	// Check concurrency limits
	if !h.limitsManager.TryAcquire(placementKey) {
		h.logger.LogError("concurrency limit exceeded", nil, map[string]interface{}{
			"request_id":    requestID,
			"routing_key":   routingKey,
			"placement_key": placementKey,
		})
		http.Error(w, "Service Unavailable: Too Many Requests", http.StatusTooManyRequests)
		h.logRequest(requestID, r, routingKey, placementKey, string(decision.Reason), decision.EndpointURL, http.StatusTooManyRequests, time.Since(startTime), "concurrency_limit")
		return
	}
	defer h.limitsManager.Release(placementKey)

	// Validate request body size
	if r.ContentLength > 0 {
		if err := h.limitsManager.ValidateRequestBodySize(placementKey, r.ContentLength); err != nil {
			h.logger.LogError("request body too large", err, map[string]interface{}{
				"request_id":     requestID,
				"routing_key":    routingKey,
				"placement_key":  placementKey,
				"content_length": r.ContentLength,
			})
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			h.logRequest(requestID, r, routingKey, placementKey, string(decision.Reason), decision.EndpointURL, http.StatusRequestEntityTooLarge, time.Since(startTime), "body_size_limit")
			return
		}
	}

	// Check circuit breaker
	breaker := h.circuitManager.GetBreaker(placementKey)
	if !breaker.Allow() {
		// Circuit is open, check for fallback
		placementCfg, hasFallback := h.config.GetPlacementConfig(placementKey)
		if hasFallback && placementCfg.Fallback != "" {
			// Route to fallback
			fallbackEndpoint := h.config.GetCellEndpoints()[placementCfg.Fallback]
			decision.PlacementKey = placementCfg.Fallback
			decision.EndpointURL = fallbackEndpoint
			failoverReason = "circuit_open"
			placementKey = placementCfg.Fallback

			h.logger.LogInfo("circuit open, routing to fallback", map[string]interface{}{
				"request_id":         requestID,
				"original_placement": decision.PlacementKey,
				"fallback_placement": placementCfg.Fallback,
			})
		} else {
			// No fallback, fail fast
			h.logger.LogError("circuit breaker open, no fallback", nil, map[string]interface{}{
				"request_id":    requestID,
				"routing_key":   routingKey,
				"placement_key": placementKey,
				"circuit_state": breaker.GetState(),
			})
			w.Header().Set(headerCircuitState, string(breaker.GetState()))
			http.Error(w, "Service Unavailable: Circuit Breaker Open", http.StatusServiceUnavailable)
			h.logRequest(requestID, r, routingKey, placementKey, string(decision.Reason), decision.EndpointURL, http.StatusServiceUnavailable, time.Since(startTime), "circuit_open")
			return
		}
	}

	// Check health status
	if !h.healthChecker.IsHealthy(placementKey) {
		// Endpoint unhealthy, check for fallback
		placementCfg, hasFallback := h.config.GetPlacementConfig(placementKey)
		if hasFallback && placementCfg.Fallback != "" {
			// Route to fallback
			fallbackEndpoint := h.config.GetCellEndpoints()[placementCfg.Fallback]
			originalPlacement := decision.PlacementKey
			decision.PlacementKey = placementCfg.Fallback
			decision.EndpointURL = fallbackEndpoint
			failoverReason = "upstream_unhealthy"
			placementKey = placementCfg.Fallback

			h.logger.LogInfo("endpoint unhealthy, routing to fallback", map[string]interface{}{
				"request_id":         requestID,
				"original_placement": originalPlacement,
				"fallback_placement": placementCfg.Fallback,
			})
		} else {
			// No fallback configured, route to default placement (fail-safe)
			defaultPlacement := h.config.GetDefaultPlacement()
			if defaultPlacement != placementKey {
				defaultEndpoint := h.config.GetCellEndpoints()[defaultPlacement]
				decision.PlacementKey = defaultPlacement
				decision.EndpointURL = defaultEndpoint
				failoverReason = "upstream_unhealthy"
				placementKey = defaultPlacement

				h.logger.LogInfo("endpoint unhealthy, routing to default", map[string]interface{}{
					"request_id":         requestID,
					"original_placement": decision.PlacementKey,
					"default_placement":  defaultPlacement,
				})
			}
		}
	}

	// Proxy request to upstream
	statusCode, err := h.proxyRequest(w, r, decision, requestID, failoverReason)

	// Record result in circuit breaker
	if err != nil || statusCode >= 500 {
		breaker.RecordFailure()
	} else {
		breaker.RecordSuccess()
	}

	if err != nil {
		h.logger.LogError("proxy error", err, map[string]interface{}{
			"request_id":    requestID,
			"routing_key":   routingKey,
			"placement_key": decision.PlacementKey,
			"upstream_url":  decision.EndpointURL,
		})

		// Only write error if we haven't started writing response
		if statusCode == 0 {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			statusCode = http.StatusBadGateway
		}
	}

	h.logRequest(requestID, r, routingKey, decision.PlacementKey, string(decision.Reason), decision.EndpointURL, statusCode, time.Since(startTime), failoverReason)
}

// proxyRequest proxies the request to the upstream endpoint
func (h *Handler) proxyRequest(w http.ResponseWriter, r *http.Request, decision *routing.RoutingDecision, requestID, failoverReason string) (int, error) {
	// Parse upstream URL
	upstreamURL, err := url.Parse(decision.EndpointURL)
	if err != nil {
		return 0, err
	}

	// Build the upstream request URL
	targetURL := &url.URL{
		Scheme:   upstreamURL.Scheme,
		Host:     upstreamURL.Host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}

	// Create upstream request
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		return 0, err
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			upstreamReq.Header.Add(key, value)
		}
	}

	// Add/update forwarding headers
	upstreamReq.Header.Set(headerRequestID, requestID)

	// X-Forwarded-For: append client IP
	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if prior := upstreamReq.Header.Get(headerForwardedFor); prior != "" {
		clientIP = prior + ", " + clientIP
	}
	upstreamReq.Header.Set(headerForwardedFor, clientIP)

	// X-Forwarded-Proto
	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}
	upstreamReq.Header.Set(headerForwardedProto, proto)

	// Make upstream request
	client := &http.Client{
		Transport: h.transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		return 0, err
	}
	defer upstreamResp.Body.Close()

	// Copy response headers
	for key, values := range upstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Add explainability headers
	w.Header().Set(headerRoutedTo, decision.PlacementKey)
	w.Header().Set(headerRouteReason, string(decision.Reason))

	// Add failover reason if applicable
	if failoverReason != "" {
		w.Header().Set(headerFailoverReason, failoverReason)
	}

	// Add circuit breaker state
	breaker := h.circuitManager.GetBreaker(decision.PlacementKey)
	w.Header().Set(headerCircuitState, string(breaker.GetState()))

	// Write status code
	w.WriteHeader(upstreamResp.StatusCode)

	// Stream response body
	_, err = io.Copy(w, upstreamResp.Body)

	return upstreamResp.StatusCode, err
}

// logRequest logs the completed request
func (h *Handler) logRequest(requestID string, r *http.Request, routingKey, placementKey, routeReason, upstreamURL string, statusCode int, duration time.Duration, failoverReason string) {
	logData := logging.RequestLog{
		RequestID:    requestID,
		Method:       r.Method,
		Path:         r.URL.Path,
		RoutingKey:   routingKey,
		PlacementKey: placementKey,
		RouteReason:  routeReason,
		UpstreamURL:  upstreamURL,
		StatusCode:   statusCode,
		DurationMs:   float64(duration.Microseconds()) / 1000.0,
	}

	// Add failover reason to extra fields if present
	if failoverReason != "" {
		h.logger.LogInfo(fmt.Sprintf("request completed with failover: %s", failoverReason), map[string]interface{}{
			"request_id":      requestID,
			"method":          r.Method,
			"path":            r.URL.Path,
			"routing_key":     routingKey,
			"placement_key":   placementKey,
			"route_reason":    routeReason,
			"upstream_url":    upstreamURL,
			"status_code":     statusCode,
			"duration_ms":     logData.DurationMs,
			"failover_reason": failoverReason,
		})
	} else {
		h.logger.LogRequest(logData)
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
