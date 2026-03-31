# Contributing to Atrakta

Thank you for contributing. This repository is **contract-first** and **resolver-first**.
The machine-executable contract lives at `.atrakta/contract.json`, and the authoritative
schema source of truth is under `schemas/`.

If you haven't already, start from `docs/architecture/run-contract.md` and
`docs/architecture/adapter-invocation.md` to align on the execution and adapter contracts.

## Prerequisites

- Go **1.26** (CI uses Go 1.26)
- macOS, Linux, or Windows (CI runs on Linux; keep changes portable)

No additional tooling is required. Please avoid introducing new tool dependencies.

## Repository shape (quick orientation)

- `schemas/`: contract schemas (source of truth)
- `resolvers/`: deterministic decision logic
- `adapters/`: thin invokers (must call `atrakta run`, not internal compositions)
- `canonical/`: canonical state (single write source)
- `fixtures/`: deterministic fixture inputs and committed snapshots
- `tests/`: contract and resolver verification

## Development setup

Clone the repository and run commands from the `atrakta/` directory.

Recommended local setup:

1. Install Go 1.26.
2. Clone the repository.
3. `cd atrakta`
4. Verify the toolchain:

```bash
go build ./...
go test ./...
go run ./cmd/atrakta run-fixtures
go run ./cmd/atrakta verify-coverage
```

If you prefer a cached Go build directory, create `.tmp/go-build` and reuse it across runs:

```bash
mkdir -p .tmp/go-build
GOCACHE="$(pwd)/.tmp/go-build" go test ./...
```

Common entrypoints (local development):

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`

Note: `atrakta run` is the **single execution primitive**. Other commands are
implementation parts and debug surfaces.

## Running tests (local)

Run all tests:

```bash
go test ./...
```

If you want to mirror CI caching behavior:

```bash
mkdir -p .tmp/go-build && GOCACHE="$(pwd)/.tmp/go-build" go test ./...
```

Run the snapshot and coverage gates in the same order used by CI:

```bash
go run ./cmd/atrakta run-fixtures
go run ./cmd/atrakta verify-coverage
```

Also run the CI gates locally when changing schemas/resolvers/fixtures:

```bash
mkdir -p .tmp/go-build && GOCACHE="$(pwd)/.tmp/go-build" go run ./cmd/atrakta verify-coverage
```

## Snapshot gate workflow (fixtures/snapshots)

CI regenerates a fixed set of JSON artifacts and fails if the regenerated output differs
from the committed snapshots under `fixtures/snapshots/`.

To regenerate all snapshots (the same operation CI performs):

```bash
mkdir -p .tmp/atrakta-artifacts
go run ./cmd/atrakta export-snapshots --dir .tmp/atrakta-artifacts
```

Then update the committed snapshots:

```bash
cp -f .tmp/atrakta-artifacts/*.json fixtures/snapshots/
```

Finally re-run:

```bash
go test ./...
go run ./cmd/atrakta verify-coverage
```

### When should snapshots change?

Snapshots should change only when:

- The behavior of a resolver or contract meaningfully changes **by design**, or
- A schema change requires a different envelope/output, or
- A fixture was added/updated intentionally.

If snapshots change unexpectedly, treat it as a regression and investigate determinism and
contract alignment first.

## Adding fixtures

Fixtures are intended to be:

- **Deterministic**
- **Minimal** (small, targeted cases)
- **Contract-representative** (exercise the relevant schema/resolver paths)

General workflow:

1. Add or update fixture inputs under `fixtures/` following existing naming conventions.
2. Regenerate snapshots via `export-snapshots`.
3. Commit the updated files under `fixtures/snapshots/`.
4. Run `go run ./cmd/atrakta verify-coverage` to ensure explicit coverage mapping exists.

If you add a new fixture category/output, ensure the export and verification logic includes it
and that CI gates remain deterministic.

## Schema changes policy

Atrakta is contract-driven. Schema changes must be deliberate and come with corresponding:

- Updated fixtures and snapshots (when output shape changes)
- Updated coverage mapping (`go run ./cmd/atrakta verify-coverage` must pass)
- Updated documentation where the contract is described (as needed)

Guidelines:

- Prefer additive changes (new fields) over breaking changes.
- If a change is breaking, ensure it is accompanied by a clear rationale and a migration plan.
- Keep the machine contract `.atrakta/contract.json` aligned with `schemas/` expectations.

CI will fail if `schemas/operations/*.schema.json` or resolvers are added without explicit
coverage mapping.

## Resolver contract expectations

Resolvers are the deterministic core. Contributions must preserve:

- **Determinism**: no reliance on wall-clock time, randomness, map iteration order, or
  environment-specific behavior unless explicitly modeled as an input.
- **Stable ordering**: output ordering should be intentional and reproducible.
- **Policy boundaries**: operations must not bypass managed-scope and approval boundaries.
- **Managed scope rules**: do not mutate `unmanaged_user_region`; fall back to proposal-only
  under ambiguity (see `docs/architecture/managed-scope.md`).
- **Approval gating**: write paths require explicit approval (`--approve` or interactive),
  and must emit `NEEDS_APPROVAL` when required.
- **Adapter contract**: adapters must invoke `atrakta run` and handle exit codes
  `0/1/2/3` as specified in `docs/architecture/adapter-invocation.md`.

If you change resolver behavior, update fixtures/snapshots and ensure portability metadata
and exit code semantics remain aligned with the run contract.

## Pull request checklist

- [ ] `go test ./...` passes
- [ ] `go build ./...` passes
- [ ] `go run ./cmd/atrakta run-fixtures` passes
- [ ] `go run ./cmd/atrakta verify-coverage` passes (when touching schemas/resolvers/fixtures)
- [ ] Snapshots updated and committed (when behavior/output changes intentionally)
- [ ] Documentation updated when contracts change
