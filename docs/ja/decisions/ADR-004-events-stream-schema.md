# ADR-004 イベントストリームスキーマ（v0）

状態: 採用

## 背景

v0 には監査・デバッグ・再現（「何が起きたか」）のための **append-only** なイベントストリームが必要であり、整合性（改ざん検知）を検証できなければならない。

本リポジトリには現時点で、互いに近いが異なる 2 系統のストリームが存在する:

- `/.atrakta/events.jsonl`: 0.14.1 風の events ストリーム（`schema_version: 2`、`prev_hash` チェーン、`type` 多数）
- `/.atrakta/audit/events/install-events.jsonl`: v0 が現在使っている監査チェーン（`internal/audit/chain.go`）。A0〜A3 の整合性レベルを持つ

ロードマップ上、v0 は **refresh line**（0.14.1 との完全パリティは必須ではない）として位置付けられている。ただし移行は明示的でなければならない。

## 決定

v0 は Phase 1 において **v0 独自の、バージョン付きイベントストリームスキーマ**を正とする。0.14.1 の `events.jsonl` 形式は **履歴として扱う外部データ**であり、v0 の正規保存形式としては採用しない。

レイヤ整合:

- v0 のイベントストリームは **canonical の監査成果物**であり（ADR-001）、append-only を前提とする。
- 投影はイベント由来の要約を出してよいが、投影から canonical のイベントへ書き戻してはならない（ADR-003）。

## v0 イベントストリーム: 保存場所

Phase 1 の保存場所は audit ストア配下とする:

- `.atrakta/audit/events/run-events.jsonl`（新規; 実行/セッション系イベント）
- `.atrakta/audit/events/install-events.jsonl`（既存; install/onboarding 系イベント）

理由: 「監査」を整合性境界として固定し、保持/GC で監査ヘッドを特別扱いしやすくする。

## v0 イベントストリーム: スキーマ（v0, schema_version = 1）

1 行 = 1 JSON オブジェクト（JSONL）。

### 必須フィールド

- `schema_version`（number）: **1**
- `seq`（number）: ファイル内で 1 始まりの単調増加シーケンス
- `timestamp`（string）: RFC3339 / `date-time`
- `event_type`（string）: 安定したイベント識別子（後述）
- `integrity_level`（string）: `"A0" | "A1" | "A2" | "A3"`
- `payload`（object）: イベント固有データ（JSON object であること）

### 整合性フィールド（integrity_level により必須）

- `payload_hash`（string）: payload の canonical bytes に対する SHA-256（hex）。A1+ で必須
- `prev_hash`（string）: 直前イベントの `hash`。A2+ で必須（`seq=1` は空）
- `hash`（string）: チェーン hash（hex）。A2+ で必須

### 任意フィールド（Phase 1）

- `event_id`（string）: ファイル横断の相関用 ID（推奨、ただしチェーン整合性の必須条件ではない）
- `actor`（string）: `"orchestrator" | "kernel" | "worker" | ...`（自由）
- `run_id`（string）: 1 回の `start` 相当実行を識別する ID
- `interface`（string）: 例: `"cursor"`, `"vscode"`, `"cli"`
- `feature_id`（string）: 可能ならユーザーの作業単位 ID

### 例（A2）

```json
{"schema_version":1,"seq":7,"timestamp":"2026-03-25T12:34:56Z","event_type":"plan.created","integrity_level":"A2","payload":{"task_count":3,"task_graph_id":"sha256:..."},"payload_hash":"<hex-sha256>","prev_hash":"<hex-sha256>","hash":"<hex-sha256>","run_id":"run-20260325-123456","actor":"kernel"}
```

## ハッシュ戦略（canonical, Phase 1）

Phase 1 では、既に `internal/audit/chain.go` に実装されている方式を v0 スキーマに適用する:

- **payload の canonical bytes**: Go の `encoding/json` による `payload` の JSON 直列化（UTF-8、空白なし、Go エンコーダが出力する安定したキー順）
- `payload_hash = sha256(payload_bytes)`（hex）
- `hash = sha256( fmt("%d|%s|%s|%s", seq, event_type, payload_hash, prev_hash) )`（hex）

注意:

- `timestamp` は Phase 1 ではチェーン hash に含めない。既存の検証器と整合させ、似て非なる整合性方式を増やさないため。
- A3 では既存の head checkpoint と同様に、チェックポイントファイル（例: `.atrakta/audit/checkpoints/run-head.json`）を置いてよい。

## 最小イベント種別（Phase 1）

Phase 1 では、`start` 相当の実行を高レベルに説明できる最小集合のみを必須とする:

- `onboarding.accepted`
- `init.begin`
- `init.step`
- `init.end`
- `start.begin`
- `detect.performed`
- `plan.created`
- `apply.performed`
- `gate.result`
- `projection.rendered`（または `projection.skipped`）
- `projection.status`
- `projection.repaired`
- `gc.planned`
- `gc.applied`
- `migrate.checked`
- `start.end`
- `error.raised`（実行を中断する失敗）

0.14.1 の全イベント分類をそのまま持ち込むことは Phase 1 の要件ではない。

## 互換と移行戦略（明示）

v0 は 0.14.1 の `/.atrakta/events.jsonl` スキーマのパリティを保証しない。代わりに:

- **read**: 既存ワークスペースに `/.atrakta/events.jsonl` があっても、v0 はそれを履歴/外部データとして扱い、書き換えない。
- **write**: v0 が書き込むのは `/.atrakta/audit/events/` 配下の v0 監査ストリームのみ。
- **双方向コンバータ層は置かない**: 互換は v0 のイベント taxonomy と payload 規約を固定することで維持する。

### イベントマッピング規約（v0 正規）

- `start.begin`: `path`、`canonical_state`、`interface_id`、`apply_requested` を含める。
- `detect.performed`: `step_count`、`final_allowed_action` を含める。
- `plan.created`: `planned_target_count`、`planned_target_paths` を含める。
- `gate.result`: `status`、`next_allowed_action` を含め、必要に応じて `approval_scope` を含める。
- `apply.performed`: `applied_count`、`applied_target_paths` を含める。
- `projection.rendered|projection.status|projection.repaired`: `target_path`、`drift`、`written` を含める。
- `gc.planned|gc.applied`: `scope`、`apply`、`candidate_count`、`removed_count` を含める。
- `migrate.checked`: `ok`、`check_count` を含める。

これにより、移行方針は明示を維持しつつ、v0 の refresh line 方針と整合する。

## 影響

- v0 は A0〜A3 の整合性レベルと整合した、安定・版付きのイベント契約を得る。
- 0.14.1 互換は「履歴データとして扱う」方針で維持し、v0 の正規モデルを縛らない。
- Phase 1 以降も v0 taxonomy を拡張しつつ、変換レイヤとの結合を避けられる。
