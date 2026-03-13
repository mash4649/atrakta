# CLI Specification

[English](./01_cli_spec.md) | [日本語](../../ja/02_仕様/01_CLI仕様.md)

## Spec Version

- Document version: `v0.14.1`
- Target CLI: `atrakta` `v0.14.1`
- Last updated: `2026-03-04`

## Commands

```bash
atrakta init [--mode <greenfield|brownfield>] [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>] [--merge-strategy <append|include|replace>] [--agents-mode <append|include|generate>] [--no-overwrite] [--no-hook]
atrakta start [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]
atrakta doctor [--sync-proposal] [--apply-sync] [--sync-level <0|1|2>] [--parity|--integration] [--json]
atrakta gc [--scope <tmp,events>] [--apply] [--auto]
atrakta wrap install
atrakta wrap uninstall
atrakta wrap run --interface <id> --real <path> -- [args...]
atrakta hook install [--surface <surface_id,...>]
atrakta hook uninstall [--surface <surface_id,...>]
atrakta hook status [--surface <surface_id,...>] [--json]
atrakta hook repair [--surface <surface_id,...>]
atrakta ide-autostart [install|uninstall|status]
atrakta migrate check
atrakta resume [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]
atrakta projection render [--interface <id>] [--all]
atrakta projection status [--json]
atrakta projection repair [--interface <id>] [--all]
atrakta import repo <path> [--auto-analyze]
atrakta import report <batch_id>
atrakta import pulse
atrakta capability analyze <capability_id>
atrakta recipe convert <capability_id> --deterministic-input-note <note> [--status <pending|approved|rejected>] [--input-contract-ref <ref>] [--allow <primitive,...>]
atrakta memory review <capability_id> [--status <pending|approved|rejected>] [--promote] [--operator <id>]
atrakta exploration catalog [--reviewed-only] [--limit <n>]
```

## Compatibility Policy

- latest-only operation (no backward compatibility)
- old data/old CLI assumptions are fail-closed unless explicitly migrated
- `migrate check` requires `events.schema_version = 2`

## `init`

- Integrated first-time setup command
- Key flags:
  - `--mode`: `greenfield` (new project) or `brownfield` (existing project integration)
  - `--merge-strategy`: merge policy for AGENTS etc. (`append` / `include` / `replace`)
  - `--agents-mode`: AGENTS generation policy (`append` / `include` / `generate`)
  - `--no-overwrite`: do not overwrite existing files
- Execution order:
  1. `wrap install`
  2. `hook install` (skipped with `--no-hook`)
  3. `ide-autostart install` (workspace `.vscode/tasks.json`)
  4. `start`
- Returns same deferred outcomes as `start` (`NEEDS_INPUT`, `NEEDS_APPROVAL`, `BLOCKED`)

## `start`

- Load/init contract
- Verify events chain (`VerifyChainCached`; full verify on cache mismatch)
- Skip strict pipeline when Start Snapshot Fast Gate matches (`runtime/meta.v2.start_fast_v2`)
- Auto-escalate to strict path on strict interval (10 min), config diff, or managed artifact drift
- Event writes are batch/group-commit optimized (critical payloads synced immediately)
- Resolve context (`nearest_with_import`) and routing
- Build/reuse repo map with budget/refresh controls and record `repo_map` event
- Apply prompt policy conditionally
- Initialize `AGENTS.md` and `progress.json`
- detect -> plan (task DAG) -> security preflight -> apply (topological) -> gate
- Update state/progress on success
- Record runtime metrics to `.atrakta/metrics/runtime.json`

### Key Flags

- `--interfaces`: explicit target interfaces
- `--feature-id`: long-running task id
- `--sync-level`: sync policy level
- `--map-tokens`: temporary override for repo map token budget
- `--map-refresh`: temporary override for repo map refresh interval (sec)

### Environment Variables

- `ATRAKTA_INTERFACES`: alternative to `--interfaces`
- `ATRAKTA_TRIGGER_INTERFACE`: wrapper/hook interface hint
- `ATRAKTA_TRIGGER_SOURCE`: launch source (`wrapper`/`hook`)
- `ATRAKTA_NONINTERACTIVE`: disable input prompts when `1`
- `ATRAKTA_STALE_INTERFACE_DAYS`: stale proposal threshold (default 30)
- `ATRAKTA_GC_DISABLE`: disable auto GC when `1`
- `ATRAKTA_GC_TMP_MAX_BYTES`: `.tmp` auto GC threshold (default 2GiB)
- `ATRAKTA_GC_TMP_RETENTION_DAYS`: preferred deletion retention days (default 7)
- `ATRAKTA_GC_AUTO_MIN_INTERVAL_MIN`: minimum auto GC interval minutes (default 60)
- `ATRAKTA_TASK_CATEGORY`: routing category override

