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
| **init** | wrap install → hook install → ide-autostart install → start。初回セットアップの単一エントリポイント。 | `onboard`（提案生成）と `accept`（永続化）のみ。wrap／hook／ide-autostart の実行なし。 | **init** コマンドの追加。内部で wrap install、hook install、ide-autostart install を呼び、最後に start 相当を実行。`--mode greenfield|brownfield`、`--no-overwrite`、`--no-hook` 等のフラグを 0.14.1 と整合。 |
| **start** | 契約ロード → Fast Gate 判定 → イベントチェーン検証 → インターフェース解決 → detect → plan → apply → gate → state／progress／events 更新。 | `inspect`／`preview`／`simulate` はパイプライン実行とバンドル出力のみ。永続 state 更新や「セッション開始」としての一連の流れはなし。 | **start** コマンドの追加。contract ロード、Fast Path 判定、detect→plan→apply→gate の実行、`state.json`／`progress.json`／`events.jsonl`（または v0 監査ストリーム）への反映を一括で行う。 |
| **resume** | 前回の feature-id やインターフェースを引き継いで start 相当を再実行。 | なし。 | **resume** コマンドの追加。`.atrakta/runtime/auto-state.v1.json` 等から前回状態を読み、start に渡す。 |

**結論**: 「初回セットアップ」「セッション開始」「セッション再開」の 3 点がハーネスの入口。v0 は検査・提案・受け入れまではあるが、**init／start／resume の統一エントリポイント**が無い。

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
| **projection render** | 契約・正規状態から、インターフェース別に `agents_md` 等を**ディスクに書き出す**。 | `check_projection_eligibility` リゾルバで「投影可否」は判定しているが、**実際に `agents_md`（例: `AGENTS.md`）を生成して書き込む**処理はない。 | **projection** サブコマンド（render／status／repair）。canonical と contract から各インターフェース用の投影を生成し、指定パス（例: リポジトリルートの `AGENTS.md`）に書き出す。 |
| **projection status / repair** | 現在の `agents_md` と期待値の差分検出、修復提案・適用。 | なし。 | parity との連携も含め、status で drift 検出、repair で上書きまたはマージ。 |

**結論**: v0 は「投影モデル」と「投影可否」を契約で持つが、**プロジェクションを実際にレンダリングしてファイルに反映する層**が無い。ここがないと「どの AI ツールから見ても同じ `agents_md`」を保証できない。

---

## 4. イベントストリームと監査の互換

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **events.jsonl** | append-only、`schema_version=2`、`prev_hash` チェーン。`detect`／`repo_map`／`routing_decision`／`step`／`apply`／`projection_rendered` 等のイベント種別を定義。 | `audit append`／`audit verify` で監査ログはあるが、0.14.1 の events.jsonl と同じスキーマ・同じイベント種別ではない。 | どちらか: (A) v0 が **events.jsonl のスキーマと種別を採用**し、start／apply 等で同じ形式で追記する。または (B) v0 監査ストリームと events.jsonl の**双方向マッピング**を定義し、0.14.1 ツール（doctor、migrate 等）が読めるようにする。 |
| **チェーン検証** | start 時に `VerifyChainCached`。キャッシュ不一致時は full verify。 | audit verify で整合性検証はある。 | start 前に「イベントチェーン検証」を必ず行うようにし、0.14.1 と同等の保証にする。 |

**結論**: ハーネスは「何が起きたか」を append-only で残し、デバッグ・監査・再現に使う。v0 の監査は方向性は同じだが、**0.14.1 の events.jsonl とのスキーマ・種別の互換**（または明確なマッピング）が必要。

---

## 5. ランタイム状態（state / progress / task-graph / Fast Path）

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **state.json** | 現在の managed パス・フィンガープリント、projection の最終レンダ状態、integration 結果。 | `accept` で `state/onboarding-state.json` を書くが、0.14.1 の state と同じ形状ではない。 | start／apply 後に **state.json** を 0.14.1 と互換（または v0 スキーマで明確に版管理）で更新する。 |
| **progress.json** | タスク完了トラッキング。 | なし。 | **progress.json** の更新を plan／apply 結果に基づいて行う。 |
| **task-graph.json** | 保留・完了タスクの DAG。 | なし。 | plan の出力を **task-graph.json** として永続化。resume 時に参照。 |
| **Fast Path スナップショット** | `.atrakta/runtime/meta.v2.json` の `start_fast_v2`。contract_hash／workspace_stamp／interface set／feature_id／config key が一致すれば strict パスをスキップ。 | なし。 | start の入口で **Fast Path 判定**を実装。一致時はイベントチェーン検証とチェックポイント記録のみで即返す。不一致時のみ detect→plan→apply→gate を実行。 |
| **auto-state.v1.json** | 前回成功したインターフェース等。resume のヒント。 | なし。 | **resume** で参照するため、start 成功時に **runtime/auto-state.v1.json** を更新。 |

