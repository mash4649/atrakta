# Daily Operations

[English](./02_daily_operations.md) | [日本語](../../ja/03_運用/02_日常運用.md)

## Basic Operations

- Launch each interface via wrapper in normal usage.
- In VSCode-compatible IDEs, `ide-autostart` runs `start` on workspace open.
- Manual `atrakta start` also resolves from trigger/last-success state.
- In large repositories, tune repo map budget with `--map-tokens` / `--map-refresh`.
- Run `atrakta doctor` after major changes.
- For storage growth, dry-run with `atrakta gc --scope tmp`, then apply with `--apply` if safe.
- Legacy event logs (`schema v1`) are unsupported; archive and restart with new logs if needed.

## Feature Workflow

Run with explicit feature id:

```bash
atrakta start --feature-id feat-auth
```

- If an active feature remains, switching to another feature is blocked.
- Completed feature ids are recorded in `completed_features`.

## Sync Workflow

View proposals:

```bash
atrakta doctor --sync-proposal
```

Apply with explicit approval:

```bash
atrakta doctor --sync-proposal --apply-sync
```
