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
| **M1** ✅ | Static (none) | Hardcoded in-memory routing | Zero operational complexity; zero flexibility |
| **M2** ✅ | File + hot-reload | Atomic swap on file change | Local config; manual distribution |
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

# Check current config (M2)
curl http://localhost:8080/debug/config

# Hot-reload test (M2)
# Edit config/routing.json, wait 5-10 seconds, check /debug/config again

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

**Current (M1-M2)**: File-based configuration with hot-reload (5s polling). Atomic config swaps using `atomic.Value`; validation ensures consistency before applying updates. Falls back to last-known-good config on validation failure. Streaming reverse proxy in Go with no external dependencies.

**Routing model**: Two-level lookup (`routingKey → placementKey → endpoint`) with default fallback. Unknown routing keys map to `tier3`; missing header returns 400.

**Config hot-reload (M2)**: Polls `config/routing.json` every 5 seconds, validates changes, atomically swaps routing tables. Invalid configs are rejected with clear error logs while router continues serving with previous valid config.

**Observability**: Structured JSON logs include request ID, routing decision, timing. Each cell logs incoming requests with forwarded metadata. Debug endpoint at `/debug/config` shows current version and last reload timestamp.

[Milestone 1 specification](docs/milestones/milestone-1.md)  
[Milestone 2 specification](docs/milestones/milestone-2.md)

## Project Structure

```
cmd/router/       # Data plane: routing + proxy
cmd/cell/         # Demo backend cells
internal/
├── config/       # Config parsing, validation, hot-reload (M2)
├── routing/      # Routing decision logic + tests
├── proxy/        # HTTP reverse proxy with streaming
├── debug/        # Debug endpoints (M2)
└── logging/      # Structured JSON logging
config/           # Sample routing configuration (M2)
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