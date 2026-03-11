# Import Review Runbook

[English](../../en/03_operations/10_import_review_runbook.md) | [日本語](./10_ImportReviewRunbook.md)

## Flow

1. 外部ソースをローカルへ clone（Atrakta runtime 外）
2. local directory を import
3. analyze-only を実行
4. quarantine 結果を review
5. 承認済み skill を recipe candidate へ convert
6. memory promotion を review
7. enable/execution は manual gate 維持

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

- import 完了は実行許可ではない。
- auto analyze は auto convert / auto enable をしない。
- `exploration catalog --reviewed-only` は opt-in + review gate 前提。
