# Caching and CDN Architecture

## Motivation

Modern CDNs are not separate systems layered in front of ingress. They are programmable request pipelines where caching is one possible outcome.

In practice, caching is a decision point inside the same request-processing pipeline that already handles security, identity, routing, and proxying.

## Where Caching Sits in the Pipeline

Cache lookup can occur at multiple pipeline positions. Each placement has different correctness requirements and operational characteristics.

### Global Edge Cache (Before Routing)

Pipeline: `Request → Cache Lookup → [miss] → WAF → Auth → Routing → Cell → Upstream`

Cache sits at the global edge, before routing decisions. Cache key is derived from request path, query parameters, and select headers—no routing key or cell identity.

**Characteristics**:
- Highest cache hit ratio (all tenants share cache space for public assets)
- Cache misses traverse full pipeline (WAF, auth, routing, cell selection)
- Purge targets all edge nodes (global invalidation)

**Failure semantics**: Cache remains available during cell failures. Stale content can be served even if all cells for a tenant are unhealthy. This violates the cell-isolation invariant—cache becomes a shared fate system.

**Use case**: Public CDN assets (JavaScript bundles, images, fonts). Low cache key cardinality. Invalidation is rare or versioned (asset hashing).

### Per-Cell Cache (After Routing)

Pipeline: `Request → WAF → Auth → Routing → [Cell A Cache] → [miss] → Upstream`

Cache sits within each cell, after routing decisions. Cache key includes routing key or placement key—cache is scoped to the cell.

**Characteristics**:
- Lower hit ratio (cache partitioned per cell)
- Cache misses only traverse upstream proxying (WAF, auth, routing already complete)
- Purge targets specific cells (scoped invalidation)
- Tenant isolation via cell boundary (cross-tenant cache poisoning impossible)

**Failure semantics**: Cache fails with the cell. No stale serving during cell outages unless fallback cells have independent caches. Consistent with cell-isolation invariant.

**Use case**: Dynamic, tenant-specific content (dashboards, API responses, rendered HTML). High cache key cardinality. Frequent invalidation.

### Two-Tier Cache (Edge + Shield)

Pipeline: `Request → [Edge Cache] → [miss] → Routing → [Shield Cache] → [miss] → Upstream`

Two cache layers: global edge (high hit ratio, public assets) and per-cell or per-region shield (lower hit ratio, tenant-scoped).

**Characteristics**:
- Edge cache serves public assets, shields cache private/dynamic content
- Reduces origin load even for cache-hostile workloads (shield absorbs misses)
- Purge requires two-tier coordination (edge + shield invalidation)
- Complexity: cache key design must be consistent across tiers

**Failure semantics**: Edge cache available during shield or origin failures (serves stale public assets). Shield cache fails with the cell or region. Hybrid isolation—public content has shared fate, private content respects cell boundaries.

**Use case**: Mixed workloads (public assets + private APIs).

## Cache Key and Tenant Isolation

### Cache Key Cardinality

High cardinality cache keys (many unique combinations) reduce hit ratio but improve correctness. Tradeoff:
- **Low cardinality**: High hit ratio, risk of over-caching (serving stale or wrong content)
- **High cardinality**: Low hit ratio, safer (each tenant/user has independent cache entries)

Multi-tenant systems should bias toward high cardinality—correctness over hit ratio. Purge is simpler when cache is partitioned by tenant (invalidate all entries for `X-Tenant-ID: alice`).

## Invalidation and Change Management

Cache invalidation is the hardest problem in distributed systems. Two strategies: versioned assets (avoid purge) and explicit purge (invalidate on change).

### Versioned Assets

Immutable assets with version identifiers in URLs:
- `/assets/bundle-abc123.js` (content hash)
- `/v2/api/resource` (API version)

Cache these indefinitely (long TTL). No purge required—new versions have new URLs, old versions expire naturally. This is the preferred strategy for public assets.

**Tradeoff**: Requires client-side version coordination (HTML must reference correct bundle hash). Not applicable to APIs where URL is fixed.

### Explicit Purge

Invalidate cache entries on change:
- **Soft purge**: Mark entries stale, but serve stale if origin is unavailable. Graceful degradation.
- **Hard purge**: Delete entries immediately. Cache miss on next request. Risks thundering herd if origin is slow.

**Purge scope**:
- **Global purge**: Invalidate all edge nodes. High fanout, eventual consistency (purge propagation delay).
- **Per-tenant purge**: Invalidate entries matching `X-Tenant-ID: alice`. Lower fanout if cache is partitioned by tenant.

**Consistency**: Purge is not instantaneous. Between purge initiation and propagation, stale content may be served. Multi-tier caches amplify this (edge and shield must both purge).
