# Atrakta

**Make AI coding safe, reproducible, and observable.**

A lightweight runtime layer that manages AI coding workflows across\
**Claude Code / Codex / Cursor / Antigravity / VS Code / etc pipelines**.

Atrakta introduces **state management, safety contracts, and event
logging** for AI‑driven development.

------------------------------------------------------------------------

## ⭐ Why Atrakta?

AI coding tools are incredibly powerful --- but real projects quickly
encounter problems.

  Problem                 What happens
  ----------------------- --------------------------------------------
  Configuration drift     IDE / CLI / CI behave differently
  Session inconsistency   AI decisions change between sessions
  Hard debugging          Automation failures are hard to trace
  Risky changes           AI may introduce destructive modifications

**Atrakta solves this by adding an operational layer for AI
development.**

Think of it as:

> **Git for AI coding sessions.**

------------------------------------------------------------------------

## 🚀 What Makes Atrakta Different

### Deterministic AI Workflows

AI work becomes **reproducible and restartable**.

### Cross‑Tool Consistency

IDE, and CLI follow the **same operational rules**.

### Full Event Logging

Every AI action is recorded in a structured log.

### Safety Contracts

Prevent destructive or unsafe changes automatically.

### Task Graph Tracking

Understand and visualize AI progress.

------------------------------------------------------------------------

## ⚡ Quick Start (3 minutes)

### macOS / Linux

``` bash
curl -fsSL https://raw.githubusercontent.com/afwm/Atrakta/main/scripts/install.sh | bash
atrakta init --interfaces cursor
```

### Windows

Download the release binary and run:

``` powershell
$targetDir = "$env:USERPROFILE\AppData\Local\Programs\atrakta"
New-Item -ItemType Directory -Force $targetDir | Out-Null
Copy-Item .\atrakta.exe "$targetDir\atrakta.exe" -Force
```

Then initialize your project:

    atrakta init --interfaces cursor

------------------------------------------------------------------------

## 📦 What Atrakta Creates

Running `atrakta init --interfaces <id>` generates the operational state layer:

    AGENTS.md

    .atrakta/
      contract.json
      events.jsonl
      state.json
      progress.json
      task-graph.json

These files allow AI work to be:

-   resumable
-   observable
-   reproducible
-   safe

------------------------------------------------------------------------

## 🧭 Basic Workflow

Daily commands:

``` bash
atrakta start --interfaces cursor
atrakta resume
atrakta doctor
```

  Command   Purpose
  --------- ------------------------
  start     Start a new AI session
  resume    Resume previous work
  doctor    Diagnose system state

------------------------------------------------------------------------

## 🧠 Architecture

Atrakta sits between your AI tools and project state.

    AI Tools (Cursor / CLI / VSCode)
               ↓
          Atrakta
               ↓
       Project Operational State
       ├ contract
       ├ events
       ├ progress
       └ task graph

This architecture ensures:

-   safe AI execution
-   deterministic workflows
-   recoverable automation

------------------------------------------------------------------------

## 🔍 Observability

All AI operations are stored in:

    .atrakta/events.jsonl

This enables:

-   debugging AI automation
-   auditing AI changes
-   replaying development sessions

------------------------------------------------------------------------

## 🛠 Maintainer Workflow

Atrakta automatically publishes releases when pushing to `main`.

    .github/workflows/release.yml

Manual build fallback:

``` bash
./scripts/build_release_artifacts.sh
```

------------------------------------------------------------------------

## 📚 Documentation

| Topic | Link |
|---|---|
| English Docs | [docs/en/README.md](docs/en/README.md) |
| 日本語 Docs | [docs/ja/README.md](docs/ja/README.md) |
| Installation | [docs/en/03_operations/01_setup.md](docs/en/03_operations/01_setup.md) |
| Troubleshooting | [docs/en/03_operations/03_troubleshooting.md](docs/en/03_operations/03_troubleshooting.md) |
| CLI Specification | [docs/en/02_spec/01_cli_spec.md](docs/en/02_spec/01_cli_spec.md) |

------------------------------------------------------------------------

## 🤝 Contributing

Contributions are welcome.

You can help by:

-   opening issues
-   submitting pull requests
-   improving documentation
-   sharing feedback

------------------------------------------------------------------------

## 📜 License

MIT License

Copyright 2026\
Shogo Maganuma
