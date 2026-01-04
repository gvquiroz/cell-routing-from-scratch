# Design Annexes

## Annexes

- **[Request Processing Pipelines](request-processing-pipelines.md)**: Foundational model for composing ingress concerns (WAF, auth, rate limiting, routing). Sequential stages, short-circuit semantics, trust boundaries. Go middleware and Pingora phases as examples.

- **[Caching and CDN](caching-and-cdn.md)**: Cache as pipeline stage. Placement strategies (before/after routing, two-tier). Tenant isolation through cache keys. Invalidation and failure semantics during cell failover.

- **[Distributed Rate Limiting](rate-limiting.md)**: Per-key rate limiting design space. Local vs distributed enforcement. Why it's excluded from core router scope.

- **[Authentication and Routing Keys](auth-routing-key.md)**: Trust boundaries for routing metadata.

- **[WAF and Edge Security](waf-and-edge-security.md)**: WAF placement relative to routing and cells. Global edge vs per-cell enforcement.

- **[Shuffle Sharding](shuffle-sharding.md)**: Blast radius isolation through shard assignment. Routing implications. Interaction with health-aware failover.

- **[Pingora Parity](pingora-parity.md)**: Mapping data plane semantics to Cloudflare Pingora. Runtime tradeoffs (Go goroutines vs Tokio async). Not a rewrite guide.

- **[Workers Edge Router](workers-edge-router.md)**: Cell routing at the edge using Cloudflare Workers. Edge-specific constraints (streaming, config distribution, failure modes). How assumptions change.


## Purpose

These documents explore architectural topics beyond the core milestone progression. While milestones demonstrate incremental operational capability (M1: static routing → M4: resilience), annexes examine design choices, tradeoffs, and alternative implementations that extend or reframe the baseline architecture.

Annexes are intentionally **comparative and exploratory**. They address questions like:
- How does this architecture map to different runtimes (Pingora, Workers)?
- What changes when we add capabilities intentionally excluded from milestones (rate limiting, authentication, shuffle sharding)?
- Where are the trust boundaries and failure domains?
- What tradeoffs exist between local, sticky, and distributed enforcement?

Annexes assume familiarity with the core milestones. Readers should understand the baseline architecture (M1–M4) before exploring these extensions.
