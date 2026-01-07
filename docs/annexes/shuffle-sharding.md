# Shuffle Sharding

## Motivation

Cell-based architectures isolate failure domains—one cell's failure doesn't impact others. But what happens when multiple tenants share a cell (shared tiers)? If one tenant causes a cell failure (resource exhaustion, application bug, infinite loop), all tenants on that cell are impacted.

**Shuffle sharding** reduces blast radius by assigning each tenant to a random subset of cells from a larger pool. Even if one cell fails, most tenants are unaffected because they're also assigned to healthy cells. This annex explores how shuffle sharding interacts with cell routing, health-aware failover, and placement decisions.

## Core Concept

Instead of mapping tenants to single cells:
```
tenant-A → cell-1
tenant-B → cell-1
tenant-C → cell-1
```

Shuffle sharding assigns each tenant to multiple cells (a "shard"):
```
tenant-A → [cell-1, cell-3, cell-7]
tenant-B → [cell-2, cell-5, cell-8]
tenant-C → [cell-1, cell-4, cell-9]
```

Requests for `tenant-A` route to a healthy cell from its shard `[cell-1, cell-3, cell-7]`. If `cell-1` fails, router routes to `cell-3`. If both fail, route to `cell-7`.

**Blast radius**: If `cell-1` fails, only tenants with `cell-1` in their shard are impacted (statistically ~10–30% depending on shard size). Other tenants route to their healthy cells unaffected.

## Shard Assignment

**Random shuffle**: Each tenant's shard is a random subset of size `k` from a pool of `N` cells. Randomness ensures even distribution—no two tenants have identical shards unless by chance.

**Deterministic**: Shard assignment is stable (tenant always gets same shard) and reproducible (same tenant ID, same seed → same shard). Allows routers to compute shards locally without coordination.

**Shard size tradeoff**: Larger shards (`k`) reduce blast radius (more failover options) but increase operational complexity (more cells to manage per tenant). Typical: `k = 3–5` from pool of `N = 20–50` cells.

## Routing Model Changes

Baseline router maps `routingKey → placementKey → endpoint` (single endpoint per placement). Shuffle sharding requires `routingKey → shard → healthy_endpoint` (choose one healthy endpoint from shard).

**Placement config** becomes:
```json
{
  "routingTable": {
    "tenant-A": "shard-A"
  },
  "shards": {
    "shard-A": ["cell-1", "cell-3", "cell-7"]
  },
  "cellEndpoints": {
    "cell-1": "http://cell-1:9001",
    "cell-3": "http://cell-3:9001",
    "cell-7": "http://cell-7:9001"
  }
}
```

Router selects first healthy cell from `shard-A`. If all cells are unhealthy, fall back to default placement.

## Health-Aware Routing Interaction

Shuffle sharding **requires** health-aware routing. Without health checks, router can't distinguish healthy cells from failed cells—shuffle sharding degrades to random selection.

**Health check per cell**: Router runs health checks for all cells in all shards (not just the first cell). Health state is per-cell, shared across tenants.

**Failover order**: Within a shard, router tries cells in deterministic order (e.g., sorted by cell ID). This avoids thundering herd—all tenants with same shard fail over to same cells in same order.

**Circuit breaker per cell**: Circuit breakers track state per cell, not per shard. If `cell-1` circuit opens, all shards containing `cell-1` avoid it during cooldown.

## Operational Complexity

**Cell pool management**: Adding/removing cells from pool requires recomputing shards. Stable shard assignment (deterministic shuffle) means tenant→shard mapping doesn't change unless explicitly updated.

**Config size**: Shard definitions increase config size. For 1000 tenants with shard size 5, config contains 5000 cell references. Acceptable for control plane push (few KB), but impacts config reload latency.

## Blast Radius Calculation

For `k`-sized shards from pool of `N` cells:
- Probability a specific cell is in a tenant's shard: `k / N`
- Expected tenants impacted by one cell failure: `total_tenants × (k / N)`

Example: 1000 tenants, `k = 5`, `N = 50`:
- Each tenant has 5 cells in shard
- Each cell appears in ~10% of shards
- One cell failure impacts ~100 tenants (10%), not all 1000

## Future Exploration

Shuffle sharding introduces operational complexity that scales with tenant count:

- **Assignment algorithm**: Fisher-Yates with tenant ID as seed provides deterministic, reproducible shards. Hash-based partitioning (consistent hashing) enables incremental rebalancing.
- **Stable rebalancing**: When cells are added/removed, minimize tenant shard changes using consistent hashing or explicit migration tables.
- **Cross-cell state**: If a tenant's shard spans regions, session state either stays region-local (simpler) or syncs across cells (complex, rarely justified).
- **Cost model**: Shuffle sharding requires larger cell pools than simple tiering. Justify when blast radius reduction outweighs infrastructure cost—typically at 100+ tenants in shared tiers.
- **Effectiveness metrics**: Measure blast radius reduction (% tenants impacted per cell failure) vs operational overhead (config size, health check fanout).
