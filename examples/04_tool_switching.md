# Example 4: Tool Switching

Atrakta の最大の特徴: ツールを切り替えても開発状態が維持されます。

## シナリオ

機能開発を以下の流れで進めるケースです:

1. Cursor IDE で設計・コーディング
2. Claude Code CLI でバッチ処理
3. 再び Cursor で統合・確認

ツールが変わっても、状態・進捗・タスクグラフはすべて引き継がれます。

---

## Step 1: Cursor でセッションを開始

```bash
mkdir switch-demo
cd switch-demo
atrakta init --interfaces cursor
atrakta start --interfaces cursor
```

Cursor で作業します（コード生成、リファクタリングなど）。

---

## Step 2: Claude Code に切り替える

Cursor を閉じて、CLI に切り替えます:

```bash
atrakta switch --interfaces claude
atrakta resume
```

`resume` により:

- 前回の Cursor セッションの状態が読み込まれる
- 未完了タスクが引き継がれる
- イベントログが継続される

---

## Step 3: CLI で作業を続ける

```bash
claude "前回の続きから実装を進めて"
```

Atrakta が前回の状態を CLI に渡すため、AI は文脈を失わずに作業できます。

---

## Step 4: 再び Cursor に戻る

```bash
atrakta switch --interfaces cursor
atrakta resume
```

Cursor が開いた時点で、CLI での作業結果が状態に反映されています。

---

## 状態の確認

切り替えのたびに状態を確認できます:

```bash
# 現在の状態
cat .atrakta/state.json

# 全イベント履歴（どのツールで何をしたか）
cat .atrakta/events.jsonl

# タスクの完了状況
cat .atrakta/progress.json
```

---

## Atrakta なしとの比較

| 操作 | Atrakta なし | Atrakta あり |
|---|---|---|
| ツール切り替え | 状態をコピペで引き継ぐ | `atrakta switch` で自動 |
| セッション再開 | AI に再度説明が必要 | `atrakta resume` で即時 |
| 作業履歴 | なし | `events.jsonl` に全記録 |
| 失敗時の復帰 | 手動で状態を確認 | `atrakta doctor` で診断 |

---

## 対応インターフェース一覧

```bash
atrakta switch --interfaces cursor    # Cursor IDE
atrakta switch --interfaces claude    # Claude Code
atrakta switch --interfaces codex     # Codex CLI
atrakta switch --interfaces vscode    # VS Code
```

---

## 次のステップ

- 詳細な CLI 仕様 → [docs/en/02_spec/01_cli_spec.md](../docs/en/02_spec/01_cli_spec.md)
- 日常運用ガイド → [docs/en/03_operations/02_daily_operations.md](../docs/en/03_operations/02_daily_operations.md)
