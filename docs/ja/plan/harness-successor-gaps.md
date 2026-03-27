# v0 をハーネス後継とする際のギャップ分析

v0 が Atrakta の後継として**ハーネス**（AI ツールとプロジェクト状態のあいだに立ち、セッションの再開・監査・安全を保つランタイム）の役割を担うために、現状不足している要素を整理する。
本文書はギャップ分析であり、現行の実装・契約は [実装状況](implementation-status.md) と
[実行契約（run）](../architecture/run-contract.md)（英語版: [run-contract.md](../../architecture/run-contract.md)）を優先して参照する。

参照: 0.14.1 の役割（README）  
「Atrakta is the runtime that preserves AI development state across tools, restarts, and handoffs.」

---

## 1. セッションライフサイクル（init / start / resume）

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **init** | wrap install → hook install → ide-autostart install → start。初回セットアップの単一エントリポイント。 | 最小版は実装済み。`start` 委譲 + `init.*` イベント発火まで導入済みだが、wrap／hook／ide-autostart の実体処理は未実装。 | wrap install、hook install、ide-autostart install を実装し、`init` を 0.14.1 相当の統合入口へ近づける。 |
| **start** | 契約ロード → Fast Gate 判定 → イベントチェーン検証 → インターフェース解決 → detect → plan → apply → gate → state／progress／events 更新。 | 実装済み。`run` と共通のエントリ解決、contract ロード／検証、事前監査検証、Fast Path、runtime state／handoff 更新に加え、P1-H-001 で diagnostics / output-envelope hardening を完了。 | 残課題は binding 駆動のインターフェース解決拡張、wrap／hook 連携、`init` との最終統合。 |
| **resume** | 前回の feature-id やインターフェースを引き継いで start 相当を再実行。 | 実装済み。`handoff.v1.json` と `auto-state.v1.json` のヒントを使って `start` 経路へ再投入できる。 | handoff artifact の表現力を拡張し、acceptance artifact や再開時判定の粒度を上げる。 |

**結論**: 入口 3 点のうち **start / resume は実装済み**。`init` は最小版が入ったが、wrap／hook／ide-autostart の統合が未完で、再開品質を上げる handoff artifact の深化も残っている。

---

## 2. ツール結合（wrap / hook）

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **wrap** | `wrap install`（PATH に atrakta を置く）、`wrap run --interface <id> --real <path> -- [args]`（実ツールを atrakta 経由で起動）。 | なし。 | **wrap** サブコマンド（install／uninstall／run）。run では「実バイナリを atrakta がラップして起動し、その前後で contract／state を考慮」する流れを実装。 |
| **hook** | `hook install|uninstall|status|repair`。IDE 等のサーフェスにフックを仕込み、保存時や起動時に atrakta を噛ませる。 | なし。 | **hook** サブコマンド。各 surface（cursor、vscode 等）向けのフック登録・解除・状態確認・修復。 |
| **ide-autostart** | `.vscode/tasks.json` 等で「ワークスペースを開いたら atrakta start を実行」等。 | なし。 | **ide-autostart**（install／uninstall／status）。エディタ起動時に atrakta が走るようにする。 |

**結論**: ハーネスは「ユーザーが使う AI ツールの前後に atrakta が入る」必要がある。そのため **wrap／hook／ide-autostart** が必須。v0 は現状どれも無い。

---

## 3. プロジェクション表面（`agents_md` 等の書き出し）

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **projection render** | 契約・正規状態から、インターフェース別に `agents_md` 等を**ディスクに書き出す**。 | 初期実装済み。`projection render` で `AGENTS.md` の deterministic 出力が可能。 | 複数ターゲット（`ide_rules` / `repo_docs` / `skill_bundle`）とインターフェース別テンプレートを拡張する。 |
| **projection status / repair** | 現在の `agents_md` と期待値の差分検出、修復提案・適用。 | 初期実装済み。`projection status` の drift 検出と `projection repair` の復旧が可能。 | parity 連携、マージ戦略、差分表示の強化を行う。 |