**結論**: 「今の状態」「進捗」「タスクグラフ」「前回実行のキャッシュ」が無いと、**resume** や **Fast Path** が成立しない。v0 は accept で一部 state を書くが、0.14.1 と同等のランタイム状態の更新が start／apply 経路に必要。

---

## 6. インターフェース解決

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **解決順** | 1) 明示（`--interfaces`／環境変数）2) トリガー（wrapper／hook 由来）3) 前回成功（auto-state）4) detect（アンカー等）5) プロンプト。未解決かつ reason=unknown なら `needs_input`。 | バンドル入力にインターフェースを渡す前提で、**「どのインターフェースで動いているか」をランタイムで解決する**流れはない。 | start の前に **インターフェース解決**を実装。明示→トリガー→auto-state→detect の順で決定し、未解決時は `NEEDS_INPUT` と next action を返す。 |

**結論**: ハーネスは「今どの AI ツール（cursor／codex／等）の文脈か」を決めてから detect／plan する。v0 は **インターフェース解決のランタイム仕様と実装**が無い。

---

## 7. GC と migrate

| 要素 | 0.14.1 | v0 現状 | 必要対応 |
|------|--------|---------|----------|
| **gc** | `gc --scope tmp|events [--apply] [--auto]`。.tmp の肥大化、events の古い行の整理。 | なし。 | **gc** サブコマンド。少なくとも `tmp` と `events`（または v0 監査ログ）のスコープを用意し、`--apply` で実際に削除。 |
| **migrate check** | スキーマ版や events.schema_version をチェックし、移行が必要なら案内。 | なし。 | **migrate check**。v0 の契約・state・監査の版と、0.14.1 からの移行パスを定義し、必要なら移行スクリプト／手順を提示。 |

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
| **contract.json** | v=1、interfaces／boundary／tools／token_budget／routing／context／security／edit_safety／policies／parity／extensions。 | ルートの `.atrakta/contract.json` は機械契約として保持し、`run` がロードして detect／plan／apply に渡す前提。 | v0 の **run** 実装で、0.14.1 と同一（またはサブセット）の contract を読み、リゾルバや projection に渡す。必要なら v0 用の契約スキーマを 0.14.1 のサブセットとして定義し、互換を明示する。 |

**結論**: 契約は既に近い形で存在する。足りないのは「**start がその契約を読み、全フローに一貫して使う**」部分。

---

## 優先度まとめ（ハーネスとして動かすために）

1. **必須（ハーネスの入口と状態）**  
   - **init**（wrap + hook + ide-autostart + run の統合）  
   - **start**（contract ロード → Fast Path → インターフェース解決 → detect→plan→apply→gate → state／events 更新）  
   - **resume**（前回状態に基づく start）  
   - **wrap**（install／run）と **hook**（install／repair）  
   - **projection**（render／status／repair）  
   - **state.json／progress.json／task-graph.json** と **Fast Path スナップショット**の更新  
   - **イベントストリーム**（events.jsonl 互換または明確なマッピング）

2. **重要（運用・一貫性）**  
   - **gc**、**migrate check**  
   - **インターフェース解決**の明文化と実装  
   - **ide-autostart**

3. **あると良い（0.14.1 パリティ）**  
   - **import**／**capability**／**recipe**／**memory**／**exploration**  
   - **バイナリ配布とインストール手順**

---

## 次のステップ案

1. **run の統合**  
   - 既存の `inspect`／`preview`／`simulate` のパイプラインを、「契約ロード → Fast Gate → インターフェース解決 → 既存リゾルバ実行 → state／events 書き出し」の一連の **run** コマンドに組み込む。
2. **wrap install／run の実装**  
   - ユーザーが `atrakta wrap run --interface cursor --real /path/to/cursor -- ...` でツールを起動できるようにする。  
3. **projection render の実装**  
   - canonical と contract から `agents_md`（と必要なら他ファイル）を生成し、`projection render` で書き出す。  
4. **state／progress／task-graph／events のスキーマ確定**  
   - 0.14.1 互換か、v0 独自だが版付きで定義かを決め、start／apply の出力をそれに合わせる。  
5. **init と resume**  
   - start ができた後に、init（wrap+hook+ide-autostart+start）と resume（auto-state から start）を追加。

この順で進めると、v0 が「ハーネスとしての役割」を 0.14.1 から引き継ぐための最小セットを満たしつつ、契約・リゾルバ駆動の設計を維持できる。
