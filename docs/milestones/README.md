# Milestones

Detailed architectural specifications for each milestone. See the [main README](../../README.md) for the learning path and progression overview.

## Documents

- [Milestone 1: Static Routing](milestone-1.md) - Baseline in-memory routing and streaming proxy
- [Milestone 2: Atomic Config Reload](milestone-2.md) - Hot-reload with validation and last-known-good
- [Milestone 3: Control Plane Distribution](milestone-3.md) - WebSocket-based config push and data plane autonomy
- [Milestone 4: Local Resilience](milestone-4.md) - Health checks, circuit breakers, and overload protection

Each document explains:
- **Architectural intent**: Why this capability matters
- **Invariants preserved**: What doesn't change
- **Implementation approach**: How it works
- **Failure modes**: What can go wrong
- **Out-of-scope items**: What's deferred

These are structured as design notes focused on architectural reasoning, not step-by-step implementation guides.
