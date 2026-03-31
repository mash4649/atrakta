# アダプタ呼び出し契約

ラッパー／フックが Atrakta コアをどう呼び出すかを定義します。

## スコープ

- アダプタは薄い呼び出し層です。
- コアの実行プリミティブは `atrakta run` です。
- アダプタは内部のコマンド合成に依存してはなりません。
- v0 のアダプタはリフレッシュ後の run 契約の挙動を対象とし、`v0.14.1` のパリティは目指しません。
- アダプタはサーフェス間で同一 UX ではなく、意味論的な移植性を対象とします。

## 呼び出し

必須:

- `atrakta run` を呼び出す

任意フラグ:

- `--project-root <dir>`
- `--interface <id>`
- `--non-interactive`
- `--json`
- `--apply`
- `--approve`

アダプタ用環境変数:

- `ATRAKTA_TRIGGER_INTERFACE`
- `ATRAKTA_NONINTERACTIVE`

解決順:

1. 明示的な CLI フラグ
2. アダプタの環境変数
3. ランタイム検出

## 終了コードの扱い

- `0`: 成功
- `1`: ランタイム／契約エラー
- `2`: `NEEDS_INPUT`（アダプタは明示入力を求めて再試行）
- `3`: `NEEDS_APPROVAL`（アダプタは承認を集め、`--approve` 付きで再試行）

推奨アダプタループ:

1. `atrakta run --json` を実行
2. 終了 `2` なら不足入力を集めて再試行
3. 終了 `3` なら承認を集め、`--approve` 付きで再試行
4. `0` または回復不能な `1` で停止

## JSON 契約

`--json` 使用時、アダプタは次をパースする必要があります:

- `schemas/operations/run-output.schema.json`

出力に含まれうるもの:

- `planned_mutations[]`
- `applied_mutations[]`
- `required_inputs[]`（任意）
- `approval_scope`（任意）
- `portability`
- `resolved_projection_targets[]`
- `degraded_surfaces[]`
- `missing_projection_targets[]`
- `portability_status`
- `portability_reason`

変更オブジェクトの契約:

- `schemas/operations/mutation-proposal.schema.json`
- `schemas/operations/mutation-decision-envelope.schema.json`

## 非目標

- アダプタは canonical／state を直接書き込まない。
- アダプタはリゾルバ内部を直接呼ばない。
- アダプタは同一のプロンプトや対話フローを保証しない。
