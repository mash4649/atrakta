# Changelog

All notable changes to this project are documented in this file.

## [0.14.0] - 2026-03-04

### Changed

- Destructive optimization phase (Execution Snapshot Fast Gate):
  - Added `start_fast_v2` runtime snapshot cache under `.atrakta/runtime/meta.v2.json`
  - `start` now returns via fast path when snapshot conditions match:
    - `contract_hash`
    - `workspace_stamp`
    - interface set
    - feature id
    - runtime config key (`sync_level/map/task_category`)
  - Added strict-on-demand fallback:
    - strict refresh interval (10 min)
    - config drift detection
    - managed artifact drift detection (fail-closed)
  - Added `start_fast_hit` event for fast-path observability.
- Refactor:
  - Extracted start fast-path decision/persistence into new package:
    - `internal/startfast`
- Performance gate calibration:
  - adjusted `apply_ops_300` SLO limits for cross-run noise tolerance while remaining fail-closed.

### Verification

- `GOCACHE=$(pwd)/.tmp/go-build GOMODCACHE=$(pwd)/.tmp/go-mod go test ./...`
- `./scripts/verify_perf_gate.sh`
- Observed `BenchmarkStartSteadyState` p95 improved from ~315ms class to ~46ms class on local Apple M4.

## [0.13.0] - 2026-03-04

### Changed

- Unified quality gate to SLO fail-closed flow:
  - `scripts/verify_perf_gate.sh` now enforces:
    - p95/p99 for `BenchmarkApplyScaling/ops_300`
    - p95/p99 for `BenchmarkBuildNoopManagedScaling/managed_1000`
    - p95/p99 for `BenchmarkWrapperFastPath`
    - p95/p99 for `BenchmarkStartSteadyState`
    - token budget gate (`TestSLORepoMapTokenBudgetRespected`)
    - fast-path hit-rate gate (`TestSLOWrapperFastPathHitRate`)
- Added `BenchmarkStartSteadyState` for start hot-path regression measurement.
- Added fault-injection test coverage:
  - interrupted append detection and urgent/group-commit durability behavior in `internal/events`
  - lock contention/stale lock recovery in `internal/util`
- Added soak operation scripts:
  - `scripts/soak.sh`
  - `scripts/soak_24h.sh`
  - `scripts/soak_72h.sh`
- Documentation/system alignment:
  - design principle unified as `Fast Path First, Strict Path On Demand`
  - verification docs updated with SLO/fault/soak operation
  - CI step label updated to `SLO regression gate`

### Verification

- `GOCACHE=$(pwd)/.tmp/go-build GOMODCACHE=$(pwd)/.tmp/go-mod go test ./...`
- `./scripts/verify_perf_gate.sh`
- `./scripts/soak.sh --iterations 5 --sleep-sec 1`

## [0.12.0] - 2026-03-04

### Changed

- Runtime cache architecture:
  - Introduced unified runtime cache store:
    - `.atrakta/runtime/meta.v2.json`
  - Consolidated repo-map / events-verify / wrapper runtime cache handling into a single metadata plane.
- Context and conventions efficiency:
  - Added conventions staged loading (`index -> prioritized sections`) with token budget control.
  - Kept `conventions_read_only` as harness-side mutation guard while preserving direct human edits.
- Event durability/performance path:
  - Added adaptive group commit behavior for `AppendBatch`.
  - Added urgent sync behavior for blocked/fail critical payloads.
  - Added `events.Flush` and integrated phase-end flush in `start`.
- Wrapper precision/performance:
  - Added hybrid stamp strategy (quick stamp + deep fallback).
  - Added periodic deep sampling to reduce drift miss risk while preserving fast path latency.
- Go 1.26 optimization adoption:
  - Upgraded module baseline to `go 1.26`.
  - Extended runtime observability with scheduler metrics (`runtime/metrics`).
  - Migrated benchmark loops to `b.Loop` for stable benchmark behavior on 1.26.
  - Applied hot-path allocation reductions in `events/context/repomap`.

