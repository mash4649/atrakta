# Interface Matrix

[English](./05_interface_matrix.md) | [日本語](../../ja/02_仕様/05_インターフェース対応表.md)

## Native Projection Targets

- `cursor`
  - `.cursor/AGENTS.md`
  - `.cursor/rules/00-atrakta.mdc` (default `cursor-rule` optional template)
- `claude_code`
  - `CLAUDE.md`
  - `.claude/settings.json`
  - `.claude/mcp.json`
  - `.claude/agents/atrakta.md`
- `codex_cli`
  - `AGENTS.md` (managed-mode aware append/include/generate flow)
  - `.codex/config.toml`
- `vscode` -> `.vscode/AGENTS.md`
- `windsurf` -> `.windsurf/AGENTS.md`
- `trae` -> `.trae/AGENTS.md`
- `antigravity` -> `.antigravity/AGENTS.md`
- `github_copilot` -> `.vscode/AGENTS.md`

## CLI Wrapper-only Interfaces

- `aider`
- `gemini_cli`
- `opencode`

Wrapper-only interfaces use runtime/wrapper synchronization and do not require dedicated native projection files.

## Antigravity Workflow/Rules

- Antigravity currently has native AGENTS projection only (`.antigravity/AGENTS.md`).
- Workflow/rules extensions are projected through extension outputs:
  - `.extensions/workflows/*.md`
  - `.extensions/hooks/workflow.before_start.md`
  - `.extensions/hooks/workflow.after_apply.md`
- Consumers wire these files explicitly; Atrakta does not silently inject unsupported native workflow/rule files.

## Extension Projection Policy

- `extensions.mcp` -> `.extensions/mcp/*.md`
- `extensions.plugins` -> `.extensions/plugins/*.md`
- `extensions.skills` -> `.extensions/skills/*.md`
- `extensions.workflows` -> `.extensions/workflows/*.md`
- `extensions.hooks` -> `.extensions/hooks/*.md`

When no interface-native extension projection exists, Atrakta generates deterministic fallback/link markdown and records it in `.atrakta/extensions/manifest.json`.

## Unsupported Policy

- no silent ignore
- emit warnings with field path and reason (for example `unsupported_extension_projection`)
- strict runtime checks can promote drift/integration warnings to blocking outcomes
- reverse sync stays proposal-only for extension objects (`mcp/plugins/skills/workflows/hooks`)

## Optional Templates

- `contract-json`: `<projectionDir>/CONTRACT.json`
- `atrakta-link`: `<projectionDir>/.atrakta-link`
- `cursor-rule`: `<projectionDir>/rules/00-atrakta.mdc` (Cursor)

Optional template settings beyond `max_per_interface` (default `3`) are rejected.
