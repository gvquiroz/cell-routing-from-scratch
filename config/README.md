# Configuration Files

This project uses separate config files to clarify the role of each component in the control plane/data plane architecture.

## Files

### `routing.json`
**Used by:** Control Plane only  
**Purpose:** Authoritative source of routing configuration  
**Watched by:** CP polls this file every 5 seconds for changes  
**Updates:** Edit this file and CP will broadcast changes to all connected DPs

Example: Change version or add a new customer mapping, CP detects the change and pushes to all routers.

### `dataplane-initial.json`
**Used by:** Data Plane (router) for initial load only  
**Purpose:** Bootstrap configuration before CP connection  
**Behavior:** 
- Loaded once at startup (version 0.0.1 by default)
- Immediately replaced by CP's first config snapshot (version 1.0.0+)
- Never watched for changes when `CONTROL_PLANE_URL` is set

This separation demonstrates that:
- CP is the single source of truth for config distribution
- DPs don't need to watch files once connected to CP
- Initial DP config can be stale/minimal (will be overwritten)

## Modes

### With Control Plane (default in docker-compose)
```
DP loads: dataplane-initial.json (version 0.0.1)
     ↓
DP connects to CP
     ↓
CP sends: routing.json (version 1.0.0)
     ↓
DP config updated via CP pushes only
```

### Without Control Plane (file-only mode)
```bash
# No CONTROL_PLANE_URL set
DP loads: routing.json
     ↓
DP watches routing.json for changes (5s poll)
     ↓
DP hot-reloads on file changes
```

Set `CONTROL_PLANE_URL=""` or unset it to use file-only mode.
