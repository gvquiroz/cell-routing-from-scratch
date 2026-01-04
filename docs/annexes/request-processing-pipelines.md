# Request Processing Pipelines

## Motivation

Modern ingress systems must compose multiple concerns—security (WAF, auth), traffic control (rate limiting), observability (logging, tracing), and routing—without creating brittle coupling. Each concern needs access to the request at different stages: some inspect before routing decisions (auth), others after (upstream retries). Some modify the request (header injection), others terminate it early (rate limit exceeded).

A **request processing pipeline** models this as a sequence of stages. Each stage receives a request, performs its logic, and either continues to the next stage or terminates the request. This pattern appears in HTTP middleware (Go, Node.js), proxy phases (Nginx, Pingora), and filter chains (Envoy). The architecture is identical; syntax varies by runtime.

Pipelines enable separation of concerns, early rejection (fail-fast), and clear trust boundaries. They're fundamental to understanding how edge proxies, API gateways, and service meshes compose behavior.

## Pipeline Model

A pipeline is an ordered sequence of **stages** (also called middleware, phases, or filters):

```
Request → [Stage 1] → [Stage 2] → ... → [Stage N] → Upstream
            ↓           ↓                   ↓
         Response    Response            Response
```

Each stage can:
1. **Inspect**: Read request metadata (headers, method, path) without modification
2. **Transform**: Modify request (add/remove headers, rewrite URL)
3. **Short-circuit**: Return response immediately, skip remaining stages
4. **Continue**: Pass request to next stage

**Order matters**: Auth must run before routing (can't route unauthenticated requests). WAF should run early (block malicious traffic before expensive operations). Logging runs last (capture final routing decision).

**Execution models**:
- **Sequential**: Stage N+1 runs only after stage N completes. Simplest; adds latency linearly.
- **Concurrent with barriers**: Independent stages run in parallel, synchronize at barriers (e.g., all auth checks complete before routing).
- **Async/event-driven**: Stage completion triggers next stage. Reduces blocking but complicates error handling.

Most ingress systems use sequential execution—latency overhead is typically <1ms per stage, acceptable for total pipeline of 5–10 stages.

## Performance Implications

**Early rejection**: Pipelines enable fail-fast. If rate limiter rejects request in stage 2, stages 3–N never execute. This protects expensive operations (routing table lookups, upstream proxying) from overload.

Example pipeline:
```
1. Parse headers (1ms)
2. Rate limiting (0.5ms, may reject)
3. WAF inspection (2ms, may block)
4. Authentication (1ms, may fail)
5. Routing decision (0.5ms)
6. Upstream proxy (50ms)
```

If rate limiting rejects 10% of traffic, those requests terminate at 1.5ms instead of 55ms. Rejected requests consume minimal CPU and memory.

**Request buffering**: Some stages require full request body (WAF inspection, signature validation). Others stream (proxying to upstream). Buffering decisions affect memory usage and latency. Pipelines make this explicit—WAF stage declares "I need buffered body", routing stage declares "I stream".

**Parallelizable stages**: Health checks, logging, and metrics can run concurrently with request processing. Pipeline model allows stages to declare "I'm async" and execute off the critical path.

## Trust Boundaries

Pipelines enforce **progressive validation**—each stage assumes previous stages have validated their concerns.

**Example trust chain**:
1. **TLS termination**: Validates client cert, extracts identity. Downstream stages trust TLS-derived identity.
2. **Authentication**: Validates token/session. Sets `X-Authenticated-User`. Downstream stages trust this header.
3. **Routing**: Reads `X-Authenticated-User`, determines placement. Upstream sees only authenticated, routed requests.

If authentication stage is bypassed (bug, misconfiguration), routing stage routes unauthenticated requests—trust boundary is broken. Pipeline order must match trust dependencies.

**Immutability between stages**: Some systems pass read-only request context between stages to prevent tampering (stage 3 can't remove headers added by stage 2). Others rely on ordering—later stages can't affect earlier stages by definition (sequential execution).

## Go

Go's `http.Handler` interface enables middleware composition. Each middleware wraps the next handler

**Characteristics**:
- Sequential execution (synchronous function calls)
- Short-circuit via early return
- Transform via `r.Header.Set()`
- Order defined by nesting: outermost runs first

**Performance**: Middleware overhead is function call cost (~10ns per stage). Negligible compared to I/O (rate limiter lookup, auth token validation).

## Pingora Phases

Pingora organizes request processing into **phases**: request filter, upstream peer selection, response filter. Each phase is an async function in the proxy application:

**Characteristics**:
- Async execution (Tokio runtime)
- Short-circuit via `Ok(true)` (terminates request)
- Transform via `session` and `ctx` mutation
- Order defined by method call sequence within phase

**Performance**: Async overhead is task scheduling (~100ns). Actual I/O (rate limiter, auth service) dominates. Async allows non-blocking I/O—while waiting for auth service, Tokio schedules other requests.

## Limitations and Tradeoffs

**Ordering dependencies**: As pipelines grow, ordering constraints become fragile.

**Error handling**: If stage 5 fails, how do stages 1–4 clean up?

**State mutation**: Stages that modify request context (add headers, rewrite URL) can create unintended coupling. Stage 5 might depend on header added by stage 2—implicit dependency. Immutable context between stages prevents this but adds overhead (copy-on-write).

## Takeaway

Request processing pipelines are the foundational architecture for composing ingress concerns. They enable early rejection, enforce trust boundaries, and separate cross-cutting concerns from routing logic.

Go middleware and Pingora phases are runtime-specific implementations of the same model: ordered stages, short-circuit capability, request transformation. Understanding the model—not the syntax—transfers across systems (Envoy filters, Nginx modules, service mesh sidecars).
