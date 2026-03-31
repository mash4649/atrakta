# Phase 1 Design: `atrakta start` Internal Flow

Version: `1.0.0-alpha.2` baseline  
Created: 2026-03-25

---

Status update (2026-03-26):

- This design has been implemented in the current v0 runtime for `start`/`resume` shared flow.
- The "open questions" section remains as historical design context; current status should be checked in `docs/plan/implementation-status.md`.

## Purpose

Phase 1 immediate action item #1: define the internal flow for a new `atrakta start` command and issue-ize the work.

This document defines:

- the intended role of `atrakta start` relative to the existing `atrakta run` contract
- the concrete internal execution flow and responsibilities
- the minimum artifacts and state transitions `start` must produce (Phase 1 scope)
- a mapping to existing packages (`internal/run`, `internal/pipeline`, `internal/persist`, `internal/audit`, `internal/onboarding`, `internal/adapters`)

---

## Background (current state)

Current v0 contract positioning:

- The operational execution primitive is `atrakta run`. See `docs/architecture/run-contract.md`.
- Adapters invoke `atrakta run` and handle exit codes (0/1/2/3). See `docs/architecture/adapter-invocation.md`.

Current code reality:

- `cmd/atrakta/main.go` implements `atrakta run` with:
  - project root detection (`internal/onboarding`)
  - canonical state detection (`internal/run`)
  - interface resolution (`internal/run`)
  - onboarding route with approval gate + persistence (`internal/persist`)
  - normal route that executes ordered pipeline (`internal/pipeline`) and optionally performs managed apply
  - audit append + integrity verify (`internal/audit`)
  - a minimal run state file (`.atrakta/state/run-state.json`)

Gap analysis positioning:

- As a harness successor, v0 lacks explicit entry points `init / start / resume` as unified user-level runtime commands. See `docs/plan/harness-successor-gaps.md`.

This Phase 1 design focuses on **defining `start`** (not implementing `init`/`resume`).

---

## Goals (Phase 1)

- Provide a stable harness-level entry command: `atrakta start`.
- Keep `atrakta run` as the adapter-facing primitive (no adapter contract break).
- Define a deterministic internal flow that:
  - routes onboarding vs normal execution based on canonical state
  - enforces explicit approval before any canonical/state write
  - produces inspectable machine output consistent with run-contract exit codes
  - appends an audit event for operational trace
- Be implementable with minimal new concepts by reusing existing v0 packages.

---

## Non-goals (Phase 1)

- Implementing `init` / `resume`.
- Implementing wrapper/hook/autostart tooling (`wrap`, `hook`, `ide-autostart`).
- `run-events` only; no `events.jsonl` compatibility or migration tooling in Phase 1.
- Projection rendering/materialization and drift repair (kept out of Phase 1, but issues are listed separately if needed).

---

## Command Surface (Phase 1)

### CLI shape

`atrakta start` MUST accept the same core inputs as `atrakta run`:

- `--project-root <dir>`
- `--interface <id>`
- `--non-interactive`
- `--json`
- `--apply`
- `--approve`

Phase 1 intent:

- `start` is the recommended harness entrypoint for humans and IDE autostart.
- `run` remains the stable adapter invocation target.

### Output + exit codes

`start` MUST preserve the run-contract exit codes:

- `0`: success
- `1`: runtime/contract error
- `2`: NEEDS_INPUT (e.g., unresolved interface in normal path)
- `3`: NEEDS_APPROVAL (explicit approval required before write path)

`start --json` MUST emit the same run-output envelope shape as `run` (or a strict superset) so adapters and external tooling can parse it.

---

## Internal Flow (Phase 1)

This section is the concrete internal algorithm for `atrakta start`.

### 0. Normalize inputs

Inputs are resolved with the same precedence as `run-contract`:

- **interface**: `--interface` > `ATRAKTA_TRIGGER_INTERFACE` > detect > `NEEDS_INPUT`
- **non-interactive**: `--non-interactive` > `ATRAKTA_NONINTERACTIVE`

Implementation note:

- Use `internal/run.ResolveInterface(...)` and `internal/onboarding.DetectAssets(...)` for detect fallback.

### 1. Detect project root

- Call `internal/onboarding.DetectProjectRoot(--project-root)` to find `project_root`.
- Record the resolved `project_root` in the response payload for traceability.

### 2. Detect canonical presence state

- Call `internal/run.DetectCanonicalState(project_root)` and branch:
  - `none` → onboarding route
  - `canonical_present` / `onboarding_complete` → normal route
  - `partial_state` / `corrupt_state` → **hard error** (exit `1`)

Rationale:

- `run-contract` explicitly forbids silent repair of canonical state.

### 3. Resolve runtime interface

Resolve interface in this order:

1. explicit flag `--interface`
2. trigger env `ATRAKTA_TRIGGER_INTERFACE`
3. detect from assets (cursor/vscode/mcp/generic-cli/github-actions)
4. if still unresolved:
   - onboarding route: fallback to `generic-cli` (Phase 1 pragmatic default)
   - normal route: `NEEDS_INPUT` (exit `2`), include:
     - `required_inputs: ["interface"]`
     - `next_allowed_action` that points to `--interface` / env var

