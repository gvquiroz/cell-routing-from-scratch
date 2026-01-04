# System Architecture

## Purpose

This document describes the production-grade architectural patterns implemented in this project. The system demonstrates control plane/data plane separation, atomic configuration distribution, and graceful degradation—patterns common to edge routing infrastructure at Cloudflare, Fastly, and similar platforms.

This is not a feature list. It's a map of invariants, failure domains, and tradeoffs that define how the system behaves under normal and adverse conditions.

## Architectural Invariants

These properties hold across all operating modes:

1. **Zero-downtime updates**: Configuration changes never drop in-flight requests
2. **Last-known-good over newest**: Data planes reject invalid configs, continuing with previous valid state
3. **Fail-safe routing**: Unknown routing keys default to tier3 (catch-all), never returning 5xx for missing routes
4. **Availability over consistency**: Data planes continue routing through control plane outages
5. **Atomic state transitions**: Configuration updates are all-or-nothing; no partial application
6. **Explicit observability**: Every routing decision is logged and exposed via debug endpoints

## System Topology

### Component Separation

```
┌─────────────────────────────────────────────────────────┐
│                    Control Plane (CP)                    │
│  • Watches authoritative config (routing.json)          │
│  • Validates before distribution                        │
│  • Broadcasts snapshots to all data planes              │
│  • Port 8081: /connect (WebSocket), /health            │
└──────────────────┬──────────────────────────────────────┘
                   │ WebSocket (long-lived, CP-initiated)
                   │ • JSON messages: config_snapshot
                   │ • DP responses: ack/nack
                   ▼
┌────────────────────────────────────────────────────────┐
│               Data Plane (DP) - N instances             │
│  • Accepts requests on port 8080                       │
│  • Routes based on X-Routing-Key header                │
│  • Connects outbound to CP (exponential backoff)       │
│  • Falls back to file watch if no CP configured        │
│  • Debug endpoint: /debug/config-version + source      │
└────────────────────────────────────────────────────────┘
```

**Failure domain isolation**: Control plane crashes do not affect data plane request routing. Data planes operate autonomously with last-known-good configuration.

### Configuration Flow

**Control plane mode** (production pattern):
1. CP watches `config/routing.json` (2s ticker)
2. On change: validate, increment version, broadcast to all connected DPs
3. DPs validate received snapshot, send ack/nack
4. DP applies valid config atomically, updates source to "control_plane"
5. DPs reconnect with exponential backoff (1s → 60s) on disconnect

**File-only mode** (single-node fallback):
1. DP watches `config/routing.json` if `CONTROL_PLANE_URL` unset
2. On change: validate, apply atomically, update source to "file"
3. No dependency on control plane

**Bootstrap**: Data planes start with `config/dataplane-initial.json` (v0.0.1) in CP mode, immediately replaced by CP snapshot on connection.

## Routing Model

### Two-Level Indirection

`routingKey → placementKey → endpointURL`

Example:
```
customer-123 → tier2 → http://tier2-cell:8080
customer-456 → tier2 → http://tier2-cell:8080
customer-789 → dedicated-cell-1 → http://dedicated-cell:8080
```

**Tradeoffs**:
- ✅ Decouples tenant identity from infrastructure topology
- ✅ Allows multiple tenants per placement (cost efficiency)
- ✅ Enables placement migration without changing all tenant mappings
- ❌ Extra indirection adds cognitive overhead
- ❌ Config is slightly larger than direct mapping

### Default Routing (tier3)

Unknown routing keys fall back to `tier3` placement.

**Rationale**: Fail-safe over fail-fast. New customers get routed to shared infrastructure rather than receiving 5xx errors. Makes staged rollouts safer (new customers automatically land in default tier).

### Thread Safety

Routing maps are:
1. Built atomically from validated config
2. Swapped via atomic pointer update (`atomic.Value`)
3. Previous maps remain readable during swap (no locks needed)

**Implication**: Read path has zero contention. Routing decisions never block on configuration updates.

## Configuration Validation

### Pre-Distribution Checks

Both CP and DP validate before applying:
- All placements referenced in routes must exist in placements map
- No duplicate routing keys
- Version must be non-empty string
- All URLs must parse correctly

**Failure behavior**: 
- CP: Logs error, does not broadcast, continues watching for next change
- DP: Sends `nack` to CP, retains last-known-good config

### Atomic Application

