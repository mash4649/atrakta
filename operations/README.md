# Operations Surface

Operations are runtime-facing interfaces and aliases over resolver capabilities.
They are debug and compatibility surfaces; `atrakta run` is the primary
execution entrypoint.

- inspect
- preview
- simulate
- aliases (doctor, parity, integration)

Operations cannot bypass policy or approval boundaries.

## CLI

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`
- `go run ./cmd/atrakta inspect --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta preview --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta simulate --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta onboard --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta accept --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta mutate inspect --target .atrakta/generated/repo-map.generated.json`
- `go run ./cmd/atrakta mutate propose --target .atrakta/generated/repo-map.generated.json --content '{"k":"v"}'`
- `go run ./cmd/atrakta mutate apply --target .atrakta/generated/repo-map.generated.json --content-file patch.json --allow`
- `go run ./cmd/atrakta audit append --action manual_check --level A2`
- `go run ./cmd/atrakta audit verify --level A2`
- `go run ./cmd/atrakta doctor --execute`
- `go run ./cmd/atrakta parity --execute`
- `go run ./cmd/atrakta integration --execute`
- `go run ./cmd/atrakta extensions --project-root .`
- `go run ./cmd/atrakta run-fixtures --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta export-snapshots --dir fixtures/snapshots`
- `go run ./cmd/atrakta verify-coverage`

## Snapshot Policy

- CI regenerates `onboarding`, `inspect`, `preview`, `simulate`, `inspect.onboard`, `preview.onboard`, `simulate.onboard`, and `fixtures` snapshots.
- CI fails when generated snapshots differ from `fixtures/snapshots/*.json`.
- CI fails when `schemas/operations/*.schema.json` or resolvers are added without explicit coverage mapping.
- When resolver or schema behavior changes intentionally, regenerate snapshots and commit updates.
