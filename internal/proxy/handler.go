package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

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
)

// Handler handles incoming HTTP requests and proxies them to cells
type Handler struct {
	router    *routing.Router
	logger    *logging.Logger
	transport *http.Transport
}

// NewHandler creates a new proxy handler
func NewHandler(router *routing.Router, logger *logging.Logger) *Handler {
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

	return &Handler{
		router:    router,
		logger:    logger,
		transport: transport,
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
		h.logRequest(requestID, r, routingKey, "", "", "", http.StatusBadRequest, time.Since(startTime))
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
		h.logRequest(requestID, r, routingKey, "", "", "", http.StatusInternalServerError, time.Since(startTime))
		return
	}

	// Proxy request to upstream
	statusCode, err := h.proxyRequest(w, r, decision, requestID)
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

	h.logRequest(requestID, r, routingKey, decision.PlacementKey, string(decision.Reason), decision.EndpointURL, statusCode, time.Since(startTime))
}

// proxyRequest proxies the request to the upstream endpoint
func (h *Handler) proxyRequest(w http.ResponseWriter, r *http.Request, decision *routing.RoutingDecision, requestID string) (int, error) {
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

	// Write status code
	w.WriteHeader(upstreamResp.StatusCode)

	// Stream response body
	_, err = io.Copy(w, upstreamResp.Body)

	return upstreamResp.StatusCode, err
}

// logRequest logs the completed request
func (h *Handler) logRequest(requestID string, r *http.Request, routingKey, placementKey, routeReason, upstreamURL string, statusCode int, duration time.Duration) {
	h.logger.LogRequest(logging.RequestLog{
		RequestID:    requestID,
		Method:       r.Method,
		Path:         r.URL.Path,
		RoutingKey:   routingKey,
		PlacementKey: placementKey,
		RouteReason:  routeReason,
		UpstreamURL:  upstreamURL,
		StatusCode:   statusCode,
		DurationMs:   float64(duration.Microseconds()) / 1000.0,
	})
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
