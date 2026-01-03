# Architecture Decisions - Milestone 1

## Overview
This document captures key architectural decisions made during Milestone 1 implementation.

## Routing Design

### Decision: Two-Level Mapping
We use a two-level indirection: `routingKey → placementKey → endpointURL`

**Rationale:**
- Allows multiple routing keys (customers) to share the same placement
- Decouples customer identity from infrastructure topology
- Makes it easier to migrate customers between tiers or create dedicated cells

### Decision: Immutable In-Memory Maps
Routing maps are initialized once at startup and never modified.

**Rationale:**
- Thread-safe without locks (read-only access)
- Simple and predictable behavior
- Sufficient for Milestone 1 (no hot reload requirement)
- Forces restarts for config changes, which is acceptable for now

### Decision: Default to tier3
Missing or unknown routing keys default to `tier3` placement.

**Rationale:**
- Fail safe: every request gets routed somewhere
- tier3 acts as a "catch-all" tier
- Prevents 5xx errors for unknown customers
- Makes debugging easier (unknown traffic goes to one place)

## Proxy Implementation

### Decision: Standard Library Reverse Proxy Approach
We built a custom proxy handler rather than using `httputil.ReverseProxy`.

**Rationale:**
- More control over headers (explainability headers)
- Clearer logging integration
- Streaming support without buffering
- Direct access to status codes for logging
- Educational value (shows how proxies work)

**Trade-offs:**
- More code to maintain
- Need to handle edge cases ourselves
- Could switch to `httputil.ReverseProxy` in future if complexity grows

### Decision: Connection Pooling with Timeouts
We configure the HTTP transport with specific timeouts and connection reuse.

**Timeouts configured:**
- Dial timeout: 5s (connection establishment)
- TLS handshake: 5s
- Response headers: 10s (TTFB)
- Request timeout: 30s (total request)
- Idle connection: 90s (keep-alive)

**Rationale:**
- Prevents hung connections
- Reuses connections for better performance
- Protects against slow upstream servers
- Standard production-ready values

## Observability

### Decision: Structured JSON Logging
All logs are emitted as JSON to stdout.

**Rationale:**
- Easy to parse by log aggregation systems
- Consistent format across all log entries
- No third-party dependencies
- Standard practice for containerized applications

### Decision: Explainability Headers
Every response includes `X-Routed-To` and `X-Route-Reason` headers.

**Rationale:**
- Makes routing decisions transparent
- Aids debugging and testing
- No performance impact
- Useful for monitoring and alerting
- Educational (helps understand the system)

### Decision: Request ID Propagation
Generate request IDs if not present; propagate if already exists.

**Rationale:**
- Enables distributed tracing
- Links logs across services
- Standard practice in microservices
- Low overhead

## Error Handling

### Decision: Return 502 for Upstream Failures
Unreachable or failing upstreams return 502 Bad Gateway.

**Rationale:**
- Correct HTTP semantics (5xx = server error)
- 502 specifically means "bad upstream"
- Client can distinguish from router errors (500)
- Standard reverse proxy behavior

### Decision: Return 500 for Configuration Errors
Missing placement endpoints return 500 Internal Server Error.

**Rationale:**
- Indicates a configuration problem (server fault)
- Should never happen in production (config is validated)
- Helps catch deployment issues

## Testing Strategy

### Decision: Unit Tests for Routing Logic Only
We only test the pure routing decision function, not the proxy.

**Rationale:**
- Routing logic is complex and critical
- Proxy is mostly glue code using standard library
- Integration tests would require mocking HTTP
- Keeps tests fast and focused
- Can add integration tests in future milestones

## Deployment

### Decision: Multi-Container Docker Compose
Development environment uses Docker Compose with separate containers.

**Rationale:**
- Easy to run locally
- Mirrors production architecture (separate cells)
- Tests container builds
- No need to manage multiple terminals
- Includes networking (service discovery)

### Decision: Multi-Stage Docker Builds
Both Dockerfiles use multi-stage builds (builder + runtime).

**Rationale:**
- Smaller final images (alpine-based)
- Separates build dependencies from runtime
- Faster subsequent builds (layer caching)
- Standard Go Docker pattern

## Future Considerations

### What We Explicitly Deferred
- **Health checks:** No active health monitoring of cells
- **Control plane:** No dynamic routing table updates
- **Rate limiting:** No per-customer or per-cell limits
- **Circuit breakers:** No automatic failover logic
- **Retries:** No automatic retry on failure
- **Metrics:** No Prometheus/StatsD instrumentation
- **Config reload:** Requires restart to change routing

### Why These Are Deferred
These features add significant complexity and are not needed for the learning goals of Milestone 1. Each will be addressed in future milestones as we build up the system incrementally.

## Non-Decisions (What We Didn't Do)

### Authentication/Authorization
We explicitly assume `X-Routing-Key` is already validated by an upstream service.

**Rationale:**
- Auth is a separate concern
- Router should be a simple, fast proxy
- Keeps Milestone 1 focused
- Real systems have auth at the edge (API gateway, load balancer)

### TLS Termination
Router communicates with cells over HTTP (not HTTPS).

**Rationale:**
- Cells are on internal network
- TLS termination typically happens at edge
- Simplifies demo environment
- Can add mTLS in future if needed