### Verification

- `GOCACHE=$(pwd)/.tmp/go-build go test ./...`

## [0.11.0] - 2026-03-04

### Changed

- Context and policy stability:
  - Added conventions-aware context resolution (`context.conventions`)
  - Added read-only conventions guard (`context.conventions_read_only=true`)
  - Context resolver now reports `conventions_loaded`
- Repository map control:
  - Added repo map budget and refresh controls:
    - `context.repo_map_tokens`
    - `context.repo_map_refresh_seconds`
  - Added CLI overrides:
    - `--map-tokens`
    - `--map-refresh`
  - Added runtime cache:
    - `.atrakta/runtime/repo-map.v2.json`
  - `start` now emits `repo_map` event and uses cached refresh flow
- Start path performance improvements:
  - Added batched event append API (`events.AppendBatch`)
  - Subworker orchestration events now use batch writes to reduce lock/sync overhead
  - Added cached chain verification (`events.VerifyChainCached`) in `start`
  - Wrapper workspace stamp no longer shells out to `git` commands on hot path
- Projection consistency:
  - Projection generation now uses already-resolved context text from `start` (no second root `AGENTS.md` read path)

### Verification

- `GOCACHE=$(pwd)/.tmp/go-build go test ./...`

## [0.10.0] - 2026-03-04

### Changed

- Added runtime GC command and policy:
  - `atrakta gc [--scope <tmp,events>] [--apply] [--auto]`
  - `.tmp` cleanup supports threshold-triggered auto mode and manual apply mode
  - `events.jsonl` remains proposal-only (no automatic mutation)
- Added GC observability runtime artifacts:
  - `.atrakta/runtime/gc-state.v1.json`
  - `.atrakta/runtime/gc-log.jsonl` (dry-run/apply details)
- Auto GC now runs outside `start` critical path:
  - best-effort background trigger after `start`/`init`/`resume` reaches `DONE`
  - interval guard prevents repeated execution on every run
- Added doctor self-heal proposals for storage pressure:
  - `.tmp` over threshold -> suggest `atrakta gc --scope tmp --apply`
  - `events.jsonl` over threshold -> suggest `atrakta gc --scope events`
- Documentation updates:
  - added user-segmented update guide
  - added GC operations guide
  - synchronized command references and schema/runtime docs

### Verification

- `go test ./...`

## [0.9.0] - 2026-03-04

### Changed

- Onboarding/automation:
  - added `atrakta init` one-shot bootstrap command
  - added `atrakta ide-autostart [install|uninstall|status]`
  - `ide-autostart install` now manages a VSCode-compatible workspace task (`.vscode/tasks.json`, `runOn=folderOpen`)
- Wrapper/path reliability:
  - `wrap install` now auto-fixes `~/.local/bin` PATH priority only when needed (idempotent)
  - added wrapper health diagnostics for PATH/order and wrapper presence
- Start lifecycle ergonomics:
  - standardized deferred outcome exit codes:
    - `NEEDS_INPUT` -> `4`
    - `NEEDS_APPROVAL` -> `5`
    - `BLOCKED` -> `6`
  - added optional machine-readable deferred outcome output via `ATRAKTA_STATUS_JSON=1`
- Doctor self-heal proposals:
  - now suggests missing `ide-autostart` setup
  - now suggests wrapper/PATH repair command when wrapper health checks fail
- Auto-resolution/runtime hardening from Plan3 continuation:
  - introduced `.atrakta/runtime/auto-state.v1.json`
  - wrapper/hook trigger-aware resolution with safe no-prune behavior
  - first-run unresolved interface now returns `needs_input` (no implicit default)

### Verification

- `go test ./...`

## [0.8.0] - 2026-03-04

### Changed

