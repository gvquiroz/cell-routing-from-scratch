# Cell Routing from Scratch

An educational implementation of the ingress routing layer in a cell-based architecture. This repository explores the design space between control plane sophistication and data plane autonomy, with a focus on making architectural tradeoffs explicit.

## The Problem

Modern distributed systems isolate workloads into cells (dedicated tenant environments or shared tiers) to contain blast radius and enable independent scaling. The ingress layer must route incoming requests to the correct cell based on request metadata while maintaining strict operational invariants:

1. **Routing decisions are local**: No synchronous dependency on a control plane during request processing
2. **Config updates are asynchronous**: Control plane pushes updates; data plane applies them atomically
3. **Failure isolation**: Data plane continues serving with last-known-good config if control plane is unavailable

These constraints appear in production systems (Envoy xDS, Linkerd, Istio, internal edge systems) but are rarely demonstrated in isolation. This repository implements them incrementally to make the tradeoffs explicit.

## Core Invariants

- **Control plane never in hot path**: All routing decisions use in-memory state; no RPC to CP during request proxying
- **Atomic config updates**: New routing tables are validated and swapped atomically; no partial application
- **Graceful degradation**: Router survives CP downtime/restarts without dropping traffic
- **Explainability**: Every response includes routing metadata (`X-Routed-To`, `X-Route-Reason`) for debugging

## Architecture Progression

Each milestone introduces a new layer of control plane sophistication while preserving data plane autonomy:

| Milestone | Control Plane | Data Plane Behavior | Key Tradeoff |
|-----------|---------------|---------------------|--------------|
| **M1** | Static (none) | Hardcoded in-memory routing | Zero operational complexity; zero flexibility |
| **M2** | File + hot-reload | Atomic swap on file change | Local config; manual distribution |
| **M3** | WebSocket push | Receives CP updates; survives CP outage | CP coordinates fleet; DP stays autonomous |
| **M4** | Health state + limits | Local health checks, rate limits, circuit breakers | Per-router state; no global coordination |
| **M5** | (Pingora reimpl) | Compare stdlib vs async runtime tradeoffs | Go simplicity vs Rust performance |

[Full milestone specifications](docs/milestones/README.md)

## Quick Start

```bash
# Start router + 4 demo cells
docker compose up --build

# Observe routing behavior
curl -H "X-Routing-Key: visa" http://localhost:8080/    # → dedicated cell
curl -H "X-Routing-Key: acme" http://localhost:8080/    # → shared tier1
curl -H "X-Routing-Key: unknown" http://localhost:8080/ # → default tier3

# Each response includes routing decision headers
# X-Routed-To: visa
# X-Route-Reason: dedicated | tier | default

# Automated test suite
./test-routing.sh
```

**Structured logs** (router and cells):
```bash
docker compose logs -f router
docker compose logs -f cell-visa
```

## What This Teaches

**Control plane/data plane separation**  
How to decouple routing logic from config distribution; why CP must never block request processing.

**Atomic configuration updates**  
Techniques for validating and swapping routing state without partial application or race conditions.

**Failure modes and degradation**  
How routers behave when control plane is slow, unavailable, or serving invalid config.

**Resilience patterns (M4)**  
When to use health checks vs circuit breakers; local vs distributed rate limiting tradeoffs.

**Proxy runtime tradeoffs (M5)**  
Comparing Go's stdlib HTTP proxy (goroutines, GC) against Pingora's async runtime (Tokio, Rust ownership).

## Implementation Notes

**Current (M1)**: Static routing with streaming reverse proxy in Go. No external dependencies beyond stdlib. Routing maps are immutable after initialization; no locking required for concurrent requests.

**Routing model**: Two-level lookup (`routingKey → placementKey → endpoint`) with default fallback. Unknown routing keys map to `tier3`; missing header returns 400.

**Observability**: Structured JSON logs include request ID, routing decision, timing. Each cell logs incoming requests with forwarded metadata.

[Milestone 1 specification](docs/milestones/milestone-1.md)

## Project Structure

