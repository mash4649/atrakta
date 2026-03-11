# Daily Operations

[English](./02_daily_operations.md) | [日本語](../../ja/03_運用/02_日常運用.md)

## Basic Operations

- Launch each interface via wrapper in normal usage.
- In VSCode-compatible IDEs, `ide-autostart` runs `start` on workspace open.
- Manual `atrakta start` also resolves from trigger/last-success state.
- In large repositories, tune repo map budget with `--map-tokens` / `--map-refresh`.
- Run `atrakta doctor` after major changes.
- Run parity/integration checks after major changes.
- For storage growth, dry-run with `atrakta gc --scope tmp`, then apply with `--apply` if safe.
- Legacy event logs (`schema v1`) are unsupported; archive and restart with new logs if needed.

### Recommended Post-change Checks

```bash
atrakta doctor
atrakta doctor --parity
atrakta doctor --integration
atrakta projection status --json
```

## Projection Workflow for Interface Switching

- Before switching active tools, run explicit projection renders to reduce drift.

```bash
atrakta projection render --interface cursor
atrakta projection render --interface claude_code
atrakta projection render --interface codex_cli
atrakta projection status --json
```

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

Save proposal snapshots for review/audit:

```bash
mkdir -p .atrakta/proposals
atrakta doctor --sync-proposal > .atrakta/proposals/sync-$(date +%Y%m%d-%H%M%S).json
```

Apply with explicit approval:

```bash
atrakta doctor --sync-proposal --apply-sync
```
