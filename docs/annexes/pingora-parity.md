# Pingora Parity

## Motivation

The baseline router (M1–M4) uses Go's stdlib HTTP stack: `http.Transport` for connection pooling, goroutine-per-request concurrency, and `atomic.Value` for config swaps. This is simple and correct but leaves questions about how the same data plane semantics map to production edge proxies.

Pingora is Cloudflare's Rust-based HTTP proxy framework built on Tokio's async runtime. Comparing the two exposes tradeoffs between implementation simplicity (Go) and edge-grade performance characteristics (Rust/async). This annex outlines how the existing router's behavior translates to Pingora, where constraints differ, and why this is a comparison study rather than a migration guide.

## Semantic Equivalence

The router's core behavior must remain identical:
- Two-level routing (`routingKey → placement → endpoint`)
- Atomic config updates with validation
- Health-aware failover to fallback placements
- Circuit breaker state machine (closed → open → half-open)
- Concurrency limits per placement
- Last-known-good config on control plane failure

Pingora's abstractions (peer selection, load balancing modules, health check subsystem) provide these capabilities but with different primitives.

## Key Mapping Points

**Connection pooling**: Go's `http.Transport` uses a fixed pool per host. Pingora's connection pool is more granular—per upstream, per peer—with configurable idle timeouts and connection limits. The baseline router's simplicity (single URL per placement) maps cleanly, but multi-endpoint placements require Pingora's peer selection logic.

**Concurrency model**: Goroutine-per-request vs Tokio tasks. Go's model is easier to reason about (blocking I/O, synchronous handlers). Pingora requires async/await throughout. Circuit breaker state updates and health check loops remain logically identical but syntactically different.

**Config reload**: Go's `atomic.Value` for lock-free reads translates to Rust's `Arc<RwLock<T>>` or `ArcSwap`. Validation and rejection logic are identical; memory model differs (GC vs ownership).

**Health checks**: The baseline router runs one goroutine per endpoint with periodic HTTP probes. Pingora's health check module is built-in but configurable—active vs passive checks, check intervals, failure thresholds. Behavioral parity is achievable; API surface differs.

**Circuit breakers**: Per-endpoint state machines map directly to Pingora's middleware model. State transitions (record success/failure, allow/reject request) are semantically identical. Pingora's async context requires `async fn` signatures.

## Where Constraints Change

**Streaming**: Go's `io.Copy` for request/response bodies is simple. Pingora requires explicit handling of HTTP body streams (`Stream` trait). Streaming semantics are equivalent but implementation is more verbose.

**TLS**: Baseline router uses `http.Transport` TLS config. Pingora's TLS handling is lower-level (BoringSSL bindings). Client cert validation, ALPN, and session resumption are configurable but require explicit setup.

**Observability**: Go's structured logging is external. Pingora has built-in metrics (Prometheus-compatible) and tracing hooks. Explainability headers (`X-Routed-To`, `X-Circuit-State`) remain identical; instrumentation integration differs.

## Why This Is a Comparison, Not a Rewrite

The educational value of the baseline router is its simplicity: standard library, minimal abstractions, ~1000 lines of routing logic. Pingora improves performance (memory, latency, throughput) but adds complexity (async runtime, Rust ownership, framework APIs).

This annex documents how to **think about** porting the router's semantics to Pingora, not how to **execute** that port. Readers interested in edge proxy tradeoffs can compare the two implementations without building both.

## TODO / Open Questions

- **Benchmark comparison**: Latency (p50, p99) and memory usage under load. Go vs Pingora at 10K req/s.
- **Peer selection**: How Pingora's load balancing interacts with single-endpoint placements vs multi-endpoint placements.
- **Control plane protocol**: WebSocket vs HTTP/2 server push for config distribution. Does Pingora's async model change reconnection strategy?
- **Graceful shutdown**: How Pingora's connection draining compares to Go's `Server.Shutdown` with in-flight request handling.
- **Error handling**: Pingora's `Error` types vs Go's `error` interface. How circuit breaker failures propagate.

**Future expansion**: Side-by-side implementation in `cmd/router-pingora/` for direct comparison. Performance benchmarks. Operational runbook differences.
