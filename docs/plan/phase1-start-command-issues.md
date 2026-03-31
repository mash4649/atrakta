# Phase 1 Issue Backlog: `atrakta start`

Version: `1.0.0-alpha.2` baseline  
Created: 2026-03-25

This document converts Phase 1 immediate action item #1 into a concrete issue backlog.
It is scoped to defining and implementing a new `atrakta start` command with a stable internal flow.

Primary reference: `docs/plan/phase1-start-command-design.md`.

Status update (2026-03-26):

- `P1-START-001` to `P1-START-007` are implemented in current v0 runtime.
- `P1-START-008` remains as doc consistency follow-up (mainly plan doc drift cleanup).

---

## Issue P1-START-001: Add `atrakta start` CLI surface (parity with `run` flags)

### Summary

Add a new top-level command `atrakta start` with the same core flags as `atrakta run`, positioned as the harness entrypoint for humans/IDE autostart.

### Tasks

- Add `start` to `cmd/atrakta/main.go` command switch.
- Define flagset for `start`:
  - `--project-root`, `--interface`, `--artifact-dir`, `--json`, `--non-interactive`, `--apply`, `--approve`
- Ensure exit code contract matches `run` (0/1/2/3).
- Update `usage()` to list `start` under “primary”.

### Acceptance criteria

- `atrakta start --json` emits a schema-valid output envelope (same schema as `run` or an allowed superset).
- `atrakta start` routes onboarding vs normal path using canonical state detection.
- Exit codes match run contract:
  - `2` for deterministic missing input (e.g., interface unresolved in normal path)
  - `3` for approval-required gating (no writes before approval)

### Dependencies

- None (can be implemented as wrapper delegating to existing run logic).

---

## Issue P1-START-002: Factor shared “entry flow” so `run` and `start` cannot diverge

### Summary

Refactor current `runCommand(...)` implementation into an internal entry function so both `run` and `start` share identical routing and gating logic.

### Tasks

- Introduce a new internal package (name TBD, e.g. `internal/entry` or `internal/commands/start`) that exposes:
  - an input struct (flags + env-derived settings)
  - an `Execute(...)` function returning `(exitCode int, response any, err error)`
- Move onboarding/normal routing + approval gating + audit append into that package.
- Have `run` and `start` call the same internal function.

### Acceptance criteria

- There is exactly one implementation of:
  - canonical state routing
  - interface resolution behavior
  - approval gating behavior
  - audit append behavior
- `atrakta run` behavior is unchanged (snapshot tests, fixtures, and output schema remain valid).

### Dependencies

- P1-START-001 (or done in the same PR).

---

## Issue P1-START-003: Machine contract loading and enforcement for `start`

### Summary

Align implementation with the stated contract that `.atrakta/contract.json` is the machine-executable contract for `run/start`.

### Tasks

- Define a minimal loader in `internal/run` (or `internal/contract`) to:
  - read `.atrakta/contract.json`
  - validate against a schema (existing or new schema under `schemas/operations/`)
- Decide enforcement points:
  - onboarding path: contract may not exist (allowed)
  - normal path: contract MUST exist (or produce a deterministic failure)
- Thread contract-derived data into pipeline input construction, replacing ad-hoc asset-only decisions where appropriate.

### Acceptance criteria

- In canonical-present path, `start` loads `.atrakta/contract.json` and fails deterministically if missing/corrupt.
- The decision is reflected in machine output (reason/evidence) and in audit event payload.

### Dependencies

- P1-START-002 (recommended) to avoid implementing twice.

---

## Issue P1-START-004: Preflight audit integrity verify on `start`

### Summary

Before executing resolver pipeline or performing apply, `start` should verify audit integrity (at the configured level) to catch tampering early.

### Tasks

- Add a preflight step in start flow:
  - locate `.atrakta/audit`
  - call `internal/audit.VerifyIntegrity(...)` at least at A2
- Define behavior when audit store is missing:
  - onboarding path: missing audit store is acceptable
  - normal path: missing audit store is either acceptable (bootstrap) or a warning; choose and encode in output

### Acceptance criteria

- If audit log exists and is invalid, `start` exits `1` with clear diagnostic.
- If audit log is valid, `start` proceeds and appends its operational event as usual.

### Dependencies

- P1-START-002 (optional).

---

## Issue P1-START-005: Define and implement `start` Fast Path snapshot (no-op short circuit)

### Summary

Add a deterministic fast-path that skips full detect/plan/apply when nothing relevant changed.

### Tasks

- Define snapshot key inputs (Phase 1 minimal):
  - contract hash (from `.atrakta/contract.json`)
  - interface id
  - canonical policy index hash
  - workspace stamp (selected files or detected assets list hash)
- Define storage location and schema:
  - e.g. `.atrakta/runtime/start-fast.v1.json` (new)
- On successful “start completed” paths, write snapshot.
- On next `start`, recompute key and short-circuit if identical:
  - still verify audit integrity
  - still emit a `run_execute`-like audit event with `fast_path=true`

### Acceptance criteria

- Re-running `atrakta start` twice in an unchanged workspace produces:
  - second run returns `0` with a response indicating fast-path used
  - no generated/canonical/state writes on the fast path
  - audit append still occurs

### Dependencies

- P1-START-003 (contract hash required)
- P1-START-002 (recommended)

---

## Issue P1-START-006: Add runtime auto-state to improve interface resolution (for future `resume`)

### Summary

Write a small runtime “last known good interface” record on successful start, and use it in interface resolution precedence (explicit/trigger/auto-state/detect).

### Tasks

- Define schema and location:
  - e.g. `.atrakta/runtime/auto-state.v1.json`
- On successful start paths, write:
  - last interface id
  - interface source
  - timestamp (optional; avoid non-determinism in snapshot gates unless explicitly excluded)
- Update interface resolution logic to consult auto-state before detect (normal path).

### Acceptance criteria

- If `--interface` is omitted and no trigger env is set:
  - normal path can resolve interface deterministically using auto-state
  - if auto-state is missing, fall back to detect
- Behavior is reflected in output `interface.source`.

### Dependencies

- P1-START-002

---

## Issue P1-START-007: Start output schema decision (reuse `run` schema vs introduce `start` schema)

### Summary

Lock down how `start` machine output is validated.

### Tasks

- Decide one:
  - (A) `start` output reuses `schemas/operations/run-output.schema.json` exactly
  - (B) introduce `schemas/operations/start-output.schema.json` as a strict superset and update adapter docs accordingly
- Update validation hook(s) in `internal/validation` and CLI.

### Acceptance criteria

- `atrakta start --json` is schema-validated in the same way as `run`.
- Documentation points adapters to the correct schema(s).

### Dependencies

- P1-START-001

---

## Issue P1-START-008: Documentation updates (contract + gaps alignment)

### Summary

Update docs to avoid conceptual drift: `run-contract`, adapter invocation, and harness gap analysis should all align on start/run roles.

### Tasks

- Add a short note to `docs/architecture/run-contract.md`:
  - `run` is the execution primitive and stable adapter target
  - `start` is the harness entrypoint alias built on `run`
- Add a short note to `docs/architecture/adapter-invocation.md`:
  - adapters MUST continue invoking `run` (not `start`)
- Add a short note to `docs/plan/harness-successor-gaps.md`:
  - reflect that `start` is defined and implemented, and keep remaining gaps explicit

### Acceptance criteria

- The three documents do not contradict each other about which command adapters invoke and which command humans should use.

### Dependencies

- None (docs-only), but should land alongside P1-START-001.
