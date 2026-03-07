# Example 2: Cursor Workflow

Cursor IDE で Atrakta を使って、再現可能な AI セッションを運用する手順です。

## 前提条件

- Cursor IDE インストール済み
- Atrakta インストール済み（[Example 1](./01_basic_init.md) 参照）

---

## Step 1: プロジェクトを初期化

```bash
mkdir cursor-project
cd cursor-project
atrakta init --interfaces cursor
```

---

## Step 2: Cursor でプロジェクトを開く

Cursor でこのフォルダを開きます。

`.atrakta/` と `AGENTS.md` が作成されていることを確認してください。

---

## Step 3: セッションを開始

```bash
atrakta start --interfaces cursor
```

これにより:

- 現在の状態がスナップショットされる
- イベントログの記録が始まる
- Cursor の AI 操作がトラッキングされる

---

## Step 4: Cursor で AI 作業を行う

通常通り Cursor を使います。

- コード生成
- リファクタリング
- ドキュメント生成

Atrakta はバックグラウンドで AI の操作をすべてログに記録します。

---

## Step 5: セッションを再開する

翌日や別のマシンで作業を再開する場合:

```bash
atrakta resume
```

Cursor は前回の AI セッションの状態から続きを始めます。

---

## Step 6: イベントログを確認する

AI が何をしたかを確認:

```bash
cat .atrakta/events.jsonl
```

各エントリには以下が含まれます:

- タイムスタンプ
- 操作の種類
- 対象ファイル
- 成否

---

## Step 7: 状態を診断する

問題が起きた場合:

```bash
atrakta doctor
```

コントラクト違反、ログの破損、状態の不整合を検出します。

---

## よくあるパターン

### 毎日の作業開始

```bash
cd my-project
atrakta resume        # 前回の続きから
# → Cursor を開いて作業
```

### 新しいタスクを始める

```bash
atrakta start --interfaces cursor
# → Cursor で作業
```

### 作業を中断する

特別な操作は不要です。次回 `atrakta resume` で再開できます。

---

## 次のステップ

- CLI ツールを使う場合 → [03_cli_workflow.md](./03_cli_workflow.md)
- 複数ツールを切り替える場合 → [04_tool_switching.md](./04_tool_switching.md)
