# Configuration Files

This project uses separate config files to clarify the role of each component in the control plane/data plane architecture.

## Files

### `routing.json`
**Used by:** Control Plane only  
**Purpose:** Authoritative source of routing configuration (legacy format)  
**Format:** Simple key-value mappings

### `routing-m4.json`
**Used by:** Router with Milestone 4 resilience features  
**Purpose:** Routing config with health checks, circuit breakers, and limits  
**Format:** Extended placement configuration with resilience settings

### `dataplane-initial.json`
**Used by:** Data Plane (router) for initial load only  
**Purpose:** Bootstrap configuration before CP connection  
**Behavior:** 
- Loaded once at startup
- Immediately replaced by CP's first config snapshot
- Never watched for changes when `CONTROL_PLANE_URL` is set

## Configuration Formats

### Legacy Format (M1-M3)
```json
{
  "version": "1.0.0",
  "routingTable": {
    "customer-id": "placement-key"
  },
  "cellEndpoints": {
    "placement-key": "http://cell:9001"
  },
  "defaultPlacement": "tier3"
}
```

### Milestone 4 Format (Health Checks + Circuit Breaking + Limits)
```json
{
  "version": "2.0.0",
  "routingTable": {
    "customer-id": "placement-key"
  },
  "placements": {
    "placement-key": {
      "url": "http://cell:9001",
      "fallback": "tier3",
      "health_check": {
        "path": "/health",
        "interval": "10s",
        "timeout": "2s"
      },
      "circuit_breaker": {
        "failure_threshold": 5,
        "timeout": "30s"
      },
      "concurrency_limit": 100,
      "max_request_body_bytes": 10485760
    }
  },
  "defaultPlacement": "tier3"
}
```

### Placement Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string | Yes | Upstream endpoint URL |
| `fallback` | string | No | Placement key to route to if unhealthy/circuit open |
| `health_check` | object | No | Active health check configuration |
| `health_check.path` | string | Yes | HTTP path for health checks (e.g., `/health`) |
| `health_check.interval` | string | Yes | Check frequency (e.g., `10s`) |
| `health_check.timeout` | string | Yes | Check timeout (e.g., `2s`) |
| `circuit_breaker` | object | No | Circuit breaker configuration |
| `circuit_breaker.failure_threshold` | int | Yes | Consecutive failures before opening |
| `circuit_breaker.timeout` | string | Yes | How long to stay open (e.g., `30s`) |
| `concurrency_limit` | int | No | Max concurrent requests to this placement |
| `max_request_body_bytes` | int64 | No | Max request body size in bytes |

## Modes

### With Control Plane (default in docker-compose)
```
DP loads: dataplane-initial.json
     ↓
DP connects to CP
     ↓
CP sends: routing.json or routing-m4.json
     ↓
DP config updated via CP pushes only
```

### Without Control Plane (file-only mode)
```bash
# No CONTROL_PLANE_URL set
DP loads: routing.json or routing-m4.json
     ↓
DP watches file for changes (5s poll)
     ↓
DP hot-reloads on file changes
```

Set `CONTROL_PLANE_URL=""` or unset it to use file-only mode.