**結論**: v0 は projection の初期機能（render／status／repair）を持つ。次は **対象サーフェスの拡張** と **差分／修復品質の強化** が課題。

---

## 4. イベントストリームと監査

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **events.jsonl** | append-only、`schema_version=2`、`prev_hash` チェーン。`detect`／`repo_map`／`routing_decision`／`step`／`apply`／`projection_rendered` 等のイベント種別を定義。 | v0 では ADR-004 に基づく独自スキーマ（`schema_version=1`）の `run-events.jsonl` を audit ストア配下に追加し、start／resume／apply 経路から追記・検証している。P1-H-001 以降の診断系も run output 側で整合した。旧 `/.atrakta/events.jsonl` は read-only 扱い。 | 変換層は置かず、v0 の runtime event taxonomy を拡張して運用する。 |
| **チェーン検証** | start 時に `VerifyChainCached`。キャッシュ不一致時は full verify。 | audit verify で整合性検証はある。 | start 前に「イベントチェーン検証」を必ず行うようにし、0.14.1 と同等の保証にする。 |

**結論**: v0 は append-only の runtime イベントストリーム導入まで完了。旧 `events.jsonl` は履歴として残し、今後は **v0 の runtime event taxonomy を拡張**していく。

---

## 5. ランタイム状態（state / progress / task-graph / Fast Path）

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **state.json** | 現在の managed パス・フィンガープリント、projection の最終レンダ状態、integration 結果。 | `start/resume`（non-fast-path）で `.atrakta/state.json` を更新する基盤は導入済み。 | 0.14.1 互換が必要ならフィールド対応表と移行方針を確定する。 |
| **progress.json** | タスク完了トラッキング。 | `.atrakta/progress.json` の永続化は導入済み。 | progress 粒度（何を完了とみなすか）を明文化し、適用／再開時更新ルールを固定する。 |
| **task-graph.json** | 保留・完了タスクの DAG。 | `.atrakta/task-graph.json` の永続化は導入済み。 | DAG 意味論（ノード状態遷移、resume 再解釈）を明文化する。 |
| **Fast Path スナップショット** | `.atrakta/runtime/meta.v2.json` の `start_fast_v2`。contract_hash／workspace_stamp／interface set／feature_id／config key が一致すれば strict パスをスキップ。 | `.atrakta/runtime/start-fast.v1.json` で no-change short-circuit を実装済み。 | 0.14.1 との差分（キー構成、メタデータ）を整理し、必要なら v0 内部の判定基準を調整する。 |
| **auto-state.v1.json** | 前回成功したインターフェース等。resume のヒント。 | `.atrakta/runtime/auto-state.v1.json` の更新／参照は実装済み。 | 複数 interface 履歴や confidence を扱う場合の拡張スキーマを検討する。 |

**結論**: runtime state の最小セット（state／progress／task-graph／Fast Path／auto-state）は実装済み。残りは **互換方針と意味論の固定**。

---

## 6. インターフェース解決

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **解決順** | 1) 明示（`--interfaces`／環境変数）2) トリガー（wrapper／hook 由来）3) 前回成功（auto-state）4) detect（アンカー等）5) プロンプト。未解決かつ reason=unknown なら `needs_input`。 | 実装済み（明示→トリガー env→auto-state→detect、未解決時は `NEEDS_INPUT`）。 | wrapper／hook 起点の trigger 情報を binding 定義と連携し、検出の拡張性を高める。 |

**結論**: 基本の解決順は実装済み。残課題は **binding 駆動の拡張性と wrapper／hook 連携**。

---

## 7. GC と migrate

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **gc** | `gc --scope tmp|events [--apply] [--auto]`。.tmp の肥大化、events の古い行の整理。 | 初期版を実装済み。`tmp`／`events` スコープの dry-run／apply が可能。 | スコープ拡張と保持ポリシーの明文化、run-events との連携強化。 |
| **migrate check** | スキーマ版や events.schema_version をチェックし、移行が必要なら案内。 | 初期版を実装済み。contract／state／progress／task-graph／run-events をチェックする。 | 版管理と移行ガイダンスの粒度を上げる。 |