Config updates are all-or-nothing:
```go
// Build new sta Strategy

### Structured Logging

JSON logs to stdout (standard container pattern):
```json
{
  "timestamp": "2024-01-15T10:23:45Z",
  "level": "info",
  "routing_key": "customer-123",
  "placement": "tier2",
  "upstream": "http://tier2-cell:8080",
  "status": 200,
  "duration_ms": 45,
  "request_id": "req-abc-123"
}
```

**Design choice**: No log levels except error vs info. If you're emitting it, it should be worth storing.

### Explainability Headers

Every proxied response includes:
- `X-Routed-To`: Target upstream URL
- `X-Route-Reason`: Placement key that determined routing
- `X-Request-ID`: Trace ID for distributed request tracking

**Purpose**: Makes every routing decision inspectable without needing log access. Critical for debugging in production.

### Debug Endpoints
Failure Modes and Handling

### Upstream Failures

| Condition | Status | Rationale |
|-----------|--------|-----------|
| Connection refused | 502 | Upstream not listening (deployment issue) |
| Dial timeout (5s) | 502 | Upstream unreachable (network partition) |
| Response timeout (10s) | 504 | Upstream too slow (overload or deadlock) |
| Upstream returns 5xx | Proxied | Upstream owns error, pass through |

**No automatic retries**: Request may have side effects (POST/PUT). Retries belong at client layer where idempotency is known.

### Configuration Failures

| Condition | DP Behavior | CP Behavior |
|-----------|-------------|-------------|
| Invalid config on disk | Log error, keep last-good | Log error, don't broadcast |
| Invalid config from CP | Send nack, keep last-good | N/A |
| MWebSocket Protocol

### Message Types

CP → DP:
```json
{
  "type": "config_snapshot",
  "config": { /* full routing config */ }
}
```

DP Testing Philosophy

28 tests across four packages:

**internal/routing** (8 tests):
- Routing key resolution (two-level mapping)
- Default tier3 fallback
- Thread safety of atomic swap

**internal/config** (13 tests):
- JSON deserialization
- Validation (missing placements, duplicates)
- Atomic reload behavior
- Source tracking (file vs control_plane)

**internal/protocol** (4 tests):
- Message serialization/deserialization
- Type validation

**internal/dataplane** (3 tests):
- Connection establishment
- CExplicit Non-Goals

### Authentication
We assume `X-Routing-Key` is already validated by upstream edge infrastructure.

**Rationale**: Cell routers sit behind API gateways/load balancers in production. Auth is enforced at the perimeter, not repeated at every internal hop. Keeps this component focused on routing decisions.

### TLS Termination
Router ↔ cell communication uses HTTP, not HTTPS.

**Rationale**: Cells are internal services on trusted networks. TLS termination happens at edge (load balancer). mTLS between internal services is a deployment concern, not core to routing logic. Can be added via sidecar proxies (Envoy, Linkerd) without changing router code.

### Health Checking
No active health probes of upstream cells.

**Rationale**: Deferred to M4 (Circuit Breakers milestone). Proper health checking requires:
- Passive: failure rate tracking
- Active: periodic health probes  
- Policy: circuit breaker thresholds
- Coordination: CP must know which cells are unhealthy

Adding half-baked health checks now would create false confidence. Better to route to all configured upstreams and fail fast with 502 than route around presumed-unhealthy upstreams incorrectly.

### Rate Limiting
No per-tenant or per-cell rate limits.

**Rationale**: Rate limiting requires:
- Sliding window counters (memory overhead)
- Distributed coordination (Redis/Memcached for multi-DP)
- Policy configuration (which customers get what limits)
- Backpressure semantics (429 responses, retry-after headers)

This is a full subsystem. Edge infrastructure typically handles this upstream (Cloudflare, Kong, Envoy).

### Metrics
No Prometheus/StatsD instrumentation.

**Rationale**: Structured logs provide request-level observability for this learning project. Production systems need:
- Histograms (latency distribution, not just mean)
- Counters (requests/sec per route, status codes)
- Gauges (active connections, config version)

Can be added via middleware without changing routing logic. Out of scope for foundational architecture demonstration.

### Retries
No automatic retry on upstream failure.

**Rationale**: Retries require idempotency guarantees (safe for GET, dangerous for POST/PUT). Client layer knows request semantics, router doesn't. Automatic retries can amplify cascading failures (retry storm). If needed, implement at client SDK level with backoff and jitter.

## Lessons from Implementation

### Atomic Pointer Swap
`atomic.Value` for config swaps eliminated need for `sync.RWMutex`. Read path has zero lock contention, critical for high-throughput routing. Tradeoff: Slightly awkward API (type assertions), but worth it for performance.

### Full Snapshots over Deltas
WebSocket protocol sends complete config every time, never diffs/patches. This is less "efficient" but dramatically simpler:
- No versioning conflicts
- Reconnection is trivial (just resend latest)
- No accumulation of missed deltas
- Easier to reason about correctness

**Bandwidth cost**: ~10KB per config update for 100 routes. Negligible compared to debugging time saved.

### Exponential Backoff Prevents Thundering Herd
1000 data planes reconnecting every 1s after CP restart would create ~1000 conn/sec spike. Exponential backoff with jitter spreads reconnections over time. Jitter not yet implemented but would be next optimization.

### Fail-Safe Defaults Reduce Operational Burden
Unknown routing keys → tier3 (vs 5xx error) means:
- New customers work immediately
- Typos in config don't break production
- Gradual rollouts are safer (new keys automatically land somewhere)

Tradeoff: Easier to hide misconfigurations. Monitoring must alert on tier3 traffic spikes.
  environment:
    CONTROL_PLANE_URL: ws://control-plane:8081/connect

tier1-cell:     # Upstream
  ports: 8080
tier2-cell:     # Upstream  
  ports: 8080
tier3-cell:     # Upstream
  ports: 8080
```

**Networking**: All containers on shared `app-network`, service discovery via Docker DNS.
1. DP connects to `ws://control-plane:8081/connect`
2. CP immediately sends full config snapshot
3. DP validates, responds with ack/nack
4. CP watches file, broadcasts on change
5. On disconnect: DP retries with exponential backoff

**Design note**: Always full snapshots, never deltas. Simplifies state synchronization, eliminates out-of-order delivery concerns, makes reconnection trivial (no state to rebuild).

### Backoff Strategy

```
Attempt 1: 1s
Attempt 2: 2s
Attempt 3: 4s
Attempt 4: 8s
...
Max: 60s (stays at 60s after reaching)
```

**Reset condition**: Successful connection resets backoff to 1s. immediate reconnect attempt

**Availability guarantee**: Control plane is for orchestration, not request path. Zero DP request failures during CP outage.arded to upstream
