# 08. GitHub Issue登録（Parity / Extension / Brownfield）

この手順は、実装バックログを GitHub Issue として一括登録するための運用です。

## 入力ファイル

- `.github/issue_drafts/parity_extension_brownfield.json`
- `.github/issue_drafts/parity_extension_brownfield_epic_map.json`
- `scripts/dev/create_parity_issue_pack.sh`

## スクリプトが行うこと

1. 28件の Issue 定義（ラベル、マイルストーン、依存）を読み込む
2. Epic/Story 対応と追加タスクを本文に反映する
3. ラベル・マイルストーンの不足分を作成する（skip 指定可）
4. Issue を順次作成する
5. 作成後に `Depends on:` コメントで依存関係を追記する

## 事前確認（dry-run）

```bash
./scripts/dev/create_parity_issue_pack.sh \
  --repo <owner/repo> \
  --dry-run
```

## 全件登録

```bash
./scripts/dev/create_parity_issue_pack.sh \
  --repo <owner/repo>
```

## 一部だけ登録

```bash
./scripts/dev/create_parity_issue_pack.sh \
  --repo <owner/repo> \
  --start-id 9 \
  --limit 8
```

## 主なオプション

- `--skip-label-setup`: ラベル作成をスキップ
- `--skip-milestone-setup`: マイルストーン作成をスキップ
- `--draft <path>`: issue定義JSONを差し替え
- `--epic-map <path>`: epic/story対応JSONを差し替え

## 出力

作成結果の Draft ID -> GitHub Issue番号 マップは以下に保存されます。

- `.github/issue_drafts/out/issue-map-YYYYMMDD-HHMMSS.json`

## GitHub Project へ投入する

バックログProjectを作成または再利用し、Issueを投入します。

```bash
./scripts/dev/populate_parity_project.sh \
  --repo <owner/repo> \
  --owner <owner> \
  --project-title "Atrakta Parity / Extension / Brownfield Backlog" \
  --issue-map .github/issue_drafts/out/issue-map-YYYYMMDD-HHMMSS.json
```

dry-run:

```bash
./scripts/dev/populate_parity_project.sh \
  --repo <owner/repo> \
  --dry-run
```

## Projectフィールドを同期する

Issueラベル `priority:P0/P1/P2` を Project の `Priority` フィールドへ同期します。

```bash
./scripts/dev/sync_parity_project_fields.sh \
  --owner <owner> \
  --project-number <n>
```

dry-run:

```bash
./scripts/dev/sync_parity_project_fields.sh \
  --owner <owner> \
  --project-number <n> \
  --dry-run
```

## 推奨ビュー（Milestone別）

Project UI の保存ビューに以下の filter を設定してください。

- `milestone:"Milestone 1: Schema & Docs Foundation"`
- `milestone:"Milestone 2: Core Projection"`
- `milestone:"Milestone 3: Brownfield & Doctor"`
- `milestone:"Milestone 4: Extensions & Runtime"`
- `milestone:"Milestone 5: Operations & Quality"`
