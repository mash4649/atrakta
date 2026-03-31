# 現在進捗トラッカー（Phase 1 中心）

更新日: 2026-03-28

本ファイルは「今どこまで終わっていて、次に何をやるか」を短く追うための単一トラッカー。
詳細な履歴は `implementation-status.md`、全体計画は `maturity-roadmap.md` を参照。

## 1. 残タスク（作業単位）

| 項目 | 状態 | 備考 |
|------|------|------|
| `projection 拡張`（`ide_rules`/`repo_docs`/`skill_bundle`） | 進行中 | render は対象拡張済み。status / repair で追跡継続 |
| run-events taxonomy の拡張（未実装コマンド分） | 進行中 | `projection/gc/migrate` まで追加済み |

## 2. 直近で完了した項目

| 項目 | 状態 | 備考 |
|------|------|------|
| `start` hardening（P1-H-001） | 完了 | edge-case diagnostics / output-envelope consistency / schema validation |
| `projection render/status/repair` 初期版 + Phase 2 target expansion | 完了 | `AGENTS.md` と `ide_rules` / `repo_docs` / `skill_bundle` を実体化 |
| `cmd_run.go` ライフサイクル分割（P1-H-008） | 完了 | `cmd_run_lifecycle.go` に分離し、main の責務を縮小 |
| task-graph 意味論 hardening（P1-H-009） | 完了 | replay 保存時のノード寿命を保持し、空 replay でも既存 DAG を保持 |
| `gc` | 完了 | `tmp/events` スコープ対応 |
| `migrate check`（P3-OPS-002） | 完了 | `state` / `progress` / `task-graph` / `run-events` の schema_version チェック。read-only 維持。 |
| `doctor` 拡張（P3-OPS-003） | 完了 | state / projection / event chain の診断と next action を JSON で返す。修復・書き込みなし。 |
| `構造化エラーメッセージ`（P3-ERR-001） | 進行中 | exit 1 出力を error envelope 化。 |
| `run-events` 規約固定（第1段） | 完了 | ADR-004 更新、`projection/gc/migrate` 発火追加 |
| `init` 最小統合版 | 完了 | `start` 委譲、`init.begin/init.step/init.end` 発火 |
| `ide-autostart` | 完了 | `.vscode/tasks.json` / `.cursor/autostart.json` 生成 |
| v0 ↔ 0.14.1 `events.jsonl` 双方向コンバータ | 対象外 | 不要方針で除外済み |

## 3. ロードマップ進捗（簡易）

### Phase 1（Runtime Integration）

| タスク群 | 進捗 |
|---------|------|
| `start`/`resume` 基盤、state/progress/task-graph、run-events 導入 | 完了 |
| `start` hardening（P1-H-001） | 完了 |
| run-events taxonomy 拡張と発行範囲整備 | 進行中 |
| task-graph 意味論 hardening（P1-H-009） | 完了 |
| `init` 最小統合版 | 完了 |

### Phase 2（Tool Binding / Projection）

| タスク群 | 進捗 |
|---------|------|
| `projection render/status/repair` | 完了（target expansion 反映済み） |
| `wrap`/`hook`/`ide-autostart` | 完了 |
| projection 対象拡張 | 進行中 |

### Phase 3（Operational Maturity）

| タスク群 | 進捗 |
|---------|------|
| `gc` / `migrate check` | 完了（初期版） |
| `doctor` 拡張 | 完了 |
| `構造化エラーメッセージ` | 進行中 |

## 4. 次にキリがいいところ

1. `P3-ERR-001` の構造化エラー体系を完了する
2. `P3-QUAL-001` / `P3-QUAL-002` の受け入れ・評価基盤を整える
3. run-events taxonomy の残りイベント種別を詰める
