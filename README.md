# Cell Routing from Scratch

Modern large-scale systems increasingly rely on cell-based architectures to limit blast radius, improve isolation, and scale independently across tenants and regions. While cells are widely discussed, the mechanics of routing requests into the correct cell are often hidden and custom built.

This repository is a learning-focused, incremental implementation of the ingress routing layer in a cell-based architecture.

Starting from first principles, it builds a minimal reverse proxy that:
- Receives traffic on a global endpoint
- Identifies the tenant from trusted request metadata
- Resolves the target cell in-memory
- Proxies the request without relying on a centralized control plane at request time

## Milestone 1: Minimal Deterministic Router

This milestone implements a production-ready reverse proxy that routes requests to cells based on a trusted routing key.

### Features

- **Header-based routing**: Routes based on `X-Routing-Key` header
- **In-memory mapping**: Deterministic routing with no control plane dependencies
- **Default fallback**: Unknown or missing keys default to `tier3`
- **Streaming proxy**: Supports streaming request/response bodies without buffering
- **Explainability headers**: Responses include `X-Routed-To` and `X-Route-Reason`
- **Structured logging**: JSON logs with request_id, timing, and routing decisions
- **Connection pooling**: Configurable timeouts and keep-alive for upstream connections

### Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ X-Routing-Key: acme
       ▼
┌─────────────────────────────┐
│  Router (Port 8080)         │
│  ┌───────────────────────┐  │
│  │ customerToPlacement   │  │
│  │ "acme" → "tier1"      │  │
│  │ "globex" → "tier2"    │  │
│  │ "initech" → "tier3"   │  │
│  │ "visa" → "visa"       │  │
│  └───────────────────────┘  │
│  ┌───────────────────────┐  │
│  │ placementToEndpoint   │  │
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

### Routing Logic

1. Extract `X-Routing-Key` from request header
2. Lookup routing key → placement key (e.g., "acme" → "tier1")
3. If missing or unknown → default to "tier3"
4. Lookup placement key → endpoint URL
5. Proxy request to the endpoint
6. Add response headers:
   - `X-Routed-To`: The placement key used
   - `X-Route-Reason`: One of:
     - `dedicated` - Routed to a dedicated cell (non-tier)
     - `tier` - Routed to a shared tier based on known key
     - `default` - Routed to tier3 due to missing/unknown key

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Or Go 1.22+ for local development

### Run with Docker Compose

```bash
docker compose up --build
```

This starts:
- Router on `http://localhost:8080`
- Four cells (tier1, tier2, tier3, visa) on internal network

### Example Requests

**1. Route to dedicated cell (visa):**
```bash
curl -H "X-Routing-Key: visa" http://localhost:8080/api/payments
```

Expected response headers:
```
X-Routed-To: visa
X-Route-Reason: dedicated
X-Request-Id: <generated-id>
```

**2. Route to tier1 (acme customer):**
```bash
curl -H "X-Routing-Key: acme" http://localhost:8080/health
```

Expected response headers:
```
X-Routed-To: tier1
X-Route-Reason: tier
```

**3. Route to tier2 (globex customer):**
```bash
curl -H "X-Routing-Key: globex" http://localhost:8080/
```

Expected response headers:
```
X-Routed-To: tier2
X-Route-Reason: tier
```

**4. Default routing (unknown key):**
```bash
curl -H "X-Routing-Key: unknown" http://localhost:8080/
```

Expected response headers:
```
X-Routed-To: tier3
X-Route-Reason: default
```

**5. Default routing (missing key):**
```bash
curl http://localhost:8080/
```

Expected response headers:
```
X-Routed-To: tier3
X-Route-Reason: default
```

**6. Verbose output with headers:**
```bash
curl -v -H "X-Routing-Key: visa" http://localhost:8080/test
```

### Automated Testing

Run the provided test script to verify all routing scenarios:

```bash
./test-routing.sh
```

This will test all routing paths and verify the response headers are correct.

### View Logs

The router outputs structured JSON logs:

```bash
docker compose logs -f router
```

Example log entry:
```json
{
  "timestamp": "2026-01-03T10:15:30Z",
  "request_id": "a1b2c3d4e5f6...",
  "method": "GET",
  "path": "/api/payments",
  "routing_key": "visa",
  "placement_key": "visa",
  "route_reason": "dedicated",
  "upstream_url": "http://cell-visa:9004",
  "status_code": 200,
  "duration_ms": 12.5
}
```

## Development

### Run Tests

```bash
go test ./...
```

### Run Locally (without Docker)

Start cells manually:
```bash
CELL_NAME=tier1 PORT=9001 go run cmd/cell/main.go &
CELL_NAME=tier2 PORT=9002 go run cmd/cell/main.go &
CELL_NAME=tier3 PORT=9003 go run cmd/cell/main.go &
CELL_NAME=visa PORT=9004 go run cmd/cell/main.go &
```

Update endpoints in [cmd/router/main.go](cmd/router/main.go) to use `localhost`:
```go
placementToEndpoint := map[string]string{
    "tier1": "http://localhost:9001",
    "tier2": "http://localhost:9002",
    "tier3": "http://localhost:9003",
    "visa":  "http://localhost:9004",
}
```

Start router:
```bash
go run cmd/router/main.go
```

## Project Structure

```
.
├── cmd/
│   ├── router/          # Router entrypoint
│   │   └── main.go
│   └── cell/            # Demo cell server
│       └── main.go
├── internal/
│   ├── routing/         # Routing decision logic
│   │   ├── router.go
│   │   └── router_test.go
│   ├── proxy/           # HTTP proxy handler
│   │   └── handler.go
│   └── logging/         # Structured logging
│       └── logger.go
├── docker-compose.yml   # Multi-service orchestration
├── Dockerfile.router    # Router container
├── Dockerfile.cell      # Cell container
├── go.mod
└── README.md
```

## Configuration

### Routing Mappings (Milestone 1)

Defined in [cmd/router/main.go](cmd/router/main.go):

**Customer to Placement:**
- `acme` → `tier1`
- `globex` → `tier2`
- `initech` → `tier3`
- `visa` → `visa` (dedicated)

**Placement to Endpoint:**
- `tier1` → `http://cell-tier1:9001`
- `tier2` → `http://cell-tier2:9002`
- `tier3` → `http://cell-tier3:9003`
- `visa` → `http://cell-visa:9004`

**Default:** `tier3`

### Timeouts

Configured in [internal/proxy/handler.go](internal/proxy/handler.go):
- Dial timeout: 5s
- TLS handshake timeout: 5s
- Response header timeout: 10s
- Request timeout: 30s
- Idle connection timeout: 90s

## Limitations (Milestone 1)

- Static in-memory configuration (no hot reload)
- No control plane integration
- No health checks or circuit breakers
- No rate limiting or retries
- No authentication (assumes trusted upstream)
- Single router instance (no HA)

Future milestones will address these limitations.

## License

MIT

The project evolves in clearly defined milestones, progressively adding:
	•	hot-reloaded configuration,
	•	control-plane / data-plane separation,
	•	health-aware routing and failover,
	•	rate limiting, retries, and circuit breaking,

while preserving the core invariant shared by production systems: routing decisions are fast, local, and independent of control plane availability.

The initial implementation uses Go for clarity and approachability. A later milestone re-implements the same behavior using Cloudflare’s Pingora to compare design tradeoffs between application-level proxies and edge-grade proxy runtimes.

This repository is not a framework or a production gateway. It is an educational artifact intended to make the architecture and tradeoffs of cell routing explicit and understandable.
