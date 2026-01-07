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

## Config Distribution

Baseline router atomically swaps config on WebSocket push. Workers fetch config from KV on each request or cache it in global scope.

**KV-based config**:
- CP writes `routing-config` KV key (versioned JSON).
- Worker reads KV on first request, caches in global scope.
- Revalidate periodically (e.g., every 60s) or on version mismatch.
- Eventual consistency: different Workers may see different config versions for ~seconds.

## Failure Modes

**KV unavailability**: If Workers can't read KV, routing breaks unless fallback config is hardcoded in Worker script. Baseline router keeps last-known-good config in memory.

**Upstream unreachable**: Same as baseline—return 502 or route to fallback.

## Future Exploration

Edge routing at Workers scale introduces constraints not present in traditional proxies:

- **Config caching**: Cache config in global scope (persists across requests in same isolate), revalidate on version mismatch or TTL expiry. Typical: 60s TTL with async refresh.
- **Cold start impact**: Cold starts add 5–50ms. For cell routing, this is acceptable—routing decisions are fast once initialized. Pre-warming via scheduled triggers mitigates impact for latency-sensitive paths.
- **Multi-region awareness**: Workers execute globally, but cells may be region-specific. Routing decisions should incorporate region hints (Cloudflare's `cf.colo`) to prefer nearby cells.
