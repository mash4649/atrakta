# Troubleshooting

[English](./03_troubleshooting.md) | [日本語](../../ja/03_運用/03_トラブルシュート.md)

## `events.jsonl corrupted`

- Caused by hash chain mismatch.
- Legacy event logs (`schema v1`) are unsupported.
- Archive `.atrakta/events.jsonl` and run `atrakta doctor` with a fresh log.

## `migrate check failed: schema_version unsupported`

- Old schema events remain.
- Latest spec does not accept old schema.
- Archive old log and retry.

## `task graph invalid`

- `.atrakta/task-graph.json` is broken or manually edited inconsistently.
- Re-run `atrakta start` to regenerate.

## `feature switch blocked`

- `progress.json.active_feature` differs from requested feature.
- Complete current feature or recover consistency first.

## Resume after interruption

- Use `atrakta resume` to restart with latest checkpoint conditions.
- Override with `--feature-id` / `--interfaces` / `--sync-level` when needed.

## Wrapper not working

- Check `ATRAKTA_WRAP_DISABLE=1` is not set.
- Verify wrapper binaries under `~/.local/bin`.
- Re-run `atrakta wrap install` (it can patch PATH priority in rc files when needed).
- Restart shell and confirm `command -v cursor` resolves to `~/.local/bin` first.

## Hook runs unexpectedly

- Pause: `ATRAKTA_HOOK_DISABLE=1`
- Remove: `atrakta hook uninstall`

## `command not found: atrakta`

- Usually missing binary or PATH entry.
- Quick fix on macOS/Linux:
  - `curl -fsSL https://raw.githubusercontent.com/mash4649/atrakta/main/scripts/install.sh | bash`
- macOS / Linux:
  - `mkdir -p ~/.local/bin`
  - `install -m 0755 ./atrakta ~/.local/bin/atrakta`
  - `hash -r`
  - `command -v atrakta`
- Windows (PowerShell):
  - `$targetDir = "$env:USERPROFILE\AppData\Local\Programs\atrakta"`
  - `Copy-Item .\atrakta.exe "$targetDir\atrakta.exe" -Force`
  - Add `$targetDir` to user PATH
  - `where atrakta`

## Unexpected strict behavior

- With `sync-level=2`, AGENTS-derived proposals are not used for decisions.

## `.tmp` keeps growing

- Dry-run: `atrakta gc --scope tmp`
- Apply: `atrakta gc --scope tmp --apply`
- If auto GC does not run, check `ATRAKTA_GC_DISABLE`.

## `events.jsonl` is too large

- Inspect proposal: `atrakta gc --scope events`
- Current behavior is proposal-only (no automatic mutation).
