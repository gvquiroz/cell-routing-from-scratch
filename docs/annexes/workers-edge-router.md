# Workers Edge Router

## Motivation

The baseline router runs as a long-lived process (Docker container, VM, Kubernetes pod). Cell routing at the edge—using Cloudflare Workers or similar edge compute—changes foundational assumptions: no persistent connections, no local filesystem, millisecond CPU budgets, and different failure modes.

This annex explores how the same two-level routing model (`routingKey → placement → endpoint`) applies in an edge environment, what constraints differ, and where the architecture must adapt.

## Edge-Specific Constraints

**No persistent state**: Workers are stateless request handlers. The baseline router's in-memory health check state and circuit breaker state don't persist between requests. Health awareness requires external state (KV, Durable Objects, or coordination service).

**No streaming by default**: Baseline router uses `io.Copy` for streaming request/response bodies. Workers support streaming via `ReadableStream` / `TransformStream`, but default behavior buffers. Large payloads (multi-GB uploads) require explicit stream handling.

**Config distribution**: Baseline router receives config via WebSocket or file watch. Workers fetch config from KV (eventual consistency, global replication) or Durable Objects (strong consistency, single-region coordination). No persistent TCP connection for config push.

**Cold start latency**: First request to a Worker incurs initialization cost (script parse, global scope execution). The baseline router's startup time (load config, connect to CP) is one-time; Workers' cold start is per-isolate, per-region.

**CPU time limits**: Workers have per-request CPU budgets (10–50ms depending on plan). Circuit breaker state updates, health check logic, and routing decisions must complete within this budget. Baseline router has no such constraint.

## Routing Model Adaptations

**Two-level lookup remains**: `X-Routing-Key → placement → endpoint` is still a map lookup. Logic is identical, runtime constraints differ.

**Health checks become external**: Workers can't run background goroutines for periodic health probes. Options:
- **Scheduled Workers**: Cron-triggered health checks write state to KV/DO.
- **Passive health checks**: Track upstream errors per request, mark unhealthy after threshold.
- **Control plane health state**: CP performs checks, pushes health status to KV alongside routing config.

**Circuit breakers need coordination**: Per-endpoint circuit state can't live in Worker memory. Options:
- **Durable Objects**: One DO per endpoint, holds circuit state, handles open/close transitions.
- **KV with TTL**: Store circuit state in KV, key = `circuit:placement`, value = `open|closed`, TTL = circuit timeout.
- **Regional stickiness**: Route same `routingKey` to same Worker isolate (Durable Objects guarantee) to preserve in-memory state. Breaks on Worker migration.

**Fallback routing**: Identical to baseline—if primary placement is unhealthy or circuit is open, route to fallback. Health state lookup is now KV/DO read instead of in-memory check.

## Config Distribution

Baseline router atomically swaps config on WebSocket push. Workers fetch config from KV on each request or cache it in global scope.

**KV-based config**:
- CP writes `routing-config` KV key (versioned JSON).
- Worker reads KV on first request, caches in global scope.
- Revalidate periodically (e.g., every 60s) or on version mismatch.
- Eventual consistency: different Workers may see different config versions for ~seconds.

**Durable Objects config**:
- CP pushes config to a single DO (e.g., `config-coordinator`).
- Workers RPC to DO for routing table.
- Strong consistency, single point of coordination.
- Adds latency (DO call per request unless cached).

## Failure Modes

**KV unavailability**: If Workers can't read KV, routing breaks unless fallback config is hardcoded in Worker script. Baseline router keeps last-known-good config in memory.

**DO unavailability**: If Durable Object is unreachable (region failure, evacuation), Workers can't fetch state. Fallback to default placement or fail requests with 503.

**Upstream unreachable**: Same as baseline—return 502 or route to fallback. No difference in failure semantics, only in how "unhealthy" is detected (external health checks vs local probes).

## TODO / Open Questions

- **Streaming benchmarks**: How Workers' streaming primitives compare to `io.Copy` for large payloads (multi-GB uploads/downloads).
- **Circuit breaker coordination cost**: Latency overhead of DO-based circuit state vs local in-memory state. Acceptable for edge routing?
- **Config caching strategy**: How often to revalidate KV config without adding latency. Cache in global scope vs per-request fetch.
- **Cold start impact**: How much does cold start latency matter for cell routing at the edge? Are requests retried upstream if Worker times out?
- **Multi-region failover**: How routing decisions differ when Workers are globally distributed vs baseline router in single region. Does health state need to be region-aware?

**Future expansion**: Reference implementation in `cmd/router-workers/` using Workers SDK. Performance comparison: Workers vs baseline router. Operational runbook for edge deployment.
