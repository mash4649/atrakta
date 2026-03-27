# 現在進捗トラッカー（Phase 1 中心）

更新日: 2026-03-27

本ファイルは「今どこまで終わっていて、次に何をやるか」を短く追うための単一トラッカー。
詳細な履歴は `implementation-status.md`、全体計画は `maturity-roadmap.md` を参照。

## 1. 残タスク（作業単位）

| 項目 | 状態 | 備考 |
|------|------|------|
| `wrap`（install/uninstall/run） | 未着手 | `init` 依存の主要要素 |
| `hook`（install/uninstall/status/repair） | 未着手 | `init` 依存の主要要素 |
| `ide-autostart`（install/uninstall/status） | 未着手 | `hook` 連携 |
| projection 拡張（`ide_rules`/`repo_docs`/`skill_bundle`） | 未着手 | `AGENTS.md` 以外の実体化 |
| run-events taxonomy の拡張（未実装コマンド分） | 進行中 | `projection/gc/migrate` まで追加済み |

## 2. 直近で完了した項目

| 項目 | 状態 | 備考 |
|------|------|------|
| `start` hardening（P1-H-001） | 完了 | edge-case diagnostics / output-envelope consistency / schema validation |
| `projection render/status/repair` 初期版 | 完了 | `AGENTS.md` を対象に実装済み |
| `gc` | 完了 | `tmp/events` スコープ対応 |
| `migrate check` | 完了 | contract/state/progress/task-graph/run-events チェック |
| `run-events` 規約固定（第1段） | 完了 | ADR-004 更新、`projection/gc/migrate` 発火追加 |
| `init` 最小統合版 | 完了 | `start` 委譲、`init.begin/init.step/init.end` 発火 |
| v0 ↔ 0.14.1 `events.jsonl` 双方向コンバータ | 対象外 | 不要方針で除外済み |

## 3. ロードマップ進捗（簡易）

### Phase 1（Runtime Integration）

| タスク群 | 進捗 |
|---------|------|
| `start`/`resume` 基盤、state/progress/task-graph、run-events 導入 | 完了 |
| `start` hardening（P1-H-001） | 完了 |
| run-events taxonomy 拡張と発行範囲整備 | 進行中 |
| `init` 最小統合版 | 完了 |

### Phase 2（Tool Binding / Projection）

| タスク群 | 進捗 |
|---------|------|
| `projection render/status/repair` | 初期版完了 |
| `wrap`/`hook`/`ide-autostart` | 未着手 |
| projection 対象拡張 | 未着手 |

### Phase 3（Operational Maturity）

| タスク群 | 進捗 |
|---------|------|
| `gc` / `migrate check` | 完了（初期版） |
| doctor 拡張、エラー体系強化など | 未着手 |

## 4. 次にキリがいいところ

1. `wrap run` を先行実装して実運用導線を作る
2. `hook` / `ide-autostart` を順に接続する
3. run-events taxonomy の残りイベント種別を詰める
