# Design Annexes

## Purpose

These documents explore architectural topics beyond the core milestone progression. While milestones demonstrate incremental operational capability (M1: static routing → M4: resilience), annexes examine design choices, tradeoffs, and alternative implementations that extend or reframe the baseline architecture.

Annexes are intentionally **comparative and exploratory**. They address questions like:
- How does this architecture map to different runtimes (Pingora, Workers)?
- What changes when we add capabilities intentionally excluded from milestones (rate limiting, authentication, shuffle sharding)?
- Where are the trust boundaries and failure domains?
- What tradeoffs exist between local, sticky, and distributed enforcement?

## Why Separate from Milestones

Milestones establish a cohesive learning path with specific invariants: local routing decisions, stateless data planes, atomic config updates, control plane autonomy. Adding topics like distributed rate limiting or WAF placement would dilute that focus.

Annexes preserve the milestone narrative while providing Staff+/Principal-level design lenses for readers interested in:
- Production considerations beyond educational scope
- Runtime comparisons (Go stdlib vs Pingora vs Workers)
- Security and trust boundaries
- Blast radius and sharding strategies

## Scope and Completeness

These documents are **intentionally incomplete**. They outline design spaces, surface tradeoffs, and pose open questions. They are not implementation guides. Code examples appear only when essential to illustrate a constraint or pattern.

Annexes assume familiarity with the core milestones. Readers should understand the baseline architecture (M1–M4) before exploring these extensions.

## Annexes

- **[Pingora Parity](pingora-parity.md)**: Mapping data plane semantics to Cloudflare Pingora. Runtime tradeoffs (Go goroutines vs Tokio async). Not a rewrite guide.

- **[Workers Edge Router](workers-edge-router.md)**: Cell routing at the edge using Cloudflare Workers. Edge-specific constraints (streaming, config distribution, failure modes). How assumptions change.

- **[Rate Limiting](rate-limiting.md)**: Per-key rate limiting design space. Local vs sticky vs distributed enforcement. Why it's excluded from core router scope.

- **[Authentication and Routing Keys](auth-routing-key.md)**: Trust boundaries for routing metadata. Header injection vs JWT claims vs mTLS. Separation of auth from routing logic.

- **[WAF and Edge Security](waf-and-edge-security.md)**: WAF placement relative to routing and cells. Global edge vs per-cell enforcement. Observability and failure implications.

- **[Shuffle Sharding](shuffle-sharding.md)**: Blast radius isolation through shard assignment. Routing implications. Interaction with health-aware failover.

## Tone and Audience

These documents are written for senior engineers familiar with distributed systems, edge architectures, and operational tradeoffs. They prioritize architectural reasoning over tutorial-style explanation. Tone is calm, technical, and non-marketing.

Open questions and TODO sections are explicit—these annexes document design explorations, not complete solutions.