### 4A. Onboarding route (canonical state: `none`)

1. Build onboarding proposal bundle:
   - `internal/onboarding.BuildOnboardingProposal(project_root)`
   - validate schema via existing validation hook (implementation details in `cmd/atrakta/main.go`)
2. Evaluate surface portability (proposal-only degrade policy):
   - use `resolve_surface_portability` with binding capabilities based on `--interface` resolution
3. Approval gate:
   - If approval is not present and interactive prompt is disallowed:
     - return `NEEDS_APPROVAL` (exit `3`)
     - set `approval_scope: "onboarding_accept"`
     - **MUST NOT** write `.atrakta/*` (canonical/state)
   - Otherwise prompt (interactive) or accept `--approve`
4. If approved:
   - persist using `internal/persist.AcceptOnboarding(project_root, proposal_bundle)`
   - append audit event `accept_onboarding` is already emitted by `AcceptOnboarding`
5. Emit final response.

### 4B. Normal route (canonical present)

1. Load canonical summary (minimal, used for run output and apply plans)
2. Execute ordered resolver pipeline in **inspect** mode:
   - build input using detected assets + interface + canonicalPresent true
   - run `internal/pipeline.ExecuteOrdered("inspect", input)`
3. Extract portability decision:
   - If portability is not `supported`: treat as proposal-only and **disable apply** even if requested
4. Build managed apply plans for generated targets:
   - `.atrakta/generated/repo-map.generated.json`
   - `.atrakta/generated/capabilities.generated.json`
   - `.atrakta/generated/guidance.generated.json`
5. If `--apply` is not requested:
   - append audit event `run_execute` with `apply_requested=false`
   - exit `0`
6. If `--apply` requested:
   - approval gate:
     - if not approved and non-interactive, return `NEEDS_APPROVAL` (exit `3`) with `approval_scope: "managed_apply"`
     - else prompt or accept `--approve`
   - apply each plan through managed mutation path (`internal/mutation.Apply` currently used in `cmd/atrakta/main.go`)
   - write `.atrakta/state/run-state.json` with applied metadata
   - append audit event `run_execute` with `apply_performed=true`
   - exit `0`

### 5. Response envelope invariants

`start` output MUST include (at minimum):

- `status` (`ok`, `needs_input`, `needs_approval`, or `error`-like, aligned with existing run output schema)
- `path` (`onboarding` or `normal`)
- `project_root`
- `canonical_state`
- `interface` resolution (id + source)
- portability fields (`portability_status`, `resolved_projection_targets`, etc.)
- `next_allowed_action` and `required_inputs` when in `NEEDS_INPUT` / `NEEDS_APPROVAL`

---

## Relationship to `atrakta run`

Phase 1 policy:

- `atrakta run` remains the adapter contract surface.
- `atrakta start` is introduced as a **harness-level alias** that executes the same internal behavior as `run`, but is positioned as the human/IDE entrypoint.

Implementation shaping (Phase 1):

- Prefer factoring shared logic into `internal/start` (or `internal/commands/start`) and reuse it from both `run` and `start` to avoid divergence.
- Alternatively, implement `start` as a thin wrapper that calls the same internal entry function currently used by `runCommand(...)` (refactor required).

---

## Required artifacts and stores (Phase 1)

Existing artifacts that `start` may create/update (only behind approval gates):

- `.atrakta/canonical/**` (onboarding accept)
- `.atrakta/generated/**` (onboarding accept and managed apply)
- `.atrakta/state/**` (onboarding accept and apply success)
- `.atrakta/audit/**` (append-only; verify integrity at A2 or higher)

Phase 1 explicitly does NOT introduce:

- events.jsonl mapping
- full DAG-level task planning semantics beyond minimal `task-graph.json` persistence

Those remain deferred and tracked as follow-up issues.

---

## Observability and inspectability

`start` MUST remain inspectable:

- resolver pipeline output remains available via `inspect_bundle` (or equivalent field) in JSON output (as `run` currently does)
- audit appends an operational event (`run_execute`) for replay/traceability

See `docs/architecture/inspectability-contract.md` and `docs/architecture/audit-integrity.md` for baseline expectations.

---

## Phase 1 open questions (to resolve via issues)

1. **Machine contract loading**: `run-contract` says `.atrakta/contract.json` is the machine-executable contract. Current `run` path uses detected assets + canonical summary but does not load/validate `.atrakta/contract.json`. Decide how `start` should load and enforce it.
2. **Fast Path snapshot**: define the minimal snapshot key (contract hash + workspace stamp + interface + feature id) and its storage location.
3. **Interface resolution precedence**: whether to add `auto-state` (last successful interface) before detect, as described in gaps doc.
4. **Event stream format**: `run-events` is the canonical runtime stream; define which event types Phase 1 must emit.
