# Atrakta Documentation

Japanese / 日本語: [docs/ja/README.md](ja/README.md)

This is the entry point for usage and specification docs.

## Start Here

- [Run Contract](architecture/run-contract.md)
- [Adapter Invocation Contract](architecture/adapter-invocation.md)
- [Surface Portability](architecture/surface-portability.md)
- [Concept Coverage Matrix](plan/concept-coverage-matrix.md)
- [Zero-Config Onboarding](architecture/onboarding-zero-config.md)
- [Execution Plan](plan/v0-execution-plan.md)
- [Implementation Status](plan/implementation-status.md)
- [Maturity and Adoption Roadmap](plan/maturity-roadmap.md)
- [Phase 1: `atrakta start` Design](plan/phase1-start-command-design.md)
- [Phase 1: `atrakta start` Issue Backlog](plan/phase1-start-command-issues.md)
- [Package Manager Publishing](plan/package-manager-publishing.md)
- [Inspect / Preview / Simulate Contract](architecture/inspectability-contract.md)
- [ADR-004: Events Stream Schema](decisions/ADR-004-events-stream-schema.md)
- [License Options (proposal)](plan/license-options.md)

## How To Use

Primary CLI:

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`

Legacy/debug entrypoints:

- `go run ./cmd/atrakta onboard --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta inspect --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta preview --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta simulate --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta accept --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta mutate inspect --target .atrakta/generated/repo-map.generated.json`
- `go run ./cmd/atrakta mutate propose --target .atrakta/generated/repo-map.generated.json --content '{"k":"v"}'`
- `go run ./cmd/atrakta mutate apply --project-root . --target .atrakta/generated/repo-map.generated.json --content-file patch.json --allow`
- `go run ./cmd/atrakta audit append --action manual_check --level A2`
- `go run ./cmd/atrakta audit verify --level A2`
- `go run ./cmd/atrakta doctor --execute`
- `go run ./cmd/atrakta parity --execute`
- `go run ./cmd/atrakta integration --execute`
- `go run ./cmd/atrakta extensions --project-root .`
- `go run ./cmd/atrakta run-fixtures --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta export-snapshots --dir fixtures/snapshots`
- `go run ./cmd/atrakta verify-coverage`

## Specification Map

- [Layer Boundary Contract](architecture/layer-boundary.md)
- [Guidance Strength and Precedence](architecture/guidance-precedence.md)
- [Projection Model and One-Way Rule](architecture/projection-model.md)
- [Failure Routing and Strict Lifecycle](architecture/failure-routing.md)
- [Managed Scope and Mutation Policy](architecture/managed-scope.md)
- [Legacy Governance and Promotion Rule](architecture/legacy-governance.md)
- [Operations Capability Model](architecture/operations-capability.md)
- [Extension Boundary and Evaluation Order](architecture/extension-boundary.md)
- [Plugin SDK Interface Definition](architecture/plugin-sdk.md)
- [MCP Server Integration](architecture/mcp-server.md)
- [Audit Integrity and Retention Policy](architecture/audit-integrity.md)
- [Surface Portability](architecture/surface-portability.md)
- [Community Templates](community-templates.md)

## Artifacts

- `schemas/` contains the contract definitions.
- `fixtures/` contains deterministic fixture inputs and snapshots.
- `operations/README.md` describes runtime-facing operations and snapshot policy.
- `tests/` contains contract and resolver verification.
