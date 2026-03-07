# Example 1: Basic Initialization

Atrakta を新しいプロジェクトに導入する最短手順です。

## Step 1: インストール

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/mash4649/atrakta/main/scripts/build/install.sh | bash
```

インストール確認:

```bash
atrakta --version
```

**Windows**

[Releases](https://github.com/mash4649/atrakta/releases) から `atrakta_*_windows_amd64.zip` をダウンロードして展開し、`atrakta.exe` を PATH に追加してください。

詳細: [セットアップガイド](../docs/en/03_operations/01_setup.md)

---

## Step 2: プロジェクトを初期化

```bash
mkdir my-ai-project
cd my-ai-project
atrakta init --interfaces cursor
```

実行後に作成されるファイル:

```
my-ai-project/
├── AGENTS.md                   ← AI エージェント向けの指示書
└── .atrakta/
    ├── contract.json           ← 安全ルールと変更ポリシー
    ├── events.jsonl            ← 追記専用イベントログ
    ├── state.json              ← 現在のセッション状態
    ├── progress.json           ← タスク完了状況
    └── task-graph.json         ← タスクの依存グラフ
```

---

## Step 3: セッションを開始

```bash
atrakta start --interfaces cursor
```

これで AI セッションのトラッキングが始まります。

---

## Step 4: セッションを再開

作業を中断して後から再開する場合:

```bash
atrakta resume
```

前回の状態・進捗・タスクグラフがそのまま復元されます。

---

## Step 5: 状態を確認

```bash
atrakta doctor
```

現在のセッション状態、コントラクト、イベントログの整合性を診断します。

---

## 確認: イベントログを見る

AI が行ったすべての操作は `.atrakta/events.jsonl` に記録されています:

```bash
cat .atrakta/events.jsonl
```

---

## `atr` — 短縮エイリアス

インストール時に `atr` が自動で作成されます。`atrakta` と完全に同じです:

```bash
atr init --interfaces cursor
atr start --interfaces cursor
atr resume
atr doctor
```

## 次のステップ

- Cursor を使っている場合 → [02_cursor_workflow.md](./02_cursor_workflow.md)
- CLI (Claude Code / Codex) を使っている場合 → [03_cli_workflow.md](./03_cli_workflow.md)
- 複数ツールを切り替えたい場合 → [04_tool_switching.md](./04_tool_switching.md)