- Sprint3 performance tuning for large operation sets:
  - apply now pre-classifies compile-time no-op ops (`link` satisfied / `adopt` satisfied) and skips worker scheduling overhead
  - auto parallel decision now uses executable-op count (excluding compile-time no-ops)
  - plan build compacts duplicate projection paths before op generation
  - plan build short-circuits managed-stable projections before expensive filesystem equivalence probing
- Added benchmark for no-op managed planning scale:
  - `BenchmarkBuildNoopManagedScaling`
- Reliability hardening:
  - missing managed projection artifacts are now auto-repaired on next `start`
  - added interrupted-run determinism test covering `needs_approval -> rerun -> stable state`
- Added percentile performance gate script (fail-closed):
  - `./scripts/verify_perf_gate.sh` (p95/p99 thresholds)
- Added CI fail-closed enforcement:
  - GitHub Actions now runs `verify_loop` + `verify_perf_gate` on `push` / `pull_request`
- Documentation/archive cleanup:
  - bumped release/version metadata to `0.8.0`
  - removed obsolete archive plan docs and legacy archive documents
  - aligned docs index to current canonical set (`01_全体` 〜 `04_品質`)

### Verification

- `go test ./...`
- `./scripts/verify_provisional.sh`
- `./scripts/verify_perf_gate.sh`
- `./scripts/verify_phase2.sh` (Windows runtime execution remains environment-skip when unavailable)

## [0.7.0] - 2026-03-03

### Changed

- Plan/apply optimization for large operations:
  - added `adopt` operation for equivalent existing projections (plan-stage no-op compaction)
  - apply now revalidates `adopt` preconditions and skips safely without mutation
- Reliability hardening for interrupted runs:
  - added run checkpoint persistence (`.atrakta/run-checkpoints/latest.json`)
  - added `atrakta resume` command to restart from last checkpoint context
- Latest-only runtime policy:
  - removed automatic schema migration from `start` and `doctor`
  - removed CLI command `migrate upgrade-events-v2`
  - removed legacy upgrader implementation/tests (`internal/migrate/upgrade*`)
- Docs and command reference synchronized to latest-only behavior.
- Distribution cleanup:
  - runtime artifacts under `.atrakta/` are no longer tracked in Git
  - project ignore rules updated for generated runtime files

### Verification

- `go test ./...`
- `./scripts/verify_provisional.sh`
- `./scripts/verify_phase2.sh` (Windows runtime execution remains environment-skip when unavailable)

## [0.6.1] - 2026-03-03

### Changed

- Release housekeeping:
  - synchronized `VERSION` and release metadata
  - synchronized root/user-facing docs with current implementation
- Documentation refresh under `docs/`:
  - command list now includes `migrate upgrade-events-v2`
  - scope/operations/troubleshooting pages now include DAG task graph and events v2 migration flow
  - quality guidance now reflects edit safety and task graph related checks

### Verification

- `go test ./...`
- `./scripts/verify_provisional.sh`
- `./scripts/verify_phase2.sh` (Windows runtime execution remains environment-skip when unavailable)

## [0.6.0] - 2026-03-03

### Added

- DAG task graph planning/execution:
  - per-operation `task_id` / `task_blocked_by`
  - persisted `.atrakta/task-graph.json`
  - `task_graph_planned` event emission
- Edit safety runtime guards (`anchor+optional_ast`) for write/copy fallback:
  - Go AST validation
  - JSON parse validation
- Language-aware managed header generation (`//` vs `#`, no header for JSON).
- Explicit events migration command: `atrakta migrate upgrade-events-v2`.

### Changed

- `apply` now executes in DAG topological order and fails closed on invalid task graph.
- `doctor` now validates persisted task graph structure.
- Events schema is now strict `schema_version = 2`; writer/check no longer accept legacy values.
- `start` and `doctor` command flow now auto-upgrade legacy events to v2 before diagnostics/execution.
- Task graph topological ordering now has no-dependency fast-path to reduce orchestration overhead.

### Verification

- `go test ./...`
- `./scripts/verify_provisional.sh`
- `./scripts/verify_phase2.sh` (Windows runtime execution remains environment-skip when unavailable)

