# Milestone 5: Pingora Implementation

**Status:** 📋 Planned

## Goal

Reimplement Milestones 1-4 using [Cloudflare Pingora](https://github.com/cloudflare/pingora), a Rust-based HTTP proxy framework. Compare tradeoffs between Go's stdlib approach and an edge-grade proxy runtime.

## Why Pingora?

- **Production-proven**: Powers Cloudflare's edge
- **Performance**: Async Rust with high throughput and low latency
- **Modularity**: Plugin architecture for routing, load balancing, etc.
- **Learning**: Understand how edge proxies differ from stdlib reverse proxies

## Scope

Implement equivalent functionality:

### M1: Static Routing
- Same routing logic (header → placement → endpoint)
- Streaming proxy
- Explainability headers
- Structured logging

### M2: Hot-Reload
- File-based config reload
- Atomic swap without downtime

### M3: Control Plane Push
- WebSocket integration
- DP autonomy

### M4: Resilience
- Health checks
- Rate limiting (Pingora's built-in rate limiter)
- Retries
- Circuit breaker

## Key Differences to Explore

| Aspect | Go Implementation | Pingora Implementation |
|--------|-------------------|------------------------|
| **Concurrency** | Goroutines + channels | Tokio async runtime |
| **Connection pooling** | http.Transport | Pingora's connection pool |
| **Memory model** | GC | Rust ownership |
| **Config reload** | RWMutex/atomic.Value | Arc/RwLock |
| **Rate limiting** | Custom token bucket | Pingora's built-in |
| **Performance** | Good (1-2ms overhead) | Excellent (<0.5ms) |

## Deliverables

1. **Pingora router**: Equivalent to Go router (cmd/router)
2. **Benchmarks**: Side-by-side comparison
3. **Documentation**: Lessons learned, tradeoffs, recommendations
4. **README section**: When to choose Go vs Pingora

## Success Criteria

- Feature parity with Go implementation (M1-M4)
- Clear performance comparison
- Educational writeup on architectural differences

## Out of Scope

- Not replacing Go implementation (both exist for comparison)
- Not adding Pingora-specific features beyond M1-M4 parity
- Not a production deployment guide

## Resources

- [Pingora GitHub](https://github.com/cloudflare/pingora)
- [Pingora Documentation](https://github.com/cloudflare/pingora/tree/main/docs)
- [Building a Pingora proxy](https://blog.cloudflare.com/pingora-open-source)
