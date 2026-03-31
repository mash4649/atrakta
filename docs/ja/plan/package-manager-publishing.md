# パッケージマネージャー公開

Atrakta は Go バイナリとして配布します。パッケージマネージャー
公開は、**別の実装**ではなく、そのバイナリを届けるための
配布ラッパーです。

## 目的

利用者が普段使うエコシステムから、次のものを取得しやすくします。

- npm
- pip
- cargo

パッケージマネージャー側の責務は次の通りです。

- バージョン付きの Atrakta バイナリを導入する
- CLI の挙動を公開済みバイナリと一致させる
- 実体の取得元は GitHub Releases のアセットに寄せる
- Go 本体モジュールに新しい実行時依存を持ち込まない

## 公開モデル

パッケージマネージャーは、輸送と起動の層としてのみ使います。

- 正規の配布物は `goreleaser` と GitHub Releases のプラットフォーム別バイナリです。
- package metadata は最小限に保ち、リリースと同じ版で管理します。
- 更新はタグ付きバイナリと同じ semantic version に合わせます。
- ダウンロード時には checksum と provenance の検証を維持します。

## エコシステム別の考え方

### npm

インストール時に現在のプラットフォームに合う binary を解決する、
薄い wrapper package を優先します。

### pip

tool entrypoint として binary を配置する command-line package を
優先します。

### cargo

Go コードを再ビルドするのではなく、公開済み binary を起動する
wrapper crate を優先します。

## 安全ルール

- package payload に source code を載せない。
- 通常インストールで release binary を迂回しない。
- Go モジュールに ecosystem 専用の runtime library を追加しない。
- package metadata は小さく、決定的で、監査しやすく保つ。

## 現在の状態

パッケージマネージャー公開は Phase 4 の配布タスクです。現時点では
目標形を文書化しておき、実装は release artifact と installer の
流れが安定してから行います。
