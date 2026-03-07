# Atrakta

> **AI coding without interface lock‑in**

Stable AI development across:

**Claude Code · Codex · Cursor · Antigravity · VS Code · CLI · future
tools**

Atrakta is a lightweight runtime layer that makes AI development:

**safe · reproducible · resumable · observable**

------------------------------------------------------------------------

## 🎥 Demo

Switch tools without losing development state.

Cursor → Claude CLI → VSCode

Same workflow. Same state.

*(Replace with demo GIF below)*

------------------------------------------------------------------------

## ⭐ Why Atrakta?

AI coding tools are evolving extremely fast.

However, development workflows often become unstable.

  Problem                What happens
  ---------------------- ---------------------------------------
  Interface lock‑in      Workflow depends on one editor
  Session instability    AI decisions change every run
  Debugging difficulty   Automation failures are hard to trace
  Risky modifications    AI may introduce destructive changes

Atrakta introduces a **runtime layer for AI development**.

> Think of it as **Git for AI coding sessions**.

------------------------------------------------------------------------

## 🚀 Key Features

### Interface‑independent workflows

Switch between AI tools without changing your workflow.

### Reproducible sessions

    atrakta resume

Continue exactly where the AI left off.

### Deterministic development

Atrakta manages:

-   contract
-   events
-   progress
-   task graph

### Full observability

All AI actions are recorded:

    .atrakta/events.jsonl

------------------------------------------------------------------------

## ⚡ Quick Start

macOS / Linux

    curl -fsSL https://raw.githubusercontent.com/afwm/Atrakta/main/scripts/install.sh | bash
    atrakta init --interfaces cursor

Windows

Download release binary and run.

------------------------------------------------------------------------

## 🧠 Architecture

AI Tools → Atrakta → Project State

contract / events / progress / task graph

------------------------------------------------------------------------

## 📚 Documentation

  Topic               Link
  ------------------- ---------------------------------------------
  English Docs        docs/en/README.md
  日本語 Docs         docs/ja/README.md
  Installation        docs/en/03_operations/01_setup.md
  Troubleshooting     docs/en/03_operations/03_troubleshooting.md
  CLI Specification   docs/en/02_spec/01_cli_spec.md

------------------------------------------------------------------------

## 🤝 Contributing

PRs and issues welcome.

If you believe AI coding needs a stable runtime,\
⭐ consider starring this repository.

------------------------------------------------------------------------

## License

Apache License 2.0

Copyright 2026\
Shogo Maganuma
