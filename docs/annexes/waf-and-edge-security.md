# WAF and Edge Security

## Motivation

Web Application Firewalls (WAF) inspect HTTP requests for malicious patterns (SQL injection, XSS, known exploits) and block or challenge suspicious traffic. In a cell-based architecture, WAF placement relative to routing and cells changes blast radius, latency, and failure modes.

This annex explores where WAF fits in the request path, tradeoffs between global edge enforcement and per-cell enforcement, and how WAF failures interact with routing decisions.

## WAF Placement Options

### Global Edge WAF (Before Routing)

**Pattern**: WAF runs at the edge (CDN, edge proxy, API gateway) before requests reach the router. Malicious traffic is blocked before routing decisions are made.

**Pros**:
- Protects entire fleet (routers, cells) from malicious traffic
- Reduces load on routers (blocked requests never enter routing layer)
- Centralized rule management (single WAF config for all cells)
- Lower latency for legitimate traffic (malicious requests filtered early)

**Cons**:
- WAF becomes single point of failure (if WAF fails, all traffic blocked or bypassed?)
- Can't enforce per-cell security policies (all cells see same WAF rules)
- False positives impact all tenants (overly aggressive rules block legitimate traffic globally)

**When acceptable**: Uniform security policies across all cells, WAF availability is high, shared infrastructure for threat detection.

### Per-Cell WAF (After Routing)

**Pattern**: Each cell runs its own WAF. Router forwards requests to cells; each cell inspects and enforces security policies locally.

**Pros**:
- Cell-specific security policies (dedicated cells can have stricter rules)
- Failure isolation (WAF failure in one cell doesn't impact others)
- Per-tenant customization (tenant-specific WAF rules in dedicated cells)

**Cons**:
- Malicious traffic reaches router (wasted proxy bandwidth, router CPU)
- Higher latency (WAF inspection happens after routing, inside cell)
- Distributed rule management (each cell must sync WAF rules)

**When acceptable**: Heterogeneous security requirements, cells have independent security teams, failure isolation is critical.

### Hybrid (Edge + Per-Cell)

**Pattern**: Edge WAF enforces baseline security (block obvious exploits, rate limit by IP). Per-cell WAF enforces cell-specific policies (tenant-specific rules, application-layer validation).

**Pros**:
- Baseline protection for all traffic, custom policies for specific cells
- Failure isolation (edge WAF failure doesn't break per-cell enforcement)
- Flexible (global rules at edge, tenant rules at cell)

**Cons**:
- Complexity (two WAF layers to manage, potential rule conflicts)
- Higher latency (two inspection passes per request)
- Cost (running WAF at both layers)

**When acceptable**: Large-scale deployments, multi-tenant SaaS, security requirements vary by tenant.

## Failure Modes

**WAF unavailable**: If WAF service is unreachable (edge WAF down, per-cell WAF crashed), router must decide:
- **Fail open**: Forward requests without WAF inspection (risk: malicious traffic reaches cells)
- **Fail closed**: Reject all requests with 503 (risk: availability impact, all traffic blocked)

**False positives**: Legitimate requests blocked by WAF. Observability must distinguish WAF blocks from routing failures. Response code `403 Forbidden` (WAF block) vs `502 Bad Gateway` (upstream failure).

**WAF bypass**: If attacker can bypass edge WAF (direct IP access, HTTP/2 smuggling), requests reach router without inspection. Per-cell WAF provides defense-in-depth.

## Routing and WAF Interaction

**Pre-routing inspection**: WAF sees raw request before routing decision. Can't enforce per-cell rules (doesn't know target cell yet). Can block based on request patterns only.

**Post-routing inspection**: WAF knows target cell (routing decision made). Can enforce cell-specific rules. But malicious traffic already reached router (wasted bandwidth).

**Routing key inspection**: WAF can inspect `X-Routing-Key` for anomalies (unexpected tenant IDs, header injection attempts). Prevents routing to unauthorized cells.

## Future Exploration

These questions represent design decisions that depend on operational context:

- **Rule distribution**: WAF rules could be pushed alongside routing config (unified control plane) or managed separately (dedicated security pipeline). Unified is simpler; separate allows security team autonomy.
- **Challenge flows**: CAPTCHA/JS challenges require session state. Options: stateless signed tokens, edge KV storage, or delegating challenges to cells.
- **WAF vs routing rate limits**: WAF-based limiting (per-IP, per-fingerprint) operates on different signals than routing-key limiting. Keeping them separate preserves clarity; unifying reduces config surface.
- **False positive handling**: Per-tenant allowlists in dedicated cells enable self-service. Global allowlists require security review but apply fleet-wide.
- **Bypass detection**: Correlating edge WAF logs with cell WAF logs detects direct-IP access. Requires log aggregation infrastructure.
