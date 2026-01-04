# Cell Routing from Scratch

An educational implementation of cell-based ingress routing that makes control plane and data plane separation explicit. This repository demonstrates how production edge systems maintain local routing decisions while allowing centralized configuration distribution, and why that separation matters for large-scale reliability.

## Problem Statement

Cell-based architectures isolate workloads into independent failure domains—dedicated tenant environments or shared capacity tiers. The ingress layer routes requests to cells based on request context (customer ID, tenant identifier, etc.) without introducing synchronous dependencies during request processing.

This constraint—local routing decisions with asynchronous config updates—appears in every production edge system: Envoy's xDS protocol, service mesh control planes, CDN edge nodes. The pattern is well-understood at scale but rarely demonstrated in isolation.

This repository builds that pattern from first principles across four milestones:

1. Static routing with in-memory tables
2. Atomic config swaps with last-known-good fallback  
3. WebSocket-based control plane distribution
4. Local resilience (health checks, circuit breakers, overload protection)

Each milestone preserves the fundamental invariants while adding operational capability.

## Architectural Invariants

These constraints hold across all milestones:

**Control plane never in request path**  
Routing decisions use only local, in-memory state. No RPC to a control plane during request processing. Config updates are asynchronous—data plane applies them when received, not when requested.

**Atomic configuration updates**  
New routing tables are validated before application. Swaps are atomic via `atomic.Value`; no partial state visible to concurrent requests. Invalid configs are rejected, previous config remains active.

**Graceful degradation**  
Data plane survives control plane downtime indefinitely. Routers operate with last-known-good config. Control plane unavailability does not impact request serving.

**Observable routing decisions**  
Every response includes routing metadata (`X-Routed-To`, `X-Route-Reason`) sufficient to debug placement decisions from logs alone.

These invariants mirror patterns from production systems: Envoy operates from local snapshots pushed via xDS; service mesh data planes maintain autonomy from their control planes; CDN edge nodes make routing decisions from locally-cached config.

## Milestone Progression

| Milestone | Control Plane | Data Plane Behavior | Architectural Tradeoff |
|-----------|---------------|---------------------|------------------------|
| **M1** ✅ | None (static) | Hardcoded in-memory routing | Zero operational complexity; zero config flexibility |
| **M2** ✅ | File + hot-reload | Atomic swap on file change | Local config management; manual propagation to fleet |
| **M3** ✅ | WebSocket broadcast | Receives CP updates; survives CP failure | Centralized config source; DP maintains local autonomy |
| **M4** ✅ | Health + circuit breakers | Local resilience without coordination | Per-router state; fail-safe fallback routing |

Each milestone isolates a specific layer of operational capability while preserving the core invariants. See [docs/milestones](docs/milestones/) for detailed architecture notes.

## Running Locally

```bash
# Start control plane + router + 4 demo cells
docker compose up --build

# Verify routing (each response includes X-Routed-To, X-Route-Reason headers)
curl -H "X-Routing-Key: visa" http://localhost:8080/    # dedicated cell
curl -H "X-Routing-Key: acme" http://localhost:8080/    # shared tier1
curl -H "X-Routing-Key: unknown" http://localhost:8080/ # default tier3

# Check config source and version
curl http://localhost:8080/debug/config

# Test config propagation: edit config/routing.json, wait ~5 seconds
# Control plane detects change, broadcasts to routers
# Check /debug/config to confirm version update

# Run test suite
go test ./...
```

Router logs and cell logs available via `docker compose logs -f router` and `docker compose logs -f cell-visa`.

## What This Demonstrates

**Control plane / data plane separation**  
Why routing decisions must be local (latency, blast radius, availability). How asynchronous config distribution maintains data plane autonomy. The tradeoff between centralized coordination and local resilience.

**Atomic state transitions**  
Techniques for validating and applying configuration without exposing partial state to concurrent requests. Why last-known-good config matters more than newest config.

**Failure domain isolation**  
How routers behave when control plane is unavailable, slow, or serving invalid config. Why graceful degradation requires local decision-making capability.

**Operational observability**  
What metadata enables debugging distributed routing decisions from logs alone. How to make routing decisions auditable without distributed tracing.

This repository does not implement a production system. It makes the architectural patterns explicit and testable in isolation.

## Implementation Status

**Current**: Milestone 4 complete. Stateless resilience mechanisms added to data plane: active health checks per endpoint, per-endpoint circuit breakers, and overload protection (concurrency limits, body size limits). All state is local and in-memory. Routers route to fallback placements when primary endpoints are unhealthy or circuits are open.

**Routing model**: Two-level indirection (`routingKey → placementKey → endpoint`) with default fallback. Placement keys represent failure domains (dedicated cells, shared tiers). Unknown routing keys default to shared tier3; missing routing key header returns 400.

**Configuration separation**:
- Control plane: watches `config/routing.json` (authoritative source)
- Data plane: bootstraps from `config/dataplane-initial.json`, immediately replaced by CP push
- File-only mode: data plane watches `routing.json` directly (no CP)

**Observability**: Structured JSON logs include request_id, routing decision, timing. Debug endpoint `/debug/config` shows current version, config source, last reload timestamp. Routing decision exposed in response headers for offline analysis.

See [milestone specifications](docs/milestones/) for architectural details and failure mode analysis.

## Repository Structure

```
cmd/
├── router/        # Data plane: routing + proxy + CP client (M3)
├── control-plane/ # Control plane: WebSocket broadcast (M3)
└── cell/          # Demo backend cells
internal/
├── config/        # Parsing, validation, atomic swap (M2/M3/M4)
├── routing/       # Routing decision logic + tests (M1)
├── proxy/         # HTTP reverse proxy with resilience (M1/M4)
├── protocol/      # WebSocket message types (M3)
├── controlplane/  # CP server implementation (M3)
├── dataplane/     # DP client with reconnection (M3)
├── health/        # Active health checking (M4)
├── circuit/       # Circuit breaker implementation (M4)
├── limits/        # Concurrency and size limits (M4)
├── debug/         # Debug endpoints (M2)
└── logging/       # Structured JSON logging (M1)
config/
├── routing.json           # Control plane config source
└── dataplane-initial.json # Data plane bootstrap config
docs/milestones/           # Architecture specifications per milestone
```

Each milestone's implementation is documented in `docs/milestones/milestone-N.md` with architectural intent, invariants preserved, and failure modes considered.

## Explicit Non-Goals

This is not a production framework, API gateway, or service mesh. It is an educational artifact that makes architectural patterns testable.

- No authentication or authorization (assumes trusted ingress)
- No metrics beyond structured logs
- No Kubernetes integration or declarative operators
- No distributed coordination or consensus protocols
- No request tracing or distributed debugging tools

Milestones add operational complexity (config reload, control plane distribution, local resilience) while maintaining focus on core architectural patterns over feature completeness.