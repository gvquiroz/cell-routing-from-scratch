# Rate Limiting

## Motivation

The baseline router enforces **concurrency limits** (max N requests per placement) but not **rate limits** (max N requests per second per routing key). Concurrency limits protect the router from saturation; rate limits protect upstreams from individual tenant overload.

This annex explores why rate limiting introduces complexity, what design options exist, and where enforcement boundaries matter.

## Why Rate Limiting Is Complex

**State distribution**: Per-key rate limits require counting requests across time windows (token bucket, sliding window, leaky bucket). Local counting (per-router state) allows per-key variance if requests aren't evenly distributed. Distributed counting (shared state) requires coordination (Redis, rate limiter service).

**Fairness vs accuracy**: Strict per-key fairness requires synchronous state checks on every request. Approximate fairness (local buckets, periodic sync) trades accuracy for latency and availability.

**Failure modes**: If rate limiter state is unavailable (Redis down, rate limiter service unreachable), router must decide: fail open (allow requests, risk overload) or fail closed (reject requests, impact availability).

## Design Options

### Local (Per-Router) Rate Limiting

Each router maintains its own token buckets per routing key. No coordination between routers.

**Pros**:
- Zero latency (in-memory check)
- No external dependencies
- Survives rate limiter service failures

**Cons**:
- Unfair if traffic is unevenly distributed (one router sees 90% of a tenant's traffic)
- Effective limit is `N routers × per-router limit`
- Can't enforce strict global limits

### Distributed (Coordinated) Rate Limiting

All routers share rate limit state via external service (Redis, dedicated rate limiter like Envoy's global rate limit service).

**Pros**:
- Accurate global limits regardless of traffic distribution
- Fair across entire fleet
- Limit enforcement survives individual router failures

**Cons**:
- Adds latency (network round-trip per request or periodic batch updates)
- Introduces external dependency (Redis availability, rate limiter service uptime)
- Fail-open or fail-closed decision on state unavailability

**When acceptable**: Large fleets, strict fairness required, acceptable latency overhead (~1–5ms), external state service is highly available.

## Enforcement Boundaries

**Where to enforce**: Options include:
- **Ingress layer**: Before routing decisions. Protects routers but doesn't account for cell-specific capacity.
- **Router layer**: After routing decision, before proxy. Accounts for placement but router becomes stateful.
- **Cell layer**: At upstream. Simplest isolation but doesn't prevent wasted router → cell traffic.

**Failure semantics**: On rate limit exceeded:
- Return `429 Too Many Requests` immediately (fail-fast)
- Queue request with timeout (bounded queueing)
- Route to fallback placement with lower QoS (degraded service)

## Why Excluded from Core Router

The baseline router's invariant is **stateless local decisions**. Adding distributed rate limiting breaks this—routers now depend on external state for correctness. Even local rate limiting complicates config (per-key limits in routing table?) and observability (how to debug rate limit rejections across fleet?).

Milestone 4 adds **concurrency limits** because they preserve local state: semaphore-based slot acquisition, per-router enforcement, no coordination. Rate limiting requires either distributed state or sticky routing, both of which change architectural assumptions.

## TODO / Open Questions

- **Hybrid approach**: Local rate limiting with periodic sync to Redis. What sync interval balances fairness and latency?
- **Per-placement vs per-key limits**: Should rate limits apply to `routingKey` (tenant) or `placementKey` (cell)? Or both?
- **Graceful degradation**: If distributed rate limiter is unavailable, fall back to local limits? How to avoid thundering herd when state service recovers?
- **Config schema**: How to express per-key limits in routing config? Inline in `routingTable` or separate `rateLimits` object?
