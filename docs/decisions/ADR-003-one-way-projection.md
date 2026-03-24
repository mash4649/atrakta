# ADR-003 One-Way Projection

Status: Accepted

Projection pipeline is canonical -> overlay -> include -> render.
Projection outputs cannot auto-write back to canonical state.
Reverse synchronization is proposal-only.
