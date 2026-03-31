# Surface Portability

## Goal

Atrakta v1 does not pursue cross-IDE UX parity.
It guarantees the same semantic contract, safety decision, and stop condition
across surfaces while allowing native UX differences.

## Non-Goals

- cross-IDE identical prompts or layout
- forcing one interaction model across CLI and IDE
- direct regeneration of all existing surface files

## v1 Target Vocabulary

- `agents_md`
- `ide_rules`
- `repo_docs`
- `skill_bundle`

## v1 Source Vocabulary

- `canonical_policy`
- `workflow_binding`
- `skill_asset`
- `repo_docs`
- `agents_md`
- `ide_rules`

`workflow_bundle` and `runtime_hook` are not portability targets in v1.

## Ownership

- existing `AGENTS.md`, IDE rules, and repo docs are advisory-read-first
- canonical store remains the semantic source of truth
- v1 does not regenerate surface files from canonical state
- managed writes remain limited to `.atrakta/generated/**`

## Resolution Contract

`resolve_surface_portability(input) -> portability_decision`

Decision includes:

- `supported_targets[]`
- `degraded_targets[]`
- `unsupported_targets[]`
- `ingest_plan[]`
- `projection_plan[]`
- `portability_status`

## Standard Degradation

Default degradation policy is `proposal_only`.

- degraded or unsupported portability does not silently succeed
- inspect and preview may continue
- apply is disabled
- `run` returns portability metadata with `next_allowed_action=propose`

`BLOCK` remains reserved for separate policy or integrity violations.