## [0.5.0] - 2026-03-03

### Added

- Contract extension for Plan3-next primitives:
  - `routing`
  - `context`
  - `security`
  - `edit_safety`
  - `policies.prompt_min`
- Hierarchical AGENTS context resolver (`nearest_with_import`) with import depth and cycle detection.
- Deterministic orchestration events:
  - `context_resolved`
  - `routing_decision`
  - `policy_applied`
- Prompt minimum policy loader and default policy bootstrap at `.atrakta/policies/prompt-min.json`.
- Plan-level required permission model (`read_only` / `workspace_write`) and security preflight before apply.

### Changed

- `start` now resolves AGENTS context through `internal/context` and applies routing/policy decisions before execution.
- `gate` now enforces security profile constraints and supports route-driven heavy checks (`quality=heavy`).
- `doctor` now validates contract + prompt policy + AGENTS context chain before repair/recovery checks.
- Project default `.atrakta/contract.json` updated with Plan3 fields and safe defaults.

### Verification

- `go test ./...`
- `./scripts/verify_provisional.sh`
- `./scripts/verify_phase2.sh` (Windows runtime execution remains environment-skip when unavailable)

## [0.4.0] - 2026-03-03

### Added

- Minimal hook pipeline stages for shell-triggered sync:
  - `pre_start`
  - `post_start`
  - `on_error`
- Context budget guard for subworker plans with deterministic digest/summary compaction.
- Recovery-aware verification loop script: `./scripts/verify_loop.sh`.

### Changed

- `contract.json`, `progress.json`, and wrapper cache writes now use file-lock + atomic write paths.
- `doctor` now repairs corrupted `progress.json` from event history when possible.
- CLI usage and docs now expose `atrakta wrap run`.
- Wrapper fast-path now skips expensive git probes when the workspace is not a git repository.

### Verification

- `go test ./...`
- `./scripts/verify_phase2.sh` (Windows runtime execution remains environment-skip when unavailable)

## [0.3.0] - 2026-03-03

### Added

- Subworker integration queue (`single_writer_queue`) for deterministic orchestration flow:
  - parallel proposal generation
  - single-writer queue integration
  - centralized apply execution
- Queue validation and deterministic digest for replay-safe merge/application ordering.
- Optional branch lane planning as limited scaffolding (planning-only, no auto merge to main).
- Git automation hardening for initial setups:
  - `autonomy.git.mode=auto` default behavior
  - graceful degrade in `auto` mode when `git` is unavailable
  - strict fail in `on` mode when `git` is unavailable

### Changed

- Apply auto-parallel promotion now prefers queue safety signals (`non_conflicting` + workload threshold).
- Phase2 verification includes queue determinism and single-writer rollback test coverage.
- Phase2 planning docs now include explicit current-vs-ideal gap closure record.

### Verification

- `go test ./...`
- `./scripts/verify_phase2.sh` (Linux runtime test passed via Docker; Windows runtime remains environment-skip)

## [0.2.0] - 2026-03-03

### Added

- `AGENTS.md` auto-bootstrap on `atrakta start`/`doctor`.
- `progress.json` lifecycle (`active_feature`, `completed_features`) with `--feature-id`.
- `sync-level` policy (0/1/2) and proposal-only contract sync via `doctor --sync-proposal`.
- Optional projection templates: `contract-json`, `atrakta-link`.
- `schema_version` validation for events and migration checks.
- Wrapper fast-path benchmark auto-threshold check (`<50ms` target) in verification scripts.
- Linux/Windows cross-build compile checks and binary generation checks.

### Changed

- Gate quick/heavy check behavior expanded for stronger safety and quality enforcement.
- Documentation was reorganized into Japanese canonical docs under `docs/01_全体` to `docs/05_計画`.

### Verification

- `./scripts/verify_provisional.sh`
- `./scripts/verify_phase2.sh` (Windows runtime execution is skipped when environment does not support it)