**結論**: 運用・アップグレードのために **gc** と **migrate check** が必要。

---

## 8. インポートパイプライン（任意だが推奨）

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **import repo** | 外部リポジトリをスキャンし、capability 候補として取り込む。 | なし。onboarding で「既存リポジトリ内」のアセット検出はある。 | ハーネス「必須」ではないが、他プロジェクトのスキル・レシピ・メモリを取り込むために **import repo**（と report／pulse）があると良い。 |
| **capability analyze** | 取り込んだ capability の分析・quarantine／promote。 | onboarding の `infer_capabilities` に近いが、永続レジストリと analyze コマンドはない。 | **capability analyze** と capability レジストリの永続化。 |
| **recipe / memory / exploration** | レシピ変換、メモリ昇格、探索カタログ。 | なし。 | 後続フェーズで **recipe convert**、**memory review**、**exploration catalog** を検討。 |

**結論**: ハーネスの中核（init／start／resume、wrap／hook、projection、state／events）より優先度は下がるが、0.14.1 と同程度の「外部資産の取り込み」を v0 でも扱うなら、**import と capability と recipe／memory／exploration** の順で足していく形が自然。

---

## 9. 配布とバイナリ

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **単一バイナリ** | `atrakta` として配布。 | `go run ./cmd/atrakta` 前提。 | **ビルド**で `atrakta` バイナリを生成し、リリース artifact として配布。 |
| **インストール** | curl スクリプト、Windows 用 ZIP。 | なし。 | **install スクリプト**（curl 等）、Windows 用の配布方法を用意。PATH に `atrakta` が入るようにする。 |

**結論**: ハーネスを「ユーザーがそのまま使う」ものにするには、**バイナリ配布とインストール手順**がいる。

---

## 10. 契約形状の互換

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **contract.json** | v=1、interfaces／boundary／tools／token_budget／routing／context／security／edit_safety／policies／parity／extensions。 | `.atrakta/contract.json` のロードと schema 検証は実装済み（normal path）。 | 0.14.1 との差分がある項目の対応表を整備し、必要なら変換ポリシーを追加する。 |

**結論**: contract ロード／検証は実装済み。残課題は **v0 固有の運用ポリシーの明文化**。

---

## 優先度まとめ（ハーネスとして動かすために）

1. **必須（ハーネス運用完成のため）**  
   - **init**（wrap + hook + ide-autostart + start の統合）  
   - **wrap**（install／run）と **hook**（install／repair）  
   - **run-events** のイベント種別拡張と検証範囲の整備

2. **重要（運用・一貫性）**  
   - **gc**、**migrate check**  
   - **インターフェース解決拡張**（binding 駆動、wrapper／hook 連携）  
   - **runtime state／contract の v0 運用ポリシー確定**

3. **あると良い（0.14.1 パリティ）**  
   - **import**／**capability**／**recipe**／**memory**／**exploration**  
   - **バイナリ配布とインストール手順**

---

## 次のステップ案

1. **wrap install／run の実装**  
   - ユーザーが `atrakta wrap run --interface cursor --real /path/to/cursor -- ...` でツールを起動できるようにする。  
2. **init の残り統合**  
   - `wrap + hook + ide-autostart + start` を 1 コマンドで実行できるよう、未実装の結合点を埋める。  
3. **projection の拡張**  
   - `ide_rules` / `repo_docs` / `skill_bundle` まで投影対象を広げ、インターフェース別出力を強化する。  
4. **run-events のスキーマ硬化**  
   - runtime event の種別を増やしつつ、`run-events` を v0 の正規ストリームとして固定する。  
5. **gc / migrate check のポリシー拡張**  
   - 初期版の運用保守と移行の最小セットを、保持ポリシーと移行ガイダンスまで広げる。

この順で進めると、v0 が「ハーネスとしての役割」を 0.14.1 から引き継ぐための最小セットを満たしつつ、契約・リゾルバ駆動の設計を維持できる。
