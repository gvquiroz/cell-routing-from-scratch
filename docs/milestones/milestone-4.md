# Milestone 4: Resilience & Traffic Controls

**Status:** 📋 Planned

## Goal

Add resilience patterns to the router while maintaining deterministic, local routing decisions. Handle cell failures gracefully and protect cells from overload.

## Patterns to Implement

### 1. Health Checks & Failover

**Problem**: Cell might be down or unhealthy; router should detect and route elsewhere.

**Solution**:
- Periodic health checks to each cell endpoint
- Mark unhealthy cells in routing table
- Fallback logic: if primary cell unhealthy, route to fallback
- Avoid thundering herd (stagger checks, jitter)

```yaml
placement_to_endpoint:
  tier1:
    primary: http://cell-tier1-a:9001
    fallback: http://cell-tier1-b:9001
    health_check:
      path: /health
      interval: 10s
      timeout: 2s
```

### 2. Rate Limiting

**Problem**: Protect cells from overload; enforce fair use per customer.

**Solution**:
- Per-placement rate limits (e.g., tier1: 1000 req/s)
- Per-routing-key rate limits (e.g., "acme": 100 req/s)
- Return 429 Too Many Requests when exceeded
- Use token bucket or sliding window

```yaml
rate_limits:
  - placement: tier1
    rps: 1000
  - routing_key: acme
    rps: 100
```

### 3. Retries

**Problem**: Transient cell failures should not fail the request.

**Solution**:
- Retry on 5xx or connection errors
- Exponential backoff with jitter
- Max retries: 2
- Only retry idempotent methods (GET, HEAD, PUT, DELETE)
- Log retry attempts

### 4. Circuit Breaker

**Problem**: Cascading failures when cell is consistently failing.

**Solution**:
- Per-cell circuit breaker
- Open circuit after N consecutive failures
- Half-open after timeout to test recovery
- Fail fast while open (return 503)

States:
- **Closed**: Normal operation
- **Open**: Cell is failing, return errors immediately
- **Half-Open**: Testing if cell recovered

## Architecture

```
Request → Router
          ↓
      Rate Limiter (per routing key / placement)
          ↓
      Circuit Breaker (per cell)
          ↓
      Health Check (is cell healthy?)
          ↓
      Proxy (with retries)
          ↓
      Cell
```

## Design Constraints

1. **Local decisions**: All state (health, circuit breaker, rate limits) is local to each router instance
2. **No coordination**: Routers do not share state (acceptable for learning project)
3. **Predictable behavior**: Circuit breaker and health checks should be well-documented and observable
4. **Minimal latency overhead**: Health checks and rate limiting should add <1ms to p99

## Success Criteria

- Router routes around unhealthy cells
- Rate limits enforced accurately (within 10%)
- Circuit breaker opens/closes correctly
- Retries succeed for transient errors
- All patterns are observable via logs and metrics

## Out of Scope

- No distributed rate limiting (per-router only)
- No advanced retry strategies (exponential backoff only)
- No per-endpoint health checks (per-cell only)
- No graceful degradation beyond circuit breaker

## Next: Milestone 5

Reimplement M1-M4 in Cloudflare Pingora for edge proxy comparison.
