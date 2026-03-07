# Interface Matrix

[English](./05_interface_matrix.md) | [日本語](../../ja/02_仕様/05_インターフェース対応表.md)

## Editor

- `vscode` -> `.vscode/AGENTS.md`
- `cursor` -> `.cursor/AGENTS.md`
- `windsurf` -> `.windsurf/AGENTS.md`
- `trae` -> `.trae/AGENTS.md`
- `antigravity` -> `.antigravity/AGENTS.md`
- `github_copilot` -> `.vscode/AGENTS.md`

## CLI

- `aider`
- `codex_cli`
- `gemini_cli`
- `claude_code`
- `opencode`

CLI interfaces usually do not require dedicated projection files; wrapper-triggered sync is used.

## Optional Templates

- `contract-json`: `<projectionDir>/CONTRACT.json`
- `atrakta-link`: `<projectionDir>/.atrakta-link`

Optional template settings beyond `max_per_interface` (default `3`) are rejected.
