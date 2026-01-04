# Milestone 2: Atomic Config Reload

**Status:** ✅ Complete

## Architectural Intent

Introduce configuration mutability while preserving the invariant that routing decisions use only local state. Routing tables become hot-reloadable from a file, but updates must be atomic—concurrent requests never observe partial configuration changes. Invalid configs must be rejected without disrupting ongoing request processing.

This milestone demonstrates the pattern: **last-known-good config is more valuable than newest config**. If a new config fails validation, the router continues serving with the previous valid config.

## Invariants Preserved

**Control plane never in request path**  
Config reload happens asynchronously in a background goroutine. Routing decisions read config via `atomic.Value.Load()`—no locks, no blocking.

**Atomic config updates**  
New config is validated before application. If valid, swapped atomically. Concurrent requests see either old config or new config, never a mix. Go's `atomic.Value` provides memory ordering guarantees.

**Graceful degradation**  
Invalid config is rejected with detailed error logs. Router continues serving with previous config. No request failures during attempted config updates.

## Configuration Format

JSON at `config/routing.json`:
```json
{
  "version": "1.0.0",
  "routingTable": {
    "acme": "tier1",
    "visa": "visa"
  },
  "cellEndpoints": {
    "tier1": "http://cell-tier1:9001",
    "visa": "http://cell-visa:9004"
  },
  "defaultPlacement": "tier3"
}
```

**Validation rules**:
- `version` must be non-empty (opaque string for tracking)
- `defaultPlacement` must exist in `cellEndpoints`
- All placements in `routingTable` must exist in `cellEndpoints`
- All endpoint URLs must parse as valid HTTP/HTTPS URLs

## Implementation Approach

**Config loader** (`internal/config/loader.go`):
- Background goroutine polls file every 5 seconds
- Computes SHA256 checksum to detect changes (avoids parsing unchanged files)
- On change: load, parse, validate, swap via `atomic.Value.Store()`
- On validation failure: log error, keep current config

**Router interface** (`internal/routing/router.go`):
- Router depends on `ConfigProvider` interface, not concrete `Loader` type
- Reads config via `GetRoutingTable()`, `GetCellEndpoints()`, `GetDefaultPlacement()`
- Each read is an atomic operation; config snapshot is consistent

**Startup behavior**:
- Fail fast if initial config is missing or invalid
- No hardcoded fallback; config file is required
- Environment variable: `CONFIG_PATH` (default: `config/routing.json`)

## Failure Modes

| Scenario | Behavior | Rationale |
|----------|----------|-----------|
| Initial config missing | Router exits (fail fast) | Cannot serve without routing config |
| Initial config invalid | Router exits (fail fast) | Prefer visibility over degraded state at startup |
| Reload: file unreadable | Log error, keep current config | Transient filesystem issue should not break routing |
| Reload: invalid JSON | Log error, keep current config | Malformed file should not disrupt serving |
| Reload: validation fails | Log error, keep current config | Invalid config should not replace valid config |
| Reload: partial file write | Checksum mismatch prevents load | Avoids loading incomplete config |

The pattern: **fail fast on startup, gracefully degrade during runtime**. Initial config errors are unrecoverable; runtime config errors are recoverable.

## Testing

**Unit tests** (`internal/config/config_test.go`, `loader_test.go`): 13 test cases covering parsing, validation, hot-reload, last-known-good behavior. Tests verify that invalid configs are rejected and previous config remains active.

**Manual verification**:
1. Start router: `docker compose up --build`
2. Check initial config: `curl http://localhost:8080/debug/config`
3. Edit `config/routing.json` (add routing key, change endpoint URL)
4. Wait 5-10 seconds, check `/debug/config` for updated `last_reload_at`
5. Break config (empty version field), verify logs show validation error and routing continues

**Debug endpoint**: `GET /debug/config` returns current version and last reload timestamp. Sufficient metadata to verify config propagation without inspecting router internals.

## Deferred to Future Milestones

- File watching via inotify/fsnotify (using polling for simplicity)
- SIGHUP signal handling for explicit reload
- Config versioning or rollback history
- Gradual rollout or canary deployments
- HTTP API for config updates
- Centralized config distribution (M3)

This milestone adds configuration flexibility while maintaining all M1 invariants. Routing decisions remain local; config reload is asynchronous; invalid configs cannot break routing.

Add control plane for centralized configuration management and dynamic service discovery.


## Next: Milestone 3

Control plane push via WebSocket with data plane autonomy.
