# Milestone 3: Control Plane Push

**Status:** 📋 Planned

## Goal

Introduce a control plane that pushes routing configuration updates to data plane routers via WebSocket. The data plane must remain autonomous—it continues serving requests even if the control plane is unavailable.

## Architecture

```
┌─────────────────────────────┐
│  Control Plane              │
│  - Stores routing config    │
│  - Broadcasts to routers    │
│  - WebSocket per router     │
└──────────┬──────────────────┘
           │ WebSocket (push config)
           │
           ▼
┌─────────────────────────────┐
│  Data Plane (Router)        │
│  - Receives config          │
│  - Validates                │
│  - Atomic swap              │
│  - Serves if CP down        │
└─────────────────────────────┘
```

## Key Invariants

1. **CP never in hot path**: Routing decisions use local config only
2. **DP autonomy**: Router serves requests with last-known-good config if CP is down/slow
3. **Push model**: CP broadcasts config; DP does not poll
4. **Idempotency**: Receiving same config multiple times is safe

## Protocol (Draft)

### WebSocket Messages

**CP → DP: Config Update**
```json
{
  "type": "config_update",
  "version": 42,
  "config": {
    "default_placement": "tier3",
    "customer_to_placement": {...},
    "placement_to_endpoint": {...}
  }
}
```

**DP → CP: Ack**
```json
{
  "type": "ack",
  "version": 42,
  "status": "applied" | "rejected",
  "error": "..."
}
```

**CP → DP: Heartbeat**
```json
{
  "type": "ping"
}
```

**DP → CP: Heartbeat**
```json
{
  "type": "pong"
}
```

## Requirements

1. **CP Service**: 
   - HTTP API to update config
   - WebSocket endpoint for routers
   - Broadcast config to all connected routers
2. **DP Changes**:
   - Connect to CP on startup (with retry/backoff)
   - Reconnect on disconnect
   - Continue serving if connection lost
3. **Observability**:
   - Log config version applied
   - Metrics: time since last config update, connection status
4. **Graceful Degradation**:
   - Router starts with bootstrap config if CP unavailable
   - Router survives CP restarts

## Success Criteria

- Config changes propagate to all routers within 5 seconds
- Zero request failures during CP restarts
- Router operates normally if CP is down for hours
- Clear metrics on config staleness

## Out of Scope

- No incremental/delta updates (full config each time)
- No config pull/sync (push only)
- No multi-region CP coordination
- No authentication between CP and DP (for now)

## Next: Milestone 4

Resilience patterns (health checks, rate limiting, circuit breakers).
