# Milestone 1: Static In-Memory Routing

**Status:** ✅ Complete

## Goal

Implement a minimal but production-ready reverse proxy that routes requests to cells based on a trusted header. Routing is deterministic, in-memory, and requires no control plane.

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ X-Routing-Key: acme
       ▼
┌─────────────────────────────┐
│  Router (Port 8080)         │
│  ┌───────────────────────┐  │
│  │ routingTable          │  │
│  │ "acme" → "tier1"      │  │
│  │ "globex" → "tier2"    │  │
│  │ "initech" → "tier3"   │  │
│  │ "visa" → "visa"       │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ cellEndpoints         │  │
│  │ "tier1" → cell-tier1  │  │
│  │ "tier2" → cell-tier2  │  │
│  │ "tier3" → cell-tier3  │  │
│  │ "visa" → cell-visa    │  │
│  └───────────────────────┘  │
└──────────┬──────────────────┘
           │
           ▼
    ┌──────────────┐
    │ Cell: tier1  │
    │ Port: 9001   │
    └──────────────┘
```

## Routing Logic

1. **Extract** `X-Routing-Key` from request header (required)
2. **Lookup** `routingKey → placementKey` (e.g., "acme" → "tier1")
3. **Default** if routing key unknown → "tier3"
4. **Lookup** `placementKey → endpointURL`
5. **Proxy** request to endpoint with:
   - Original method, path, query, body (streaming)
   - Added: `X-Forwarded-For`, `X-Forwarded-Proto`, `X-Request-Id`
6. **Return** response with explainability headers:
   - `X-Routed-To`: placement key used
   - `X-Route-Reason`: `dedicated` | `tier` | `default`

### Route Reason Logic

- `dedicated` - Placement is not tier1/tier2/tier3 (e.g., "visa")
- `tier` - Placement is tier1/tier2/tier3 and routing key was found
- `default` - Routing key was unknown, defaulted to tier3

### Error Handling

- Missing `X-Routing-Key` header → **400 Bad Request**
- No endpoint for placement → **500 Internal Server Error**
- Upstream unreachable → **502 Bad Gateway**

## Implementation

### Package Structure

```
internal/
├── routing/
│   ├── router.go          # Core routing decision logic
│   └── router_test.go     # Unit tests (8 tests, all passing)
├── proxy/
│   └── handler.go         # HTTP proxy with streaming
└── logging/
    └── logger.go          # Structured JSON logging
```

### Key Design Decisions

1. **Immutable Maps**: Routing tables are read-only after initialization → no locking needed for concurrent requests
2. **Streaming Proxy**: Uses `io.Copy` to avoid buffering entire request/response bodies
3. **Connection Pooling**: Configured `http.Transport` with reasonable timeouts:
   - Dial: 5s
   - TLS handshake: 5s
   - Response header: 10s
   - Request: 30s
   - Idle conn: 90s
4. **Structured Logging**: JSON to stdout with request_id, timing, routing decisions
5. **Standard Library**: No external dependencies beyond Go stdlib

## Demo Environment

Docker Compose orchestrates:
- 1x router (exposed on :8080)
- 4x demo cells (tier1, tier2, tier3, visa on internal network)

Each cell echoes its identity and key request metadata.

## Testing

### Unit Tests
```bash
go test ./internal/routing
```

8 test cases covering:
- Dedicated routing (visa)
- Tier routing (tier1, tier2, tier3)
- Default fallback (unknown key)
- Missing endpoint error

### Integration Tests
```bash
./test-routing.sh
```

Automated script that verifies:
- All routing paths
- Response headers
- Status codes

### Manual Testing
```bash
# Valid routing key
curl -H "X-Routing-Key: visa" http://localhost:8080/

# Unknown key → defaults to tier3
curl -H "X-Routing-Key: unknown" http://localhost:8080/

# Missing header → 400 error
curl http://localhost:8080/
```

## Sample Logs

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

## Out of Scope (Future Milestones)

- ❌ Config reload (M2)
- ❌ Control plane integration (M3)
- ❌ Health checks (M4)
- ❌ Rate limiting (M4)
- ❌ Circuit breakers (M4)
- ❌ Retries (M4)
- ❌ Authentication
- ❌ Metrics/monitoring
- ❌ High availability

## Next: Milestone 2

Hot-reload configuration from file with atomic swap and last-known-good fallback.
