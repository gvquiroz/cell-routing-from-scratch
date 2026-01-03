# Milestone 2: Hot-Reload Configuration

**Status:** 📋 Planned

## Goal

Load routing configuration from a file and support hot-reload without downtime. The router should:
- Read routing tables from a JSON/YAML config file
- Validate configuration before applying
- Atomically swap to new config
- Fall back to last-known-good config on validation failure
- Continue serving with existing config if reload fails

## Architecture

```
┌─────────────────┐
│  Config File    │
│  routes.yaml    │
└────────┬────────┘
         │ (watch for changes)
         ▼
┌─────────────────────────────┐
│  Config Loader              │
│  - Parse                    │
│  - Validate                 │
│  - Atomic swap              │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│  Router                     │
│  (uses current config)      │
└─────────────────────────────┘
```

## Config Format (Draft)

```yaml
default_placement: tier3

customer_to_placement:
  acme: tier1
  globex: tier2
  initech: tier3
  visa: visa

placement_to_endpoint:
  tier1: http://cell-tier1:9001
  tier2: http://cell-tier2:9002
  tier3: http://cell-tier3:9003
  visa: http://cell-visa:9004
```

## Requirements

1. **File Watching**: Detect changes to config file (inotify/fsnotify or polling)
2. **Validation**: 
   - All placements in customer_to_placement have endpoints
   - Default placement has an endpoint
   - Endpoint URLs are valid
3. **Atomic Swap**: Use RWMutex or atomic.Value to swap config
4. **Fallback**: Keep previous config if new one is invalid
5. **Logging**: Log successful reloads, validation errors, and rollbacks
6. **Signal**: Support `SIGHUP` to trigger manual reload

## Success Criteria

- Config changes applied within 1-2 seconds
- Zero request failures during reload
- Invalid config rejected with clear error messages
- Existing config preserved on validation failure

## Out of Scope

- No complex validation rules
- No gradual rollout (instant swap)
- No config versioning/history
- No control plane (still file-based)

## Next: Milestone 3

Control plane push via WebSocket with data plane autonomy.
