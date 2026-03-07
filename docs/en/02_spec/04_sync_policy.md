# Sync Policy

[English](./04_sync_policy.md) | [日本語](../../ja/02_仕様/04_同期ポリシー.md)

## Sync Levels

- `0`: do not generate sync proposals (default)
- `1`: proposal-only (generate proposals; explicit approval required for apply)
- `2`: strict (AGENTS-derived proposals are not used in decision making)

When `--sync-level` is omitted, `ATRAKTA_SYNC_LEVEL` is used.

## Proposal-only Principle

- AGENTS instructions are never applied immediately.
- Use `doctor --sync-proposal` to inspect diffs.
- Use `doctor --apply-sync` for explicit approved apply.

## Current Apply Targets

- `hints.prefer`
- `hints.disable_interfaces`

Unsupported interface IDs are filtered out automatically.
