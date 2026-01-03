# Milestones

This directory contains detailed specifications for each project milestone.

## Overview

| Milestone | Status | Description |
|-----------|--------|-------------|
| [M1: Static Routing](milestone-1.md) | ✅ Complete | In-memory routing with trusted header, streaming proxy, explainability headers |
| [M2: Hot-Reload](milestone-2.md) | 📋 Planned | File-based config with atomic swap and validation |
| [M3: Control Plane Push](milestone-3.md) | 📋 Planned | WebSocket-based config distribution with DP autonomy |
| [M4: Resilience](milestone-4.md) | 📋 Planned | Health checks, rate limiting, retries, circuit breakers |
| [M5: Pingora](milestone-5.md) | 📋 Planned | Reimplement M1-M4 in Pingora for comparison |

## Key Principles

Each milestone:
- ✅ Is runnable and testable
- ✅ Has clear scope and non-goals
- ✅ Builds on previous milestones
- ✅ Includes documentation and examples
- ✅ Focuses on teaching CP/DP separation

## Reading Order

1. Start with [Milestone 1](milestone-1.md) to understand the baseline implementation
2. Read subsequent milestones to see how complexity is added incrementally
3. Each milestone document includes:
   - Goal and motivation
   - Architecture diagrams
   - Implementation requirements
   - Success criteria
   - Explicit out-of-scope items