```
cmd/router/       # Data plane: routing + proxy
cmd/cell/         # Demo backend cells
internal/
├── routing/      # Routing decision logic + tests
├── proxy/        # HTTP reverse proxy with streaming
└── logging/      # Structured JSON logging
docs/milestones/  # Architecture specifications per milestone
```

## Development

```bash
go test ./...                          # Run unit tests
go run cmd/router/main.go              # Run router locally
CELL_NAME=tier1 PORT=9001 go run cmd/cell/main.go &  # Run cell
```

## Non-Goals

This is not a production framework, API gateway, or deployable service mesh. It is an educational artifact designed to make architectural patterns and tradeoffs explicit.

- No authentication/authorization (assumes trusted upstream)
- No metrics/tracing integration (logs only)
- No Kubernetes operators or declarative config
- No distributed coordination (per-router state only)

Future milestones add operational complexity (hot reload, CP integration, health checks) but maintain focus on architecture over features.

## License

MIT

## Quick Start

```bash
# Start router + 4 demo cells
docker compose up --build

# Test routing
curl -H "X-Routing-Key: visa" http://localhost:8080/
curl -H "X-Routing-Key: acme" http://localhost:8080/

# Run automated tests
./test-routing.sh
```

## How It Works

**Client** → **Router** (reads `X-Routing-Key`) → **Target Cell**

```bash
# Dedicated cell
curl -H "X-Routing-Key: visa" http://localhost:8080/
# → X-Routed-To: visa, X-Route-Reason: dedicated

# Shared tier (tier1)
curl -H "X-Routing-Key: acme" http://localhost:8080/
# → X-Routed-To: tier1, X-Route-Reason: tier

# Unknown → defaults to tier3
curl -H "X-Routing-Key: unknown" http://localhost:8080/
# → X-Routed-To: tier3, X-Route-Reason: default
```

Response headers show routing decision: `X-Routed-To`, `X-Route-Reason`

## Milestones

| # | Status | Description | Docs |
|---|--------|-------------|------|
| **M1** | ✅ Complete | Static in-memory routing with streaming proxy | [Details](docs/milestones/milestone-1.md) |
| **M2** | 📋 Planned | Hot-reload config from file | [Details](docs/milestones/milestone-2.md) |
| **M3** | 📋 Planned | Control plane push via WebSocket | [Details](docs/milestones/milestone-3.md) |
| **M4** | 📋 Planned | Health checks, rate limiting, circuit breakers | [Details](docs/milestones/milestone-4.md) |
| **M5** | 📋 Planned | Reimplement in Pingora for comparison | [Details](docs/milestones/milestone-5.md) |

[📚 Milestone Overview](docs/milestones/README.md)

## Project Structure

```
cmd/router/       # Router entrypoint
cmd/cell/         # Demo cell server
internal/
├── routing/      # Routing logic + tests
├── proxy/        # HTTP proxy handler
└── logging/      # Structured logging
docs/milestones/  # Detailed specs
```

## What You'll Learn

- **CP/DP Separation**: Control plane never in hot path; routing is local and fast
- **Graceful Degradation**: Router stays up even if control plane is down
- **Resilience Patterns**: Health checks, circuit breakers, rate limiting (M4)
- **Proxy Internals**: Streaming, connection pooling, timeouts
- **Go vs Pingora**: Stdlib proxy vs edge-grade proxy runtime (M5)

## Development

```bash
# Run tests
go test ./...

# Run locally without Docker
CELL_NAME=tier1 PORT=9001 go run cmd/cell/main.go &
go run cmd/router/main.go
```

See [Milestone 1 docs](docs/milestones/milestone-1.md) for complete implementation details.

## License

MIT
	•	control-plane / data-plane separation,
	•	health-aware routing and failover,
	•	rate limiting, retries, and circuit breaking,

while preserving the core invariant shared by production systems: routing decisions are fast, local, and independent of control plane availability.

The initial implementation uses Go for clarity and approachability. A later milestone re-implements the same behavior using Cloudflare’s Pingora to compare design tradeoffs between application-level proxies and edge-grade proxy runtimes.

This repository is not a framework or a production gateway. It is an educational artifact intended to make the architecture and tradeoffs of cell routing explicit and understandable.
