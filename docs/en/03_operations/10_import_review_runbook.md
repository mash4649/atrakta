# Import Review Runbook

[English](./10_import_review_runbook.md) | [日本語](../../ja/03_運用/10_ImportReviewRunbook.md)

## Flow

1. Clone external source to local workspace (outside Atrakta runtime).
2. Import local directory.
3. Run analyze-only stage.
4. Review quarantine results.
5. Convert approved skills to recipe candidates.
6. Review memory promotion.
7. Keep enable/execution behind manual gate.

## Commands

```bash
atrakta import repo ./vendor/external-capabilities
atrakta import report <batch_id>
atrakta import pulse
atrakta capability analyze <capability_id>
atrakta recipe convert <capability_id> --deterministic-input-note "fixed input schema"
atrakta memory review <capability_id> --status approved --promote --operator ops-user
```

## Quarantine Review Checklist

- external calls scope
- filesystem scope
- network scope
- secret requirements
- nondeterministic behavior
- unbounded loops

## Notes

- Import completion is not execution permission.
- Auto analyze does not auto convert or auto enable.
- `exploration catalog --reviewed-only` is opt-in and review-gated.
