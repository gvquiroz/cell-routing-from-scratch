# cell-routing-from-scratch
Modern large-scale systems increasingly rely on cell-based architectures to limit blast radius, improve isolation, and scale independently across tenants and regions. While cells are widely discussed, the mechanics of routing requests into the correct cell are often hidden and custom build.

This repository is a learning-focused, incremental implementation of the ingress routing layer in a cell-based architecture.

Starting from first principles, it builds a minimal reverse proxy that:
	•	receives traffic on a global endpoint,
	•	identifies the tenant from trusted request metadata,
	•	resolves the target cell in-memory,
	•	and proxies the request without relying on a centralized control plane at request time.

The project evolves in clearly defined milestones, progressively adding:
	•	hot-reloaded configuration,
	•	control-plane / data-plane separation,
	•	health-aware routing and failover,
	•	rate limiting, retries, and circuit breaking,

while preserving the core invariant shared by production systems: routing decisions are fast, local, and independent of control plane availability.

The initial implementation uses Go for clarity and approachability. A later milestone re-implements the same behavior using Cloudflare’s Pingora to compare design tradeoffs between application-level proxies and edge-grade proxy runtimes.

This repository is not a framework or a production gateway. It is an educational artifact intended to make the architecture and tradeoffs of cell routing explicit and understandable.
