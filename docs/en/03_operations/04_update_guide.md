# Update Guide

[English](./04_update_guide.md) | [日本語](../../ja/03_運用/04_更新手順.md)

## Target

- For users updating Atrakta CLI
- latest-only policy (no legacy behavior compatibility)

## 1. Personal Use (local-first)

1. Replace with the new `atrakta` binary.
   - If source-managed, rebuild from Atrakta source under target project:
     - `cd <project>/atrakta`
     - `go build -o ~/.local/bin/atrakta ./cmd/atrakta`
     - `hash -r`
2. Run `atrakta doctor`
3. Run `atrakta init --interfaces <primary-interface>` once (uniform for interactive/non-interactive)
4. Verify parity/integration health
   - `atrakta doctor --parity`
   - `atrakta doctor --integration`
   - `atrakta projection status --json`

## 2. Team Use (with CI)

1. Verify `go test ./...` after update in verification repository
2. Update pinned Atrakta version in CI first
3. Run `atrakta doctor` in target repository
4. Validate representative workflows: `start`, `resume`, `gc --scope tmp`
5. Finish with `doctor --parity` / `doctor --integration` / `projection status --json`

## 2.1 Distribution Artifact Build for Maintainers

1. Normal path:
   - push to `main` triggers `.github/workflows/release.yml`
2. Manual fallback:
   - `./scripts/build/build_release_artifacts.sh`
3. Outputs:
   - `.tmp/release/v<version>/packages`
   - `.tmp/release/v<version>/checksums.txt`
4. Users consume prebuilt binaries from Releases (Go not required)

## 3. Long-running / Critical Repositories

1. Back up `.atrakta/events.jsonl` before update
2. Back up projection/extension manifests before update
   - `.atrakta/projections/manifest.json`
   - `.atrakta/extensions/manifest.json`
3. Run `atrakta migrate check`
4. Re-sync in order: `atrakta doctor` -> `atrakta start`
5. Run `atrakta doctor --parity` / `atrakta doctor --integration`
6. Optionally inspect events proposals via `atrakta gc --scope events`

## 4. Non-interactive Environments (CI etc.)

- Explicitly handle exit codes of `start` / `init` / `resume`
  - `4`: `NEEDS_INPUT`
  - `5`: `NEEDS_APPROVAL`
  - `6`: `BLOCKED`
- Enable `ATRAKTA_STATUS_JSON=1` for machine-readable handling
