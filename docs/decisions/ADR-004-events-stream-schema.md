# ADR-004 Events Stream Schema (v0)

Status: Accepted

## Context

v0 needs an append-only event stream for audit/debug/repro (“what happened”) that can be verified for integrity.

This repository currently contains two related but different streams:

- `/.atrakta/events.jsonl`: a 0.14.1-style events stream (`schema_version: 2`, `prev_hash` chain, many event `type`s).
- `/.atrakta/audit/events/install-events.jsonl`: the v0 audit chain used today (see `internal/audit/chain.go`), with integrity levels A0–A3.

The v0 roadmap explicitly positions v0 as a **refresh line**: concept coverage is prioritized over 0.14.1 parity, but migration must be explicit.

## Decision

v0 defines its **own, versioned events stream schema** for Phase 1, and treats 0.14.1’s `events.jsonl` format as **historical foreign data** (not as the primary storage schema).

Canonical ownership:

- The v0 events stream is a **canonical audit artifact** (see ADR-001) and must remain append-only.
- Projections may render summaries derived from events, but projections must not rewrite canonical events (see ADR-003).

## v0 Events Stream: Storage Location

Phase 1 storage is under the audit store:

- `.atrakta/audit/events/run-events.jsonl` (new; runtime/session events)
- `.atrakta/audit/events/install-events.jsonl` (existing; install/onboarding events)

Rationale: keep “audit” as the integrity boundary, and allow retention/GC policy to treat audit heads specially.

## v0 Events Stream: Schema (v0, schema_version = 1)

Each line is one JSON object.

### Required fields

- `schema_version` (number): **1**
- `seq` (number): 1-based monotonically increasing sequence within the file
- `timestamp` (string): RFC3339 / `date-time`
- `event_type` (string): stable identifier (see “Minimal event types”)
- `integrity_level` (string): `"A0" | "A1" | "A2" | "A3"`
- `payload` (object): event-specific data (must be JSON object)

### Integrity fields (required depending on integrity_level)

- `payload_hash` (string): hex SHA-256 of canonical payload bytes (required for A1+)
- `prev_hash` (string): previous event `hash` (required for A2+; empty for `seq=1`)
- `hash` (string): hex SHA-256 chain hash (required for A2+)

### Optional fields (Phase 1)

- `event_id` (string): stable ID for cross-file correlation (recommended; not required for chain integrity)
- `actor` (string): `"orchestrator" | "kernel" | "worker" | ...` (free-form)
- `run_id` (string): identifier for one `start`/session execution
- `interface` (string): e.g. `"cursor"`, `"vscode"`, `"cli"`
- `feature_id` (string): user-meaningful feature/work item identifier when available

### Example (A2)

```json
{"schema_version":1,"seq":7,"timestamp":"2026-03-25T12:34:56Z","event_type":"plan.created","integrity_level":"A2","payload":{"task_count":3,"task_graph_id":"sha256:..."},"payload_hash":"<hex-sha256>","prev_hash":"<hex-sha256>","hash":"<hex-sha256>","run_id":"run-20260325-123456","actor":"kernel"}
```

## Hashing Strategy (canonical, Phase 1)

v0 adopts the same integrity construction already implemented in `internal/audit/chain.go`, applied to the v0 schema:

- **Canonical payload bytes**: JSON serialization of the `payload` object using Go `encoding/json` (UTF-8, no whitespace, stable key ordering as produced by Go’s encoder).
- `payload_hash = sha256(payload_bytes)` (hex)
- `hash = sha256( fmt("%d|%s|%s|%s", seq, event_type, payload_hash, prev_hash) )` (hex)

Notes:

- `timestamp` is **not** included in the chain hash in Phase 1 to keep compatibility with the existing verifier and avoid creating two subtly different integrity schemes.
- For A3, a checkpoint file MAY be written (e.g. `.atrakta/audit/checkpoints/run-head.json`) analogous to the existing head checkpoint pattern.

## Minimal Event Types (Phase 1)

Phase 1 requires only the event types needed to explain a `start`-style run at a high level:

- `onboarding.accepted`
- `init.begin`
- `init.step`
- `init.end`
- `start.begin`
- `detect.performed`
- `plan.created`
- `apply.performed`
- `gate.result`
- `projection.rendered` (or `projection.skipped`)
- `projection.status`
- `projection.repaired`
- `gc.planned`
- `gc.applied`
- `migrate.checked`
- `start.end`
- `error.raised` (for failures that abort the run)

Additional event types from 0.14.1 MAY be introduced later, but are not required for Phase 1.

## Compatibility and Migration Strategy (Explicit)

v0 does **not** promise parity with 0.14.1’s `/.atrakta/events.jsonl` schema. Instead:

- **Read**: If a workspace already has `/.atrakta/events.jsonl`, v0 treats it as historical/foreign data and does not mutate it.
- **Write**: v0 writes only the v0 audit streams under `/.atrakta/audit/events/`.
- **No bidirectional converter layer**: compatibility is maintained by stabilizing v0 event taxonomy and payload conventions.

### Event Mapping Conventions (v0 canonical)

- `start.begin`: must include `path`, `canonical_state`, `interface_id`, `apply_requested`.
- `detect.performed`: must include `step_count`, `final_allowed_action`.
- `plan.created`: must include `planned_target_count`, `planned_target_paths`.
- `gate.result`: must include `status`, `next_allowed_action`; include `approval_scope` when applicable.
- `apply.performed`: must include `applied_count`, `applied_target_paths`.
- `projection.rendered|projection.status|projection.repaired`: must include `target_path`, `drift`, `written`.
- `gc.planned|gc.applied`: must include `scope`, `apply`, `candidate_count`, `removed_count`.
- `migrate.checked`: must include `ok`, `check_count`.

## Consequences

- v0 gains a stable, versioned audit/event contract aligned with existing A0–A3 integrity levels.
- 0.14.1 compatibility is maintained by treating legacy stream as historical data.
- Implementation work remains unblocked: Phase 1+ can extend v0 event taxonomy without introducing conversion-layer coupling.
