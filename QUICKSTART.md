# Quick Start Guide

## Prerequisites
- Docker and Docker Compose installed
- Or Go 1.22+ for local development

## Run with Docker Compose

```bash
# Start all services (router + 4 cells)
docker compose up --build

# In another terminal, test the router
curl -H "X-Routing-Key: visa" http://localhost:8080/
```

## Test Different Routing Scenarios

```bash
# 1. Dedicated cell (visa)
curl -v -H "X-Routing-Key: visa" http://localhost:8080/api/test

# 2. Tier 1 (acme)
curl -v -H "X-Routing-Key: acme" http://localhost:8080/api/test

# 3. Tier 2 (globex)
curl -v -H "X-Routing-Key: globex" http://localhost:8080/api/test

# 4. Tier 3 (initech)
curl -v -H "X-Routing-Key: initech" http://localhost:8080/api/test

# 5. Unknown key (defaults to tier3)
curl -v -H "X-Routing-Key: unknown-company" http://localhost:8080/api/test

# 6. No key (defaults to tier3)
curl -v http://localhost:8080/api/test
```

## View Logs

```bash
# Router logs (structured JSON)
docker compose logs -f router

# Specific cell logs
docker compose logs -f cell-visa
```

## Stop Services

```bash
docker compose down
```

## Development

```bash
# Run tests
go test ./...

# Run specific tests
go test ./internal/routing -v

# Build locally
go build -o bin/router ./cmd/router
go build -o bin/cell ./cmd/cell
```

## Expected Response Headers

Every response from the router includes:

- `X-Routed-To`: The placement key (tier1, tier2, tier3, or visa)
- `X-Route-Reason`: Why this placement was chosen (dedicated, tier, or default)
- `X-Request-Id`: Unique request identifier (generated or propagated)
- `X-Cell-Name`: The cell that handled the request

## Routing Mappings

### Customers → Placements
- `acme` → `tier1`
- `globex` → `tier2`
- `initech` → `tier3`
- `visa` → `visa` (dedicated)
- Unknown/missing → `tier3` (default)

### Placements → Endpoints
- `tier1` → `http://cell-tier1:9001`
- `tier2` → `http://cell-tier2:9002`
- `tier3` → `http://cell-tier3:9003`
- `visa` → `http://cell-visa:9004`
