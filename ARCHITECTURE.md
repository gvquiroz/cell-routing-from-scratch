# Architecture

## Invariants

1. **Zero-downtime updates**: Config changes never drop in-flight requests
2. **Last-known-good over newest**: Data planes reject invalid configs
3. **Fail-safe routing**: Unknown keys → tier3 (never 5xx)
4. **Availability over consistency**: Data planes work through CP outages
5. **Atomic state transitions**: All-or-nothing config updates
6. **Local routing**: All decisions in-memory; CP never on request path

## Topology

```
Control Plane (CP)
  • Watches routing.json
  • Validates → broadcasts via WebSocket
  • Port 8081: /connect, /health

Data Plane (DP)
  • Port 8080: routes requests
  • Reads X-Routing-Key header
  • Connects to CP or falls back to file watch
  • Debug: /debug/config-version
```

**Failure isolation**: CP crashes don't affect DP routing.

## Routing Model

**Two-level indirection**: `routingKey → placementKey → endpointURL`

```
customer-123 → tier2 → http://tier2-cell:8080
customer-789 → dedicated-cell-1 → http://dedicated-cell:8080
```

**Why**: Decouples tenant identity from infrastructure. Multiple tenants per placement. Placement migration without remapping all tenants.

**Default**: Unknown keys → tier3 (fail-safe over fail-fast).

**Thread safety**: `atomic.Value` pointer swap. Zero lock contention on read path.

## Config Updates

**Validation**: Both CP and DP check:
- All placements exist
- No duplicate keys
- Valid URLs and version

**On failure**:
- CP: logs error, doesn't broadcast
- DP: sends nack, keeps last-good

**Application**: Atomic pointer swap. All-or-nothing.
## Failure Handling

| Condition | Response | Reason |
|-----------|----------|--------|
| Connection refused | 502 | Upstream not listening |
| Dial timeout (5s) | 502 | Network partition |
| Response timeout (10s) | 504 | Upstream overload |
| Upstream 5xx | Proxied | Upstream owns error |
| Invalid config | Keep last-good | Never break traffic |

**No automatic retries**: Requests may have side effects. Client decides.

## Design Decisions

**Full snapshots over deltas**: CP sends complete config every time. Simpler reconnection, no versioning conflicts. Cost: ~10KB per update.

**Atomic pointer swap**: `atomic.Value` eliminates locks on read path. Critical for throughput.

**Exponential backoff**: 1s → 60s prevents thundering herd on CP restart.

**Fail-safe defaults**: Unknown keys → tier3 instead of 5xx. New customers work immediately. Typos don't break prod.

## Explicit Non-Goals

**Auth**: Assumes `X-Routing-Key` validated upstream.

**Health checks**: Deferred to M4. Route to all, fail fast with 502.

**Retries**: No automatic retry on upstream failure. Retries require idempotency guarantees. Client layer knows request semantics, router doesn't. Automatic retries amplify cascading failures (retry storm).
