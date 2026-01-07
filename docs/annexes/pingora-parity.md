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

**Connection pooling**: Go's `http.Transport` uses a fixed pool per host. Pingora's connection pool is more granular (per upstream, per peer) with configurable idle timeouts and connection limits. The baseline router's simplicity (single URL per placement) maps cleanly, but multi-endpoint placements require Pingora's peer selection logic.

**Concurrency model**: Goroutine-per-request vs Tokio tasks. Go's model is easier to reason about (blocking I/O, synchronous handlers). Pingora requires async/await throughout. Circuit breaker state updates and health check loops remain logically identical but syntactically different.

**Config reload**: Go's `atomic.Value` for lock-free reads translates to Rust's `Arc<RwLock<T>>` or `ArcSwap`. Validation and rejection logic are identical; memory model differs (GC vs ownership).

**Health checks**: The baseline router runs one goroutine per endpoint with periodic HTTP probes. Pingora's health check module is built-in but configurable: active vs passive checks, check intervals, failure thresholds. Behavioral parity is achievable; API surface differs.

**Circuit breakers**: Per-endpoint state machines map directly to Pingora's middleware model. State transitions (record success/failure, allow/reject request) are semantically identical. Pingora's async context requires `async fn` signatures.

## Where Constraints Change

**Streaming**: Go's `io.Copy` for request/response bodies is simple. Pingora requires explicit handling of HTTP body streams (`Stream` trait). Streaming semantics are equivalent but implementation is more verbose.

**TLS**: Baseline router uses `http.Transport` TLS config. Pingora's TLS handling is lower-level (BoringSSL bindings). Client cert validation, ALPN, and session resumption are configurable but require explicit setup.

## Future Exploration

A full Pingora implementation would address these mapping questions:

- **Benchmarks**: Latency (p50, p99) and memory usage under load. Initial data suggests Pingora achieves lower tail latency at high concurrency due to async I/O, but Go's simplicity reduces development time.
- **Peer selection**: Pingora's load balancing modules handle multi-endpoint placements natively. Single-endpoint placements (current design) bypass this complexity.
- **Config protocol**: HTTP/2 server push could replace WebSocket for config distribution. Async runtime doesn't fundamentally change reconnection strategy; exponential backoff applies either way.
- **Graceful shutdown**: Pingora's connection draining is explicit (`drain_connections()`). Go's `Server.Shutdown` handles in-flight requests automatically. Behavior is equivalent; API differs.
- **Error propagation**: Pingora's typed errors (`Error` enum) vs Go's `error` interface. Circuit breaker state transitions map cleanly to either model.
