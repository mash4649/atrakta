
# Atrakta — AI Coding Runtime

> **AI coding without interface lock‑in**

Stable development workflows across:

Cursor · Claude Code · Codex CLI · VS Code · aider · future AI tools

Atrakta is a lightweight runtime layer that makes AI coding:

safe · reproducible · resumable · observable

---

## 🚀 What is Atrakta

AI coding tools evolve extremely fast.

Editors change.
CLIs change.
Agents change.

But your **development workflow should remain stable.**

Atrakta provides a runtime layer between **AI tools** and your **project state**.

Think of it as:

**Git for AI coding sessions.**

---

## 🎥 Demo

Cursor → Claude CLI → VS Code

Same workflow. Same development state.

![Demo](docs/images/demo.gif)

---

## 🧠 Architecture

![Architecture](docs/images/atrakta_architecture.svg)

Layers:

1. AI Interfaces
2. Atrakta Runtime
3. Project Operational State

---

## ⚡ Key Capabilities

### Interface‑independent development

Switch tools anytime without breaking workflow.

### Deterministic pipeline

Detect → Plan → Apply

### Resumable sessions

```
atrakta resume
```

### Observability

```
.atrakta/events.jsonl
```

Append‑only event log for AI actions.

---

## 🔍 Ecosystem Position

| Category | Examples | Atrakta |
|---|---|---|
AI editor | Cursor | complements |
AI CLI | Claude Code | complements |
AI workflow | LangChain | adjacent |
AI agent | CrewAI | adjacent |
AI coding runtime | — | **Atrakta** |

---

## ⚡ Quick Start

macOS / Linux

```
curl -fsSL https://raw.githubusercontent.com/afwm/Atrakta/main/scripts/install.sh | bash
atrakta init --interfaces cursor
```

---

## 📚 Docs

English: docs/en/README.md  
Japanese: docs/ja/README.md

---

## 🤝 Contributing

Issues and PRs welcome.

If this project resonates with you, please ⭐ star the repository.

---

## License

MIT License
