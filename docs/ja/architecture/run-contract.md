# 実行契約（`run`）

## 目的

`atrakta run` は単一の実行プリミティブです。
機械可読な契約は `.atrakta/contract.json` にあります。
サーフェス間の移植性は [サーフェス移植性](surface-portability.md) で別途定義します。

位置づけ:

- v0 は `v0.14.1` の互換レイヤではなく、リフレッシュ系のラインです。
- コマンド／データのパリティより、挙動として十分かを優先します。
- レガシー資産は、明示的な import／transform フローで後から変換される場合があります。

- 初回実行: オンボーディングフロー（`detect -> propose -> accept`）
- 2 回目以降: 通常フロー（`detect -> plan -> apply` 系）

その他のコマンド（`onboard`、`accept`、`inspect`、`preview`、`simulate`）は
実装の部品およびデバッグ用サーフェスです。運用上の契約は `run` です。

## 入力

任意:

- `--project-root <dir>`（デフォルト: カレントから検出）
- `--interface <id>`
- `--non-interactive`
- `--json`
- `--apply`
- `--approve`（非対話の書き込み経路用の明示承認トークン）

アダプタセット用の環境変数:

- `ATRAKTA_TRIGGER_INTERFACE`
- `ATRAKTA_NONINTERACTIVE`

優先順位:

- インターフェース: `--interface` > `ATRAKTA_TRIGGER_INTERFACE` > 検出 > `NEEDS_INPUT`
- 非対話: `--non-interactive` > `ATRAKTA_NONINTERACTIVE`

## 出力

終了コード:

- `0`: 成功（状態更新の有無は問わない）
- `1`: ランタイムまたは契約エラー
- `2`: `NEEDS_INPUT`（決定的に続行する入力が不足）
- `3`: `NEEDS_APPROVAL`（書き込み経路の前に明示承認が必要）

出力チャネル:

- デフォルトでは人間向け要約を標準出力へ
- `--json` では機械可読なエンベロープ（`schemas/operations/run-output.schema.json`）
- 変更の詳細は `planned_mutations[]` / `applied_mutations[]` として出力
- アダプタ用ヒントに `required_inputs[]` や `approval_scope` を含められる
- 移植性メタデータに `portability`、`resolved_projection_targets[]`、
  `degraded_surfaces[]`、`missing_projection_targets[]`、`portability_status`、
  `portability_reason` を含める

## 正規（canonical）の存在状態

`run` はルーティング前にプロジェクト状態を確認します:

- `canonical_present`
- `onboarding_complete`
- `partial_state`
- `corrupt_state`

現時点の最小限の「正規あり」条件:

- `.atrakta/canonical/policies/registry/index.json` が存在する
- `.atrakta/contract.json` が正規状態と並んで機械契約として存在する

追加の状態区分:

- `partial_state`: `.atrakta/state/onboarding-state.json` はあるが正規インデックスがない
- `corrupt_state`: canonical／state ディレクトリはあるが必須マーカーファイルが欠ける

`partial_state` / `corrupt_state` が検出された場合、`run` は診断理由付きで `1` を返します。
`run` は正規状態を黙って修復しません。

## フロー分岐

### 正規なし（オンボーディング経路）

1. detect: オンボーディング提案の生成
2. propose: 提案のレンダリング（テキスト／JSON）
3. 承認ゲート:
   - 非対話または拒否 => `NEEDS_APPROVAL`（`3`）
   - 承認 => `accept` 経路
4. accept 経路が canonical／state／audit に書き込む

書き込み保証:

- 明示承認の前に canonical／state／audit へは書き込まない
- オンボーディング経路でインターフェースを検出できない場合、`run` は `generic-cli` にフォールバックする

### 正規あり（通常経路）

1. `.atrakta/contract.json` から機械契約を読み込む
2. 正規状態を読み込む（`canonical/policies/registry/index.json` と任意のオンボーディング状態）
3. 現在のワークスペースサーフェスを検出する
4. 順序付きリゾルバパイプラインを実行する（`inspect` をベースラインとする）
5. 任意の apply 経路は、生成された管理対象ターゲット向けの管理付き変更計画を組み立てる:
   - `.atrakta/generated/repo-map.generated.json`
   - `.atrakta/generated/capabilities.generated.json`
   - `.atrakta/generated/guidance.generated.json`
6. 管理付き apply の前にサーフェス移植性を解決する
7. 劣化または非対応の移植性では提案のみにフォールバックし apply をスキップする
8. apply は明示承認時のみ実行する（`--approve` または対話承認）
9. state／audit の更新は、受諾／apply 成功の経路に限る

冪等性:

- 同一プロジェクトかつ同一の実効入力でアクション不要な場合、`run` は
  書き込みを行わず `0` を返す

## 遅延永続化ポリシー

永続化はイベント駆動のみです:

- オンボーディング `accept` 成功時
- 管理付き `apply` 成功時
- 機械契約のリフレッシュが発行される場合も、同様の accept／apply ゲートに従う

上記以外の経路では canonical／state に書き込まない。

監査ポリシー:

- 運用トレースのための監査追記は許可される
- 監査追記は canonical／state の変更とはみなさない
- 通常経路ではリプレイ／追跡のため `run_execute` 監査イベント（A2）を追記する

apply 時の状態更新:

- 成功した apply は `.atrakta/state/run-state.json` に書き込む

## アダプタ呼び出し契約

[アダプタ呼び出し契約](adapter-invocation.md) を参照してください。
