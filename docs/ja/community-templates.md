# コミュニティテンプレート

これはコミュニティ導入のための初期設定テンプレートです。
各テンプレートは、Atrakta を安全な proposal-first の姿勢に保ちます。

## 1. Go サービス

Go のサービスや CLI が中心のリポジトリ向けです。

- 推奨アセット:
  - `go.mod`
  - `cmd/`
  - `internal/`
  - `tests/`
- Atrakta の注力点:
  - `run`
  - `inspect`
  - `preview`
  - `simulate`
- 既定のガイダンス:
  - generated/state/audit のパスは managed scope に保つ
  - ソースへの書き込みは明示承認があるまで proposal-only にする

## 2. Python アプリ

Python の自動化やアプリ本体が中心のリポジトリ向けです。

- 推奨アセット:
  - `pyproject.toml`
  - `src/`
  - `tests/`
  - `requirements.txt`
- Atrakta の注力点:
  - `run`
  - `onboard`
  - `mutate inspect`
  - `mutate propose`
- 既定のガイダンス:
  - パッケージ管理とテストの起点を検出する
  - 依存関係の変更は明示的に扱う

## 3. TypeScript Web アプリ

Web アプリや full-stack TypeScript プロジェクト向けです。

- 推奨アセット:
  - `package.json`
  - `src/`
  - `app/`
  - `public/`
- Atrakta の注力点:
  - `run`
  - `preview`
  - `simulate`
  - `projection render`
- 既定のガイダンス:
  - workspace ツールと project scripts を検出する
  - framework 設定の編集は managed scope に保つ

## 共通の安全ルール

- テンプレートから canonical ファイルを直接書き込まない。
- 最初の変更は proposal-first を維持する。
- 通常の `atrakta run` 実行モデルを維持する。
- 最初の proposal では最小限のファイル集合を優先する。
