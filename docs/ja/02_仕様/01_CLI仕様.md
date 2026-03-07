# CLI仕様

[English](../../en/02_仕様/01_CLI仕様.md) | [日本語](./01_CLI仕様.md)


## 仕様バージョン

- 文書バージョン: `v0.14.1`
- 対象CLI: `atrakta` `v0.14.1`
- 最終更新日: `2026-03-04`

## コマンド

```bash
atrakta init [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>] [--no-hook]
atrakta start [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]
atrakta doctor [--sync-proposal] [--apply-sync] [--sync-level <0|1|2>]
atrakta gc [--scope <tmp,events>] [--apply] [--auto]
atrakta wrap install
atrakta wrap uninstall
atrakta wrap run --interface <id> --real <path> -- [args...]
atrakta hook install
atrakta hook uninstall
atrakta ide-autostart [install|uninstall|status]
atrakta migrate check
atrakta resume [--interfaces <id,id,...>] [--feature-id <id>] [--sync-level <0|1|2>] [--map-tokens <n>] [--map-refresh <sec>]
```

## 互換性方針

- latest-only 運用（後方互換は提供しない）。
- 旧データ/旧CLI前提は fail-closed とし、明示移行なしでは受け入れない。
- `migrate check` は `events.schema_version = 2` を前提とする。

## `init`

- 初回導入用の統合コマンド
- 実行順:
  1. `wrap install`
  2. `hook install`（`--no-hook` 指定時はスキップ）
  3. `ide-autostart install`（workspace `.vscode/tasks.json`）
  4. `start`
- `start` と同じ deferred outcome を返す（`NEEDS_INPUT` / `NEEDS_APPROVAL` / `BLOCKED`）。

## `start`

- contract読込/初期化
- events chain 検証（`VerifyChainCached`。キャッシュ不一致時は全件検証）
- Start Snapshot Fast Gate（`runtime/meta.v2.start_fast_v2`）一致時は strict pipeline をスキップ
- strict間隔（10分）経過、設定差分、managed artifact 欠損時は strict path に自動昇格
- イベント書き込みは batch/group-commit 最適化される（critical payloadは即sync）
- context 解決（`nearest_with_import`）と routing 決定
- repo map 生成/再利用（予算・更新間隔制御）と `repo_map` イベント記録
- prompt policy 条件適用
- `AGENTS.md` と `progress.json` の初期化
- detect -> plan(task DAG化) -> security preflight -> apply(トポロジカル順) -> gate
- 正常時は state/progress 更新
- 実行時間メトリクスを `.atrakta/metrics/runtime.json` に記録（`runtime metrics: start ...`）

### 主要フラグ

- `--interfaces`: 対象インターフェースを明示指定
- `--feature-id`: 長時間タスク用ID
- `--sync-level`: 同期制御レベル
- `--map-tokens`: repo map のトークン予算上限（契約値を一時上書き）
- `--map-refresh`: repo map の再生成間隔秒（契約値を一時上書き）
- 環境変数:
  - `ATRAKTA_INTERFACES`: `--interfaces` の代替
  - `ATRAKTA_TRIGGER_INTERFACE`: wrapper/hook からの自動解決ヒント
  - `ATRAKTA_TRIGGER_SOURCE`: 起動ソース識別（`wrapper` / `hook`）
  - `ATRAKTA_NONINTERACTIVE`: `1` の場合は入力待ちしない
  - `ATRAKTA_STALE_INTERFACE_DAYS`: stale提案の閾値日数（既定30）
  - `ATRAKTA_GC_DISABLE`: `1` で自動GC起動を停止
  - `ATRAKTA_GC_TMP_MAX_BYTES`: `.tmp` 自動GCの閾値（既定2GiB）
  - `ATRAKTA_GC_TMP_RETENTION_DAYS`: `.tmp` 優先削除の保持日数（既定7日）
  - `ATRAKTA_GC_AUTO_MIN_INTERVAL_MIN`: 自動GC最小間隔分（既定60分）
  - `ATRAKTA_TASK_CATEGORY`: routing category指定

### インターフェース自動解決順

1. 明示指定（`--interfaces` / `ATRAKTA_INTERFACES`）
2. トリガー指定（`ATRAKTA_TRIGGER_INTERFACE`）
3. 前回成功の単一ターゲット（`.atrakta/runtime/auto-state.v1.json`）
4. detectによる観測（アンカー/managed状態）

- どれでも解決できず `reason=unknown` の場合、`start` は `needs_input` を返す（暗黙デフォルトなし）。

## `doctor`

- 整合性診断と復旧アクション提示
- `--sync-proposal`: AGENTS由来の提案表示
- `--apply-sync`: 提案の承認適用
- 追加の自己修復提案:
  - `ide-autostart` 未導入 -> `atrakta ide-autostart install`
  - wrapper/PATH 不整合 -> `atrakta wrap install`
  - `.tmp` 閾値超過 -> `atrakta gc --scope tmp --apply`
  - `events.jsonl` 閾値超過 -> `atrakta gc --scope events`
- 実行時間メトリクスを `.atrakta/metrics/runtime.json` に記録（`runtime metrics: doctor ...`）

## `gc`

- 文脈維持前提の運用GC
- 方針:
  - `.tmp`: 閾値超過時のみ自動/手動削除対象
  - `events.jsonl`: proposal-only（自動変更しない）
- `--scope`: `tmp`, `events`（複数可）
- `--apply`: 対応scopeの削除を実行
- `--auto`: 閾値超過/間隔条件を使う自動モード
- すべての実行で dry-run 結果と適用結果を `.atrakta/runtime/gc-log.jsonl` に残す

## `wrap`

- 実行ラッパーをユーザbinに配置/削除
- fast path 条件一致時は `start` を省略

## `hook`

- シェルのディレクトリ移動時に `start` を起動するフックを導入/削除
- Hook 実行時 `start` は非対話モード（`ATRAKTA_NONINTERACTIVE=1`）

## `ide-autostart`

- VSCode互換IDE向けの自動起動タスク管理
- `install`: `.vscode/tasks.json` に `runOn=folderOpen` の managed タスクを追加（冪等）
- `uninstall`: managed タスクのみ削除
- `status`: 導入状態をJSONで表示

## `migrate check`

- `events.jsonl` の `schema_version` 整合を検証
- 現行要件: `schema_version = 2`

## `resume`

- `.atrakta/run-checkpoints/latest.json` を読み、前回実行条件で `start` を再実行
- `--interfaces` / `--feature-id` / `--sync-level` を指定した場合は checkpoint 値を上書き

## 終了コード（deferred outcome）

- `4`: `NEEDS_INPUT`
- `5`: `NEEDS_APPROVAL`
- `6`: `BLOCKED`
- `ATRAKTA_STATUS_JSON=1` で機械可読な outcome を標準出力に出す
