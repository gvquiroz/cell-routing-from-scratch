# Authentication and Routing Keys

## Motivation

The baseline router trusts `X-Routing-Key` unconditionally. This is an explicit non-goal: authentication is assumed to happen upstream, and the router's job is purely routing logic. But production systems must answer: **where is the trust boundary, and how is routing metadata validated?**

This annex explores options for securing routing keys, where authentication fits in the request path, and why separating auth from routing logic matters.

## Trust Boundaries

**Untrusted clients**: Public internet requests cannot set `X-Routing-Key` directly—clients would route themselves to arbitrary placements. Routing metadata must be injected by a trusted component after authentication.

**Trusted ingress**: The router assumes an upstream component (API gateway, ingress controller, edge proxy) has already authenticated the request and set `X-Routing-Key` based on validated identity (user ID, tenant ID, customer ID).

**Inter-service**: For service-to-service calls within a cell, `X-Routing-Key` may propagate from the original ingress request (trace context, forwarded headers). Services trust the header because they're inside the trust boundary.

## Authentication Options

### Header Injection at API Gateway

**Pattern**: API gateway authenticates request (API key, OAuth token, session cookie), extracts tenant ID from token claims, sets `X-Routing-Key: <tenant_id>`, forwards to router.

**Pros**:
- Clear separation: gateway handles auth, router handles routing
- Router remains stateless (no token validation, no user lookup)
- Easy to audit (routing key in logs, tied to authenticated identity)

**Cons**:
- Gateway is single point of trust injection
- If gateway is compromised, attacker can set arbitrary routing keys
- Doesn't prevent header tampering between gateway and router (requires mTLS or signed headers)

**When acceptable**: Gateway and router are in same trust zone (same VPC, same cluster). Network is trusted.

### JWT Claims as Routing Metadata

**Pattern**: Client presents JWT. Router validates JWT signature, extracts `tenant_id` claim, uses as routing key. No explicit `X-Routing-Key` header.

**Pros**:
- No trust boundary between gateway and router (router validates token directly)
- Tamper-proof (JWT signature verification)
- Works in zero-trust networks (router doesn't trust upstream)

**Cons**:
- Router becomes auth-aware (JWT parsing, signature validation, key rotation)
- Adds latency (cryptographic signature check per request)
- Complicates routing logic (auth failure = routing failure?)

**When acceptable**: Zero-trust architecture, router must validate tokens directly, acceptable latency overhead.

### Mutual TLS (mTLS) with Client Certificates

**Pattern**: Client presents TLS client certificate. Certificate contains tenant ID in subject or SAN. Router extracts tenant ID from cert, uses as routing key.

**Pros**:
- Cryptographically strong identity (TLS handshake validates cert)
- No token parsing or validation logic in application layer
- Works for service-to-service auth

**Cons**:
- Requires mTLS infrastructure (CA, cert issuance, rotation)
- Certificate revocation is complex (CRLs, OCSP)
- Not suitable for public-facing APIs (client cert distribution problem)

**When acceptable**: Internal service mesh, service-to-service routing, mTLS already deployed.

## Separation of Auth from Routing

**Why separate**: The router's job is to map `routingKey → placement → endpoint`. Adding authentication logic (token validation, user lookup, authorization checks) couples routing to identity systems, breaking the stateless invariant.

**Where auth belongs**: Upstream of the router—API gateway, edge proxy, service mesh ingress. These components have access to identity providers, token validation services, and user databases. Router receives pre-authenticated requests with validated routing metadata.

**Exception**: Zero-trust environments where no upstream component is trusted. Router must validate tokens directly, accepting the complexity and latency tradeoff.

## Header Tampering and Signed Metadata

If network between gateway and router is untrusted, `X-Routing-Key` can be tampered with. Options:

**mTLS between gateway and router**: Encrypted channel prevents tampering. Requires cert management.

**Signed headers**: Gateway signs routing metadata (HMAC, JWT-style signature). Router validates signature before routing. Prevents tampering but adds complexity.

**Network isolation**: Gateway and router in same VPC/namespace. Network is trusted boundary. Simplest but requires infrastructure guarantees.

## TODO / Open Questions

- **Token caching**: If router validates JWTs, should it cache validation results? Key rotation implications?
- **Auth failure semantics**: If JWT validation fails, does router return 401 or 502? Who owns the error boundary?
- **Routing key conflicts**: What if JWT claims contain multiple tenant IDs (multi-tenant users)? Who decides which tenant ID becomes routing key?
- **Observability**: How to log auth decisions without leaking PII? Routing key is logged, but what about token claims?
- **Fallback routing on auth failure**: Should router route to default placement if auth fails, or reject request immediately?

**Future expansion**: Reference implementation of JWT-based routing. Performance comparison: header injection vs JWT validation latency. Security runbook for routing key validation.
