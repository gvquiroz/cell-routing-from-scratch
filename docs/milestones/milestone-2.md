# Milestone 2: Hot-Reload Configuration

**Status:** ✅ Complete

## Goal

Load routing configuration from a file and support hot-reload without downtime. The router:
- Reads routing tables from a JSON config file
- Validates configuration before applying
- Atomically swaps to new config using `atomic.Value`
- Falls back to last-known-good config on validation failure
- Continues serving with existing config if reload fails

## Architecture

```
┌─────────────────┐
│  Config File    │
│  routing.json   │
└────────┬────────┘
         │ (polled every 5s, checksum-based)
         ▼
┌─────────────────────────────┐
│  Config Loader              │
│  - LoadFromFile()           │
│  - Validate()               │
│  - atomic.Value swap        │
└────────┬────────────────────┘
         │ (atomic read)
         ▼
┌─────────────────────────────┐
│  Router                     │
│  (uses ConfigProvider)      │
└─────────────────────────────┘
```

## Config Format

```json
{
  "version": "1.0.0",
  "routingTable": {
    "acme": "tier1",
    "globex": "tier2",
    "initech": "tier3",
    "visa": "visa"
  },
  "cellEndpoints": {
    "tier1": "http://cell-tier1:9001",
    "tier2": "http://cell-tier2:9002",
    "tier3": "http://cell-tier3:9003",
    "visa": "http://cell-visa:9004"
  },
  "defaultPlacement": "tier3"
}
```

Location: `config/routing.json`

## Implementation

### Package Structure

```
internal/
├── config/
│   ├── config.go         # Config struct, LoadFromFile, Validate
│   ├── config_test.go    # Parsing and validation tests
│   ├── loader.go         # Hot-reload logic with atomic.Value
│   └── loader_test.go    # Reload and last-known-good tests
├── debug/
│   └── handler.go        # Debug endpoint handler
└── routing/
    └── router.go         # Updated to use ConfigProvider interface
```

### Core Invariants

1. **Atomic reads**: Router reads config via `ConfigProvider` interface methods
2. **Immutable configs**: Each `Config` struct is immutable after creation
3. **No blocking**: Config reads never block requests
4. **Last-known-good**: Invalid configs are rejected, previous config stays active
5. **Checksum-based reload**: Only reloads when file SHA256 changes

### Validation Rules

- `version` must be non-empty
- `defaultPlacement` must exist in `cellEndpoints`
- All placements referenced in `routingTable` must exist in `cellEndpoints`
- All endpoint URLs must be valid (parseable by `url.Parse`)

### Hot Reload Mechanism

1. Background goroutine polls config file every 5 seconds
2. Computes SHA256 checksum of file content
3. If checksum changed:
   - Load and parse config
   - Validate thoroughly
   - If valid: atomically swap via `atomic.Value.Store()`
   - If invalid: log error, keep current config
4. Router reads config atomically on each request

### Debug Endpoint

`GET /debug/config` returns:
```json
{
  "version": "1.0.0",
  "last_reload_at": "2026-01-03T20:45:12Z"
}
```

### Startup Behavior

- **Fail fast**: If initial config is missing or invalid, router exits with error
- Environment variable: `CONFIG_PATH` (default: `config/routing.json`)
- No fallback to hardcoded config (enforces config-driven approach)

## Testing

### Unit Tests
```bash
go test ./internal/config -v
```

12 test cases covering:
- Config parsing (valid, invalid JSON, missing file)
- Validation (missing version, invalid URLs, unknown placements)
- Hot reload (successful reload, keeps last-known-good on failure)
- Atomic swap behavior

All Milestone 1 routing tests continue to pass unchanged.

### Manual Testing

1. Start services:
```bash
docker-compose up --build
```

2. Verify initial config:
```bash
curl http://localhost:8080/debug/config
```

3. Test routing with initial config:
```bash
curl -H "X-Routing-Key: visa" http://localhost:8080/
```

4. Update config file:
```bash
# Edit config/routing.json to add new routing key
# or change endpoint URL
```

5. Wait 5-10 seconds, verify reload:
```bash
curl http://localhost:8080/debug/config
# Check that last_reload_at timestamp updated
```

6. Test with invalid config (should keep last-known-good):
```bash
# Edit config/routing.json to have empty version
# Check logs - should see "Config reload failed: validation error"
# Requests continue working with old config
```

## Failure Modes

| Scenario | Behavior |
|----------|----------|
| Initial config missing | Router exits with error (fail fast) |
| Initial config invalid | Router exits with error (fail fast) |
| Hot reload: file unreadable | Log error, keep current config |
| Hot reload: invalid JSON | Log error, keep current config |
| Hot reload: validation fails | Log error, keep current config |
| Hot reload: partial write | Checksum mismatch prevents load |

## Success Criteria (All Met)

- ✅ Config changes applied within 5-10 seconds (polling interval)
- ✅ Zero request failures during reload (atomic swap)
- ✅ Invalid config rejected with clear error messages in logs
- ✅ Existing config preserved on validation failure
- ✅ No external dependencies (standard library only)
- ✅ All Milestone 1 tests continue passing

## Out of Scope (Future Milestones)

- ❌ fsnotify/inotify (using polling for simplicity)
- ❌ SIGHUP signal handling
- ❌ Config versioning/history
- ❌ Gradual rollout (instant atomic swap)
- ❌ Config hot-reload via HTTP API
- ❌ Control plane integration (M3)

## Next: Milestone 3

Add control plane for centralized configuration management and dynamic service discovery.


## Next: Milestone 3

Control plane push via WebSocket with data plane autonomy.