### Interface Resolution Order

1. Explicit (`--interfaces` / `ATRAKTA_INTERFACES`)
2. Trigger (`ATRAKTA_TRIGGER_INTERFACE`)
3. Last successful single target (`.atrakta/runtime/auto-state.v1.json`)
4. Detect observation (anchors/managed state)

- If unresolved and `reason=unknown`, `start` returns `needs_input` (no implicit default).

## `doctor`

- Integrity diagnostics and recovery suggestions
- `--sync-proposal`: show AGENTS-derived proposal
- `--apply-sync`: apply proposal with explicit approval
- `--parity`: diagnose Parity Contract drift (AGENTS.md recommended check)
- `--integration`: diagnose integration consistency
- `--json`: machine-readable JSON output (same as `ATRAKTA_STATUS_JSON=1`)
- `--parity` and `--integration` are mutually exclusive
- Additional remediation suggestions:
  - missing `ide-autostart` -> `atrakta ide-autostart install`
  - wrapper/PATH mismatch -> `atrakta wrap install`
  - `.tmp` threshold exceeded -> `atrakta gc --scope tmp --apply`
  - large `events.jsonl` -> `atrakta gc --scope events`
- Records runtime metrics to `.atrakta/metrics/runtime.json`

## `gc`

- Operational GC with context preservation
- Policy:
  - `.tmp`: auto/manual candidate only when threshold exceeded
  - `events.jsonl`: proposal-only (no automatic mutation)
- `--scope`: `tmp`, `events` (multi allowed)
- `--apply`: execute deletion for selected scopes
- `--auto`: threshold/interval-driven auto mode
- Every run writes dry-run and applied results to `.atrakta/runtime/gc-log.jsonl`

## `wrap`

- Install/uninstall wrapper into user bin
- Skip `start` when fast-path conditions are met

## `hook`

- Install/uninstall shell hook for directory-change-triggered `start`
- `install` / `uninstall`: add/remove hook (`--surface` to scope by surface)
- `status`: show installation state (`--json` for machine-readable)
- `repair`: fix hook inconsistencies
- Hook-triggered `start` is non-interactive (`ATRAKTA_NONINTERACTIVE=1`)

## `ide-autostart`

- Autostart task management for VSCode-compatible IDEs
- `install`: add managed `runOn=folderOpen` task to `.vscode/tasks.json` (idempotent)
- `uninstall`: remove managed task only
- `status`: show installation state in JSON

## `migrate check`

- Validate `events.jsonl` schema version
- Current requirement: `schema_version = 2`

## `projection`

- Manage tool-specific projection (generation from canonical sources)
- `render`: generate projection for specified interface or `--all`
- `status`: show projection manifest and extension manifest state (`--json` for machine-readable; AGENTS.md recommended check)
- `repair`: fix missing or drifted projections

## `resume`

- Load `.atrakta/run-checkpoints/latest.json` and restart with prior conditions
- `--interfaces` / `--feature-id` / `--sync-level` override checkpoint values when provided

## Import/Review Surfaces

- `import repo`: deterministic local-directory import + quarantine-first registration.
- `import repo --auto-analyze` (default `true`): analyze-only hook after import.
- `import report`: batch report with pending conversion/memory review counts.
- `import pulse`: one-screen operations visibility for pending import/review state.
- `capability analyze`: analyze a single capability and persist analysis metadata.
- `recipe convert`: review-gated conversion to `recipe_candidate` with bounded defaults.
- `memory review`: review-gated memory promotion (no automatic promotion).
- `exploration catalog --reviewed-only`: opt-in retrieval path for review-passed imported assets.

## Exit Codes (Deferred Outcomes)

- `4`: `NEEDS_INPUT`
- `5`: `NEEDS_APPROVAL`
- `6`: `BLOCKED`
- Use `ATRAKTA_STATUS_JSON=1` for machine-readable outcome output
