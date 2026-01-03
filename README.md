# Cell Routing from Scratch

A learning-focused implementation of the ingress routing layer in a cell-based architecture. This reverse proxy routes requests to different "cells" based on a trusted header, demonstrating how large-scale systems route traffic to isolated tenant environments.

## Quick Start

```bash
# Start all services
docker compose up --build

# Test routing (in another terminal)
curl -H "X-Routing-Key: visa" http://localhost:8080/
curl -H "X-Routing-Key: acme" http://localhost:8080/

# Run automated tests
./test-routing.sh

# View logs
docker compose logs -f router
```

## How It Works

**Client Request** → **Router** (reads `X-Routing-Key`) → **Target Cell**

The router maintains two mappings:
1. **Customer → Placement**: `acme` → `tier1`, `visa` → `visa` (dedicated)
2. **Placement → Endpoint**: `tier1` → `http://cell-tier1:9001`

Unknown or missing keys default to `tier3`.

### Response Headers

Every response includes:
- `X-Routed-To`: The placement key (tier1/tier2/tier3/visa)
- `X-Route-Reason`: Why? (`dedicated`, `tier`, or `default`)

## Routing Examples

```bash
# Dedicated cell
curl -H "X-Routing-Key: visa" http://localhost:8080/
# → X-Routed-To: visa, X-Route-Reason: dedicated

# Shared tier
curl -H "X-Routing-Key: acme" http://localhost:8080/
# → X-Routed-To: tier1, X-Route-Reason: tier

# Unknown key → defaults to tier3
curl -H "X-Routing-Key: unknown" http://localhost:8080/
# → X-Routed-To: tier3, X-Route-Reason: default

# Missing header
curl http://localhost:8080/
# → 400 Bad Request: X-Routing-Key header is required
```

## Current Mappings

| Customer | Placement | Endpoint |
|----------|-----------|----------|
| acme | tier1 | http://cell-tier1:9001 |
| globex | tier2 | http://cell-tier2:9002 |
| initech | tier3 | http://cell-tier3:9003 |
| visa | visa (dedicated) | http://cell-visa:9004 |
| *unknown* | tier3 (default) | http://cell-tier3:9003 |

## Features

- ✅ Header-based routing with `X-Routing-Key`
- ✅ Streaming reverse proxy (no buffering)
- ✅ Structured JSON logging
- ✅ Connection pooling with timeouts
- ✅ Explainability headers

## Development

```bash
# Run tests
go test ./...

# Run locally without Docker
CELL_NAME=tier1 PORT=9001 go run cmd/cell/main.go &
CELL_NAME=tier2 PORT=9002 go run cmd/cell/main.go &
CELL_NAME=tier3 PORT=9003 go run cmd/cell/main.go &
CELL_NAME=visa PORT=9004 go run cmd/cell/main.go &
go run cmd/router/main.go
```

## Project Structure

```
cmd/
├── router/main.go       # Router entrypoint
└── cell/main.go         # Demo cell server
internal/
├── routing/             # Routing logic + tests
├── proxy/               # HTTP proxy handler
└── logging/             # Structured logging
```

Configuration is in [cmd/router/main.go](cmd/router/main.go). Timeouts are in [internal/proxy/handler.go](internal/proxy/handler.go).

## Milestone 1 Scope

**Included:**
- In-memory deterministic routing
- Streaming proxy with explainability
- JSON logging

**Not Included (future milestones):**
- Hot reload / control plane
- Health checks / circuit breakers  
- Rate limiting / retries
- Authentication
- High availability

## License

MIT
	•	control-plane / data-plane separation,
	•	health-aware routing and failover,
	•	rate limiting, retries, and circuit breaking,

while preserving the core invariant shared by production systems: routing decisions are fast, local, and independent of control plane availability.

The initial implementation uses Go for clarity and approachability. A later milestone re-implements the same behavior using Cloudflare’s Pingora to compare design tradeoffs between application-level proxies and edge-grade proxy runtimes.

This repository is not a framework or a production gateway. It is an educational artifact intended to make the architecture and tradeoffs of cell routing explicit and understandable.
