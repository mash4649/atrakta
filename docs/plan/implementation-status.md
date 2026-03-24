# Implementation Status

## Completed

- Issue 1 baseline
  - layer ownership contract documented
  - `classify_layer` resolver implemented with tests
- Issue 2 baseline
  - guidance strength and precedence contract documented
  - `resolve_guidance_precedence` resolver implemented with tests
- Issue 3 baseline
  - projection model documented
  - `check_projection_eligibility` resolver implemented with tests
- Issue 4 baseline
  - failure routing and strict lifecycle documented
  - `resolve_failure_tier` resolver implemented with tests
  - `strict_state_machine` resolver implemented with tests
- Issue 5 baseline
  - managed scope and mutation policy documented
  - `check_mutation_scope` resolver implemented with tests
- Issue 6 baseline
  - legacy governance and promotion rule documented
  - `resolve_legacy_status` resolver implemented with tests
  - `detect_legacy_drift` resolver implemented with tests
- Issue 7 baseline
  - operations capability model documented
  - `resolve_operation_capability` resolver implemented with tests
- Issue 8 baseline
  - extension boundary and evaluation order documented
  - `resolve_extension_order` resolver implemented with tests
- Issue 9 baseline
  - audit integrity and retention policy documented
  - `resolve_audit_requirements` resolver implemented with tests
- Issue 10 baseline
  - inspect/preview/simulate output contract documented
  - cross-resolver output contract test implemented

## In Progress

- Integration baseline complete (snapshot gate enabled)
- `atrakta run` contract and adapter invocation docs added
- semantic portability v1 added for `agents_md`, `ide_rules`, `repo_docs`, and `skill_bundle`

## Next

No blocking baseline tasks remain for the current v0 contract scope.

## Integration Progress

- Added `atrakta run` as the primary execution primitive:
  - onboarding path for initial accept
  - canonical-present path for detect/plan/apply
- Added machine contract documentation for `.atrakta/contract.json`
- Added semantic portability resolver and run gating:
  - capability declarations loaded from `adapters/bindings/*/binding.json`
  - `resolve_surface_portability`
  - proposal-only fallback for degraded or unsupported surfaces
- Added CLI entrypoints under `cmd/atrakta`:
  - `inspect`
  - `preview`
  - `simulate`
  - `onboard`
  - `run-fixtures`
- Added CLI schema validation hooks for input and output bundles.
- Added zero-config onboarding proposal builder under `internal/onboarding`:
  - `detect_project_root`
  - `detect_mode`
  - `detect_assets`
  - `infer_managed_scope`
  - `infer_capabilities`
  - `infer_guidance_strength`
  - `infer_default_policy`
  - `build_onboarding_proposal`
- Added onboarding proposal schema validation hook:
  - `schemas/operations/onboarding-proposal-bundle.schema.json`
- Added onboarding conflict -> failure routing linkage:
  - onboarding emits `inferred_failure_routing`
  - derived via `resolve_failure_tier` with strict triggers
- Added onboarding-to-pipeline integration path:
  - `inspect/preview/simulate --onboard-root` injects onboarding-derived failure context into bundle execution
- Added onboarding risk detection:
  - `detected_risks` from package/workflow/script content scanning
- Added accept/persist flow:
  - `accept` writes `.atrakta/canonical`, `.atrakta/generated`, `.atrakta/state`, `.atrakta/audit`
- Added mutation 3-phase runtime command surface:
  - `mutate inspect|propose|apply`
- Added audit integrity runtime commands:
  - `audit append`
  - `audit verify`
- Added operations alias command surfaces:
  - `doctor`
  - `parity`
  - `integration`
- Added extension manifest resolution command:
  - `extensions`
- Added onboarding-injected pipeline snapshots:
  - `inspect.onboard.bundle.json`
  - `preview.onboard.bundle.json`
  - `simulate.onboard.bundle.json`
- Added JSON artifact export mode via `--artifact-dir`.
- Added ordered resolver pipeline runner under `internal/pipeline`.
- Added fixture runner under `internal/fixtures`.
- Added deterministic replay test for ordered pipeline output.
- Added fixture runner test to ensure fixture corpus passes.
- Added GitHub Actions CI workflow for `go test` and snapshot export.
- Added mandatory snapshot gate: CI compares generated artifacts with `fixtures/snapshots/*.json`.
- Added onboarding proposal snapshot to the same gate for deterministic zero-config inference output.
- Added schema-driven validation hooks that load and enforce:
  - `schemas/operations/bundle-input.schema.json`
  - `schemas/operations/bundle-output.schema.json`
  - `schemas/operations/fixtures-report.schema.json`
- Added coverage gate command `verify-coverage` for:
  - operations schema coverage policy (`schemas/operations/coverage-policy.json`)
  - resolver-to-fixture coverage mapping (`fixtures/resolver-fixture-map.json`)
- Added CI step to run `go run ./cmd/atrakta verify-coverage`.
- Expanded fixture families to cover:
  - `strict_state_machine`
  - `detect_legacy_drift`
- Added onboarding inference fixture coverage:
  - `fixtures/onboarding/onboarding-proposal.fixture.json`
  - validated via `run-fixtures` and captured in `fixtures.report.json` snapshot
