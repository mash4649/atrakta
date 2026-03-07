# Hacker News Launch Post (Draft)

## Title

Show HN: Atrakta — Git-like runtime for AI coding workflows (interface-independent)

## Body

AI coding tools are evolving extremely fast.

Cursor, Claude Code, Codex CLI, and many others appear every few months.

But this creates a new problem:

Your development workflow becomes tied to a single interface.
Switch the tool, and the workflow breaks.

Atrakta is a lightweight runtime layer designed to solve this.

It sits between AI tools and your project state, and keeps AI coding:

- reproducible
- resumable
- observable
- safe

You can switch between Cursor, Claude CLI, VS Code, or future AI tools while keeping the same workflow.

Internally it uses a deterministic pipeline:

Detect → Plan → Apply

and maintains an append-only event log (hash-chained):

.atrakta/events.jsonl

Think of it as:

“Git for AI coding sessions.”

Repo:
https://github.com/afwm/Atrakta

Feedback very welcome.

---

## Likely questions + short answers

### How is this different from Cursor?

Cursor is an AI editor.
Atrakta is an AI development runtime: it makes AI coding reproducible and tool-independent.

### Why not just use git?

Git tracks code history.
Atrakta tracks AI workflow state (contracts, events, task graphs, progress) so sessions can resume deterministically.
