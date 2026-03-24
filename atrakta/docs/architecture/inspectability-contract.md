# Inspect Preview Simulate Contract

## Standard Resolver Output

All resolver outputs must include:

- input
- decision
- reason
- evidence
- next_allowed_action

## Inspect Requirements

- layer boundary inspect
- guidance strength inspect
- projection eligibility inspect
- strict lifecycle inspect
- legacy progression inspect
- extension order verify

## Preview Requirements

- managed scope preview
- mutation plan preview
- audit retention dry-run preview

## Simulate Requirements

- failure routing simulate
- policy conflict simulate

Inspectability is not optional and is part of each resolver contract by default.

## Bundle Schemas

- CLI input bundle: `schemas/operations/bundle-input.schema.json`
- CLI output bundle: `schemas/operations/bundle-output.schema.json`
- Fixture report: `schemas/operations/fixtures-report.schema.json`

## Artifact Export

- `--artifact-dir` writes JSON snapshots without changing stdout behavior
- `inspect`, `preview`, and `simulate` export `*.bundle.json`
- `export-snapshots` additionally exports onboarding-injected variants:
  - `inspect.onboard.bundle.json`
  - `preview.onboard.bundle.json`
  - `simulate.onboard.bundle.json`
- `run-fixtures` exports `fixtures.report.json`
