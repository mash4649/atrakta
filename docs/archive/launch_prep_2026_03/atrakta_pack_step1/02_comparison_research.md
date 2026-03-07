# Comparison Research (Positioning + Table)

## Position in the ecosystem

| Category | Typical tools | Atrakta |
|---|---|---|
| AI editor | Cursor, VS Code extensions | complements (not competing as an editor) |
| AI CLI | Claude Code, Codex CLI, aider | complements (tool-agnostic) |
| AI workflow framework | LangChain, LangGraph | adjacent (coding ops vs general workflows) |
| AI agent frameworks | AutoGPT-like, CrewAI | adjacent (agents vs coding ops runtime) |
| **AI coding runtime** | (no clear standard yet) | **Atrakta** |

## Feature-level comparison (draft)

| Feature | Cursor | Claude Harness | LangChain/LangGraph | Atrakta |
|---|---:|---:|---:|---:|
| Interface independent (editor/CLI switch) | ❌ | ❌ | ❌ | ✅ |
| Focus: AI coding operations | ⚠️ | ⚠️ | ❌ | ✅ |
| Reproducible sessions / resumable runs | ❌ | ⚠️ | ⚠️ | ✅ |
| Append-only event log (auditable) | ❌ | ❌ | ❌ | ✅ |
| Safety contract / managed-only guard | ❌ | ❌ | ❌ | ✅ |
| Cross-tool state (same workflow everywhere) | ❌ | ❌ | ❌ | ✅ |
| Deterministic pipeline (Detect→Plan→Apply) | ❌ | ❌ | ⚠️ | ✅ |

## Differentiator (one-liner)

Atrakta is a **runtime layer for AI coding** that preserves a consistent, safe,
and reproducible development experience **across interfaces**, even as tools change.
