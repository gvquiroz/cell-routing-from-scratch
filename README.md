# Cell Routing from Scratch

Learn how cell-based architectures route traffic by building an educational reverse proxy from first principles. This project demonstrates control plane/data plane separation, local routing decisions, and resilience patterns through clear, incremental milestones.

## Quick Start

```bash
# Start router + 4 demo cells
docker compose up --build

# Test routing
curl -H "X-Routing-Key: visa" http://localhost:8080/
curl -H "X-Routing-Key: acme" http://localhost:8080/

# Run automated tests
./test-routing.sh
```

## How It Works

**Client** → **Router** (reads `X-Routing-Key`) → **Target Cell**

```bash
# Dedicated cell
curl -H "X-Routing-Key: visa" http://localhost:8080/
# → X-Routed-To: visa, X-Route-Reason: dedicated

# Shared tier (tier1)
curl -H "X-Routing-Key: acme" http://localhost:8080/
# → X-Routed-To: tier1, X-Route-Reason: tier

# Unknown → defaults to tier3
curl -H "X-Routing-Key: unknown" http://localhost:8080/
# → X-Routed-To: tier3, X-Route-Reason: default
```

Response headers show routing decision: `X-Routed-To`, `X-Route-Reason`

## Milestones

| # | Status | Description | Docs |
|---|--------|-------------|------|
| **M1** | ✅ Complete | Static in-memory routing with streaming proxy | [Details](docs/milestones/milestone-1.md) |
| **M2** | 📋 Planned | Hot-reload config from file | [Details](docs/milestones/milestone-2.md) |
| **M3** | 📋 Planned | Control plane push via WebSocket | [Details](docs/milestones/milestone-3.md) |
| **M4** | 📋 Planned | Health checks, rate limiting, circuit breakers | [Details](docs/milestones/milestone-4.md) |
| **M5** | 📋 Planned | Reimplement in Pingora for comparison | [Details](docs/milestones/milestone-5.md) |

[📚 Milestone Overview](docs/milestones/README.md)

## Project Structure

```
cmd/router/       # Router entrypoint
cmd/cell/         # Demo cell server
internal/
├── routing/      # Routing logic + tests
├── proxy/        # HTTP proxy handler
└── logging/      # Structured logging
docs/milestones/  # Detailed specs
```

## What You'll Learn

- **CP/DP Separation**: Control plane never in hot path; routing is local and fast
- **Graceful Degradation**: Router stays up even if control plane is down
- **Resilience Patterns**: Health checks, circuit breakers, rate limiting (M4)
- **Proxy Internals**: Streaming, connection pooling, timeouts
- **Go vs Pingora**: Stdlib proxy vs edge-grade proxy runtime (M5)

## Development

```bash
# Run tests
go test ./...

# Run locally without Docker
CELL_NAME=tier1 PORT=9001 go run cmd/cell/main.go &
go run cmd/router/main.go
```

See [Milestone 1 docs](docs/milestones/milestone-1.md) for complete implementation details.

## License

MIT
	•	control-plane / data-plane separation,
	•	health-aware routing and failover,
	•	rate limiting, retries, and circuit breaking,

while preserving the core invariant shared by production systems: routing decisions are fast, local, and independent of control plane availability.

The initial implementation uses Go for clarity and approachability. A later milestone re-implements the same behavior using Cloudflare’s Pingora to compare design tradeoffs between application-level proxies and edge-grade proxy runtimes.

This repository is not a framework or a production gateway. It is an educational artifact intended to make the architecture and tradeoffs of cell routing explicit and understandable.
