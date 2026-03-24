# Zero-Config Onboarding

## Intent

Start safe with no manual settings, then progressively disclose control surfaces.

## Core Rules

- detect first
- safe default
- proposal-first mutation
- lazy canonicalization
- progressive disclosure

## Modes

- new_project
- brownfield_project

Mode is inferred automatically with confidence.
Ask for confirmation only when confidence is low.

## Detect Targets

- `agents_md` and `ide_rules`
- `workflow_binding`
- runtime and tool configs
- test and script surfaces
- risk candidates for external send and destructive actions

## Initial Safety Defaults

- read-only allow
- local write proposal-only
- destructive deny
- external send deny
- unknown capability strict
- unmapped guidance advisory only

## Onboarding Proposal Bundle

Minimum fields:

- detected_assets
- detected_risks
- inferred_mode
- inferred_managed_scope
- inferred_capabilities
- inferred_guidance_strength
- inferred_default_policy
- inferred_failure_routing
- conflicts
- suggested_next_actions

`inferred_failure_routing` is computed by mapping onboarding conflicts into
strict triggers and evaluating them through `resolve_failure_tier`.
`detected_risks` is computed from `workflow_binding`/script/package surfaces to capture
external send, destructive script, and secrets exposure candidates.

## Detect and Infer APIs

- detect_project_root
- detect_mode
- detect_assets
- infer_managed_scope
- infer_capabilities
- infer_guidance_strength
- infer_default_policy
- build_onboarding_proposal

## CLI

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`
- `go run ./cmd/atrakta onboard --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta inspect --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
