# Authentication and Routing Keys

## Motivation

The baseline router trusts `X-Routing-Key` unconditionally. This is an explicit non-goal: authentication is assumed to happen upstream, and the router's job is purely routing logic. But production systems must answer: **where is the trust boundary, and how is routing metadata validated?**

This annex explores options for securing routing keys, where authentication fits in the request pipeline, and why separating auth from routing logic matters. It also examines how API gateways compose authentication with routing decisions.

## Trust Boundaries and Pipeline Placement

Authentication validates identity and authorization. Routing uses that validated identity to determine cell placement. These are separate concerns that must execute in order:

```
Request → [TLS Termination] → [Authentication] → [Authorization] → [Routing] → [Proxy to Cell]
```

**Trust boundary**: Everything after authentication trusts that the identity is valid. If authentication is bypassed (misconfiguration, bug), routing decisions become meaningless and unauthorized requests route to cells.

### Where Authentication Sits

**Before routing (typical)**:
- Authentication extracts tenant ID, user ID, or API key
- Sets trusted headers: `X-Authenticated-User`, `X-Tenant-ID`
- Routing reads these headers to determine placement key
- Cell receives pre-authenticated requests

**Hybrid (API gateway pattern)**:
- Authentication at edge extracts coarse identity (tenant ID)
- Routing uses tenant ID to select cell
- Cell performs fine-grained authorization (user permissions within tenant)

Most systems use "before routing" to keep cells simple and trust boundaries explicit.

## Securing Routing Keys

Three approaches to ensure routing keys derive from authenticated identity:

### 1. Header Injection (Trusted Proxy)

Authentication service validates credentials, extracts tenant identity, and injects trusted headers. Router trusts these headers because they come from authenticated proxy.

**Flow**:
```
Client → [Edge Proxy with Auth] → [Router] → [Cell]
         Sets X-Tenant-ID=alice
```

**Requirements**:
- Network trust: Router must reject requests from untrusted sources (only accept from authenticated proxy)
- Header stripping: Edge proxy must strip client-provided `X-Tenant-ID` headers before adding authenticated values
- Routing configuration: Router maps `X-Tenant-ID` to placement key

**Risk**: If router accepts direct client requests, clients can forge `X-Tenant-ID`. Network segmentation or mTLS between proxy and router is critical.

**Use case**: Simple architectures where edge proxy and router are in trusted network. Cloudflare Workers + internal routers follow this pattern.

### 2. Mutual TLS (Certificate-Based Trust)

Client certificate identifies tenant. Router extracts tenant ID from certificate subject or SAN (Subject Alternative Name).

**Flow**:
```
Client with cert → [Router validates cert] → [Extract tenant from cert subject] → [Route to cell]
```

**Requirements**:
- Certificate validation: Router verifies client cert against trusted CA
- Subject parsing: Extract tenant ID from CN (Common Name) or custom extension
- Certificate management: PKI infrastructure for issuing, rotating, revoking certs

**Advantages**:
- Strong cryptographic identity: TLS handshake proves client identity
- No application-layer auth: Identity established at transport layer
- Works for non-HTTP protocols: Identity applies to TCP connection, not HTTP request

**Risk**: Certificate revocation is operational complexity (CRL or OCSP checks). Tenant offboarding requires cert revocation and propagation.

**Use case**: B2B integrations, IoT devices, internal service mesh. Rare for multi-tenant SaaS due to certificate distribution complexity.

## API Gateway Authentication Patterns

API gateways compose authentication, authorization, rate limiting, and routing in a single pipeline. Authentication is a stage that enriches request context before routing.

### Centralized Authentication (Gateway Validates)

Gateway performs all authentication. Backend services receive pre-authenticated requests with trusted identity headers.

**Pipeline**:
```
Request → [Parse API key/JWT] → [Validate credentials] → [Extract tenant/user] 
       → [Set X-Tenant-ID, X-User-ID] → [Rate limit by tenant] → [Route to cell]
```

**Implementation patterns**:
- **API key validation**: Gateway queries key store (Redis, database) to map API key → tenant ID. Sets `X-Tenant-ID` header.
- **JWT validation**: Gateway validates JWT signature, extracts `tid` and `uid` claims, sets headers.
- **OAuth token introspection**: Gateway calls OAuth server to validate access token, receives tenant/user metadata, sets headers.

**Advantages**:
- Cells remain stateless: No auth logic in backend services
- Consistent enforcement: All requests authenticated at gateway; cells cannot receive unauthenticated traffic
- Auditing: Gateway logs all authentication events

**Tradeoffs**:
- Gateway becomes critical path: Auth validation latency added to every request
- Key distribution: Gateway must have access to auth credentials (shared secret for JWT, API key database)
- Scaling: Gateway must handle auth validation load (mitigated by caching validated tokens)

**Example: Kong + OAuth 2.0 Introspection**:
- Client sends request with `Authorization: Bearer <token>`
- Kong calls OAuth introspection endpoint: `POST /introspect` with token
- OAuth server returns `{"active": true, "tenant_id": "alice"}`
- Kong adds `X-Tenant-ID: alice` header, routes to upstream service

### Hybrid (Gateway Authenticates, Cells Authorize)

Gateway validates identity and extracts tenant ID. Cells perform fine-grained authorization (user permissions within tenant).

**Pipeline**:
```
Request → [Gateway validates JWT] → [Extract tenant ID, user ID] 
       → [Set X-Tenant-ID, X-User-ID] → [Route to cell] 
       → [Cell checks user permissions] → [Cell returns response]
```

**Advantages**:
- Clear trust boundary: Gateway ensures identity is valid; cells trust identity but enforce permissions
- Routing correctness: Gateway uses authenticated tenant ID for routing (cannot route to wrong tenant)
- Domain-specific authorization: Cells enforce rules specific to their APIs (user can read but not write)

**Tradeoffs**:
- Split responsibility: Gateway and cells both have auth logic (validation vs authorization)
- Coordination: If permissions change, cells must update; gateway does not see permission changes

**Use case**: Multi-tenant SaaS API gateways. Gateway ensures tenant isolation via routing; cells enforce user permissions within tenant.
