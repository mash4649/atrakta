# MCP サーバー統合

MCP 統合は Atrakta の IDE ネイティブなアダプタ面です。
コアの実行プリミティブではなく、拡張クラスの統合として扱います。

## 役割

- MCP サーバー経由で診断と検査を提供する
- 拡張順序の中でアダプタ能力を持つ統合として参加する
- canonical state を read-only に保つ

## 契約

- `schema_version`: `mcp-server.v1`
- `kind`: `mcp`
- `transport.type`: `stdio` または `http`
- `transport.command`: 起動コマンド必須
- `capabilities`: 以下のいずれか 1 つ以上
  - `adapter`
  - `diagnostics`
  - `projection`
  - `inspection`

## 安全性

- MCP サーバーは core contract を変更してはならない。
- MCP サーバーは canonical state を直接書き込んではならない。
- MCP サーバーは diagnostics と projection inspection に限定してよい。

## `.mcp.json`

MCP を使うプロジェクトは `.mcp.json` または `mcp.json` を宣言できる。
Atrakta はそれらのアセットを検出し、ランタイム検出時に `mcp` インターフェースを解決する。

## アダプタ呼び出しとの関係

MCP は `atrakta start` ではなく `atrakta run` を呼ぶべきです。
[アダプタ呼び出し契約](adapter-invocation.md) を参照してください。
