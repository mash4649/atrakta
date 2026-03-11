# Data Model

[English](./03_data_model.md) | [日本語](../../ja/02_仕様/03_データモデル.md)

## Schema Versions (latest-only, v0.14.1)

- `contract.json`: `contract.v = 1`
- `state.json`: `state.v = 1`
- `events.jsonl`: `schema_version = 2` (writer/check both fixed to `2`)
- `capabilities/registry.json`: `v = 1`
- `projections/manifest.json`: `v = 1`
- `extensions/manifest.json`: `v = 1`
- `progress.json`: no explicit version field (fixed-key operation)
- `task-graph.json`: `v = 1`
- `runtime/auto-state.v1.json`: `v = 1`
- `runtime/gc-state.v1.json`: `v = 1`
- `runtime/meta.v2.json`: `v = 2`

## `AGENTS.md`

- Human-readable principles and operation guidance
- Optional sync directives:
  - `sync.prefer_interfaces:`
  - `sync.disable_interfaces:`

## `.atrakta/contract.json`

- `v`, `project_id`
- `interfaces.supported/core_set/fallback`
- `boundary.include/exclude/managed_root`
- `tools.approval_required_for`
- `token_budget.soft/hard`
- `hints` (prefer/disable/anchors)
- `quality` (quick/heavy checks)
- `projections.optional_templates/max_per_interface`
- `routing` (worker/quality per task category)
- `context`
  - `resolution`, `projection`, `max_import_depth`
  - `conventions[]` (always loaded)
  - `conventions_read_only` (fixed `true`)
  - `repo_map_tokens`, `repo_map_refresh_seconds`
- `security.profile` (`read_only|workspace_write|full`)
- `edit_safety` (anchor normalization rules + `off|ast|parse` per language)
- `policies.prompt_min` (conditional Goal prefix policy reference)
- `parity`
  - `v`, `canonical_sources`
  - `instruction/approval/output/execution/quality/safety/routing/projection surface`
- `extensions`
  - `v`, `merge_mode`
  - `agents`, `mcp`, `plugins`, `skills`, `workflows`, `hooks`

## `.atrakta/state.json`

- Current values for managed paths and fingerprints
- Updated from apply results
- `projection` (last rendered state/hash/status)
- `integration` (last checked result and blocking reasons)

## `.atrakta/events.jsonl`

- Append-only
- Each event stores `schema_version` and `hash`
- Chain-connected by `prev_hash`
- Plan3 extension events:
  - `context_resolved`
  - `repo_map`
  - `routing_decision`
  - `policy_applied`
- parity/integration events:
  - `projection_rendered`
  - `projection_drift_detected`
  - `integration_checked`
  - `integration_blocked`
- imported capability lifecycle events:
  - `capability_imported`
  - `capability_analyzed`
  - `capability_quarantined`
  - `capability_promoted`
  - `recipe_candidate_created`
  - `recipe_conversion_reviewed`
  - `memory_surface_assigned`
  - `memory_promotion_reviewed`

## `.atrakta/capabilities/registry.json`

- `v` (current: `1`)
- `entries[]`
  - required:
    - `id`
    - `kind` (`skill|recipe_candidate|reference_memory|gateway|api|unsupported`)
    - `path`
    - `provenance`
  - optional imported-asset metadata:
    - `source_type`
    - `source_path`
    - `import_batch_id`
    - `analysis_status`
    - `quarantine_reason`
    - `conversion_status`
    - `default_memory_surface`

## `.atrakta/projections/manifest.json`

- `v` (current: `1`)
- `entries[]`
  - `interface`, `kind`, `files[]`
  - `source_hash`, `render_hash`
  - `status`, `updated_at`

## `.atrakta/extensions/manifest.json`

- `v` (current: `1`)
- `entries[]`
  - `kind`, `id`, `files[]`
  - `source_hash`, `render_hash`
  - `status`, `updated_at`

## `.atrakta/progress.json`

- `active_feature`
- `completed_features`
- `last_commit_hash`
- `updated_at`

## `.atrakta/run-checkpoints/latest.json`

- `feature_id`, `interfaces`, `sync_level`
- `stage`, `outcome`, `reason`
- `detect_reason`, `plan_id`, `task_graph_id`, `apply_result`

## `.atrakta/metrics/runtime.json`

- `v` (current: `1`)
- `updated_at`
- `samples_ms.<command>[]` (recent `start`/`doctor` samples, max 64)

## `.atrakta/runtime/auto-state.v1.json`

- `v` (current: `1`)
- `updated_at`
- `last_target_set`
- `last_source` (`explicit` / `trigger` / `auto_last` / `prompt`, etc.)
- `usage.<interface>.count`
- `usage.<interface>.last_used_at`

## `.atrakta/runtime/meta.v2.json`

- `v` (current: `2`)
- `updated_at`
- `entries.<key>`
  - `updated_at`
  - `stamp`
  - `config_hash`
  - `ttl_seconds`
  - `payload`
- Current major keys:
  - `repo_map`
  - `events_verify`
  - `wrapper_cache`
  - `start_fast_v2`
    - `contract_hash`, `workspace_stamp`, `interfaces`, `feature_id`
    - `config_key`, `outcome`, `detect_reason`, `last_strict_at`

## `.atrakta/runtime/gc-state.v1.json`

- `v` (current: `1`)
- `updated_at`
- `last_auto_at`

## `.atrakta/runtime/gc-log.jsonl`

- Append-only
- `tmp`:
  - `total_bytes`, `threshold_bytes`, `dry_run_delete*`, `applied_delete*`
- `events`:
  - `proposal_only`, `total_bytes`, `threshold_bytes`, `proposals[]`

## `.atrakta/task-graph.json`

- `graph_id`, `plan_id`
- `task_count`, `edge_count`, `digest`
- `tasks[].id`, `tasks[].index`, `tasks[].path`, `tasks[].op`
- `tasks[].task_blocked_by`

## `.atrakta/policies/prompt-min.json`

- `id` (example: `prompt-min@1`)
- `apply` (current: `conditional`)
- `require_goal_prefix`
- `goal_label` (current default: `Goal`)
- `required`
