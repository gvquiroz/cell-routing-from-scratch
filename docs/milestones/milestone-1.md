# Milestone 1: Static In-Memory Routing

## Architectural Intent

Establish the baseline: routing decisions use only local, in-memory state. Demonstrate that cell-based routing is conceptually straightforward (a two-level map lookup with a default fallback) but requires careful handling of streaming proxies, connection pooling, and decision-making.

## Core Design

Routing is deterministic and requires no coordination:

1. Extract `X-Routing-Key` from request header (customer ID, tenant identifier, etc.)
2. Map routing key → placement key (e.g., "acme" → "tier1")
3. Map placement key → endpoint URL
4. Proxy request with original method, path, query, body (streaming)
5. Return response with routing decision headers for observability

**Placement model**: Routing keys map to failure domains (placements). A placement is either:
- Dedicated cell (single-tenant, "visa")
- Shared tier (multi-tenant, "tier1", "tier2", "tier3")

Unknown routing keys default to "tier3" (catch-all shared tier). Missing routing key header returns 400 Bad Request.

## Invariants Established

This milestone establishes the foundational invariants that hold across all future milestones. See [Architectural Invariants](../../README.md#architectural-invariants) for the full explanation.

- **Local decisions only**: No RPC during routing; tables are immutable after init
- **Streaming proxy**: Bodies streamed via `io.Copy`, not buffered
- **Connection pooling**: Configured timeouts (5s dial, 30s request, 90s idle)

## Failure Modes

| Scenario | Behavior | Rationale |
|----------|----------|-----------|
| Missing `X-Routing-Key` header | 400 Bad Request | Routing key is required context; cannot route without it |
| Unknown routing key | Route to tier3 (default) | Graceful degradation; unknown customers get shared capacity |
| No endpoint for placement | 500 Internal Server Error | Config error; should not occur in valid config |
| Upstream unreachable | 502 Bad Gateway | Network failure; client should retry |
| Upstream slow | Timeout after 30s | Prevents resource exhaustion; client sees 504 |

## Testing

**Integration test** (`test-routing.sh`): Automated script verifying all routing paths, response headers, and status codes against live demo environment.

**Demo environment**: Docker Compose orchestrates 1 router + 4 demo cells (tier1, tier2, tier3, visa). Each cell echoes its identity and request metadata.

Sample log output:
```json
{
  "timestamp": "2026-01-03T19:02:22Z",
  "request_id": "446159eb7309214e4d34b806571f1feb",
  "method": "GET",
  "path": "/api/test",
  "routing_key": "visa",
  "placement_key": "visa",
  "route_reason": "dedicated",
  "upstream_url": "http://cell-visa:9004",
  "status_code": 200,
  "duration_ms": 1.838
}
```

## Deferred to Future Milestones

- Configuration reload (M2): routing tables are hardcoded at startup
- Control plane (M3): no centralized config distribution
- Health checks (M4): no upstream health awareness
- Rate limiting (M4): no per-customer or per-tier limits
- Circuit breakers (M4): no automatic failure isolation
- Retries (M4): no automatic retry logic
- Authentication: routing key is trusted input
- Metrics: structured logs only

This milestone establishes the invariant: routing decisions use only local state. All future milestones preserve this invariant while adding operational capabilities.
