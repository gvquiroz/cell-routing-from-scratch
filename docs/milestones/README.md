# Milestones

Each milestone isolates a specific architectural capability while preserving the core invariants of local routing decisions and data plane autonomy.

## Learning Path

This progression mirrors how production edge systems evolve from static configuration to centralized orchestration:

| Milestone | Status | Capability Added | Operational Tradeoff |
|-----------|--------|------------------|---------------------|
| [M1: Static Routing](milestone-1.md) | ✅ Complete | In-memory routing, streaming proxy | Zero operations, zero flexibility |
| [M2: Atomic Config Reload](milestone-2.md) | ✅ Complete | File-based hot-reload with validation | Local config management, manual fleet updates |
| [M3: Control Plane Distribution](milestone-3.md) | ✅ Complete | WebSocket push from centralized CP | Coordinated fleet config, data plane stays autonomous |
| [M4: Local Resilience](milestone-4.md) | ✅ Complete | Health checks, circuit breakers, overload protection | Per-router resilience state, no cross-router coordination |

## Architectural Progression

**M1** establishes the baseline: routing decisions use only local, in-memory state. No external dependencies during request processing.

**M2** introduces configuration flexibility via atomic swaps. Routing tables become mutable, but updates must be validated before application. Last-known-good config prevents invalid updates from breaking routing.

**M3** separates config source from config application. Control plane becomes authoritative source; data plane receives updates asynchronously. Data plane survives control plane failure indefinitely—routing continues with last-known-good config.

**M4** adds per-router resilience without distributed coordination. Health checks detect unhealthy upstreams; circuit breakers prevent cascading failures; concurrency limits protect against overload. All state remains local to each router. Fallback routing ensures requests succeed even when primary placements fail.

## Design Principles

Each milestone:
- Preserves all invariants from previous milestones
- Adds one new operational capability
- Documents the architectural tradeoff introduced
- Includes failure mode analysis
- Remains independently runnable and testable

## Reading Approach

Start with [Milestone 1](milestone-1.md) to understand the baseline architecture. Each subsequent milestone document explains:
- Architectural intent (why this capability matters)
- Invariants preserved (what doesn't change)
- Implementation approach (how it works)
- Failure modes (what can go wrong)
- Out-of-scope items (what's deferred)

These documents are structured as design notes, not implementation guides. They focus on architectural reasoning and tradeoffs, not step-by-step instructions.
