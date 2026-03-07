# Atrakta

> **AI coding without interface lock‑in**

Stable AI development across:

Cursor · Claude Code · Codex CLI · VS Code · aider · future AI tools

Atrakta is a lightweight **AI coding runtime** that keeps
development:

**safe · reproducible · resumable · observable**

------------------------------------------------------------------------

## 🎥 Demo

Switch tools without losing development state.

Cursor → Claude CLI → VS Code

Same workflow. Same state.

![Demo](docs/images/demo.gif)

------------------------------------------------------------------------

## ⭐ Why Atrakta?

AI coding tools evolve extremely fast.

But development workflows often break when tools change.

  Problem                What happens
  ---------------------- --------------------------------------
  Interface lock‑in      Workflow depends on one editor
  Session instability    AI behaves differently each run
  Debugging difficulty   Automation failures are opaque
  Risky edits            AI can introduce destructive changes

Atrakta adds a **runtime layer** between AI tools and your project.

Think of it as:

> **Git for AI coding sessions**

------------------------------------------------------------------------

## 🚀 Key Features

### Interface‑independent workflows

Switch between:

Cursor\
Claude CLI\
VS Code\
future tools

without changing development workflow.

------------------------------------------------------------------------

### Reproducible sessions

Resume exactly where the AI left off.

    atrakta resume

------------------------------------------------------------------------

### Deterministic pipeline

Atrakta executes a deterministic workflow:

    Detect → Plan → Apply

------------------------------------------------------------------------

### Full observability

Every AI action is logged.

    .atrakta/events.jsonl

------------------------------------------------------------------------

## ⚡ Quick Start

### macOS / Linux

    curl -fsSL https://raw.githubusercontent.com/afwm/Atrakta/main/scripts/install.sh | bash
    atrakta init --interfaces cursor

### Windows

Download release binary and run.

------------------------------------------------------------------------

## 🧠 Architecture

![Architecture](docs/images/atrakta_super_architecture.svg)

------------------------------------------------------------------------

## 🔍 Comparison

  Feature                 Cursor   LangChain   Atrakta
  ----------------------- -------- ----------- -------------
  Interface independent   ❌       ❌          ✅
  AI coding runtime       ❌       ❌          ✅
  Reproducible sessions   ❌       ⚠️          ✅
  Event log chain         ❌       ❌          ✅
  Cross‑tool workflow     ❌       ❌          ✅

------------------------------------------------------------------------

## 📦 Files Created

    AGENTS.md

    .atrakta/
      contract.json
      events.jsonl
      state.json
      progress.json
      task-graph.json

------------------------------------------------------------------------

## 📚 Documentation

  Topic          Link
  -------------- -------------------
  English Docs   docs/en/README.md
  日本語 Docs    docs/ja/README.md

------------------------------------------------------------------------

## 🤝 Contributing

Issues and pull requests are welcome.

If you think AI development needs a stable runtime, please consider
starring the repository ⭐

------------------------------------------------------------------------

## License

MIT License

Copyright 2026\
Shogo Maganuma
