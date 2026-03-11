# GC Operations

[English](./05_gc_operations.md) | [日本語](../../ja/03_運用/05_GC運用.md)

## Policy

- Never delete core context (`state.json`, `progress.json`, checkpoints, auto-state)
- Keep projection/extension manifests out of GC scope (`.atrakta/projections/manifest.json`, `.atrakta/extensions/manifest.json`)
- `.tmp` is auto-GC target only when threshold is exceeded
- `events.jsonl` remains proposal-only (no automatic mutation)
- Every GC run logs dry-run/applied results

## Auto GC (outside `start` critical path)

- Auto GC is best-effort async and only after `start` reaches `DONE`
- No auto trigger under minimum interval (default 60 minutes)
- No `.tmp` deletion while under threshold

## Manual GC

```bash
# dry-run
atrakta gc --scope tmp

# apply
atrakta gc --scope tmp --apply

# events proposal only
atrakta gc --scope events
```

## Key Environment Variables

- `ATRAKTA_GC_DISABLE=1` disables auto GC
- `ATRAKTA_GC_TMP_MAX_BYTES` `.tmp` threshold (default 2GiB)
- `ATRAKTA_GC_TMP_RETENTION_DAYS` preferred retention window (default 7 days)
- `ATRAKTA_GC_AUTO_MIN_INTERVAL_MIN` minimum auto GC interval (default 60 minutes)

## Logs

- `.atrakta/runtime/gc-log.jsonl`
  - `tmp.dry_run_delete*`
  - `tmp.applied_delete*`
  - `events.proposals[]`
