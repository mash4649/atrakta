# Example 3: CLI Workflow

Claude Code / Codex CLI などの AI CLI ツールと Atrakta を組み合わせる手順です。

## 前提条件

- Claude Code または Codex CLI インストール済み
- Atrakta インストール済み（[Example 1](./01_basic_init.md) 参照）

---

## Step 1: プロジェクトを初期化

```bash
mkdir cli-project
cd cli-project
atrakta init --interfaces claude
```

`--interfaces` に指定できる値:

| 値 | 対象ツール |
|---|---|
| `cursor` | Cursor IDE |
| `claude` | Claude Code |
| `codex` | Codex CLI |
| `vscode` | VS Code |

---

## Step 2: セッションを開始

```bash
atrakta start --interfaces claude
```

---

## Step 3: CLI で AI 作業を行う

```bash
# Claude Code の例
claude "src/main.go に fibonacci 関数を実装して"
```

Atrakta は CLI の操作をラップしてイベントログに記録します。

---

## Step 4: 進捗を確認する

```bash
cat .atrakta/progress.json
```

完了・未完了のタスクが確認できます。

タスクグラフを確認:

```bash
cat .atrakta/task-graph.json
```

---

## Step 5: セッションを再開する

```bash
atrakta resume
```

AI は前回の状態から続きを実行します。

---

## Step 6: セーフティコントラクトを確認する

```bash
cat .atrakta/contract.json
```

AI が変更してよいファイル・してはいけないファイルのルールが定義されています。

デフォルトで以下が保護されます:

- `*.env` ファイル
- 本番設定ファイル
- 明示的に `managed-only` 指定されたファイル

---

## よくあるパターン

### 一連の実装タスクを CLI で自動化

```bash
atrakta start --interfaces claude

claude "以下を順番に実装して:
1. src/user.go にユーザーモデルを作成
2. src/user_test.go にテストを追加
3. README.md の API セクションを更新"
```

Atrakta がタスクグラフを管理し、完了・失敗を追跡します。

### 失敗したタスクを再試行

```bash
atrakta resume
# → 失敗したタスクから再開
```

---

## 次のステップ

- Cursor と CLI を切り替えたい場合 → [04_tool_switching.md](./04_tool_switching.md)
