# Community Templates

These templates are starter configs for community adoption.
Each template keeps Atrakta in a safe, proposal-first posture.

## 1. Go Service

Use when the repo is primarily a Go service or CLI.

- Suggested assets:
  - `go.mod`
  - `cmd/`
  - `internal/`
  - `tests/`
- Atrakta focus:
  - `run`
  - `inspect`
  - `preview`
  - `simulate`
- Default guidance:
  - keep managed scope on generated/state/audit paths
  - keep source writes proposal-only unless explicitly approved

## 2. Python App

Use when the repo centers on Python automation or app logic.

- Suggested assets:
  - `pyproject.toml`
  - `src/`
  - `tests/`
  - `requirements.txt`
- Atrakta focus:
  - `run`
  - `onboard`
  - `mutate inspect`
  - `mutate propose`
- Default guidance:
  - detect packaging and test entrypoints
  - keep dependency changes explicit

## 3. TypeScript Web App

Use when the repo is a web app or full-stack TypeScript project.

- Suggested assets:
  - `package.json`
  - `src/`
  - `app/`
  - `public/`
- Atrakta focus:
  - `run`
  - `preview`
  - `simulate`
  - `projection render`
- Default guidance:
  - detect workspace tooling and project scripts
  - keep framework config edits in managed scope

## Shared Safety Rules

- Do not write canonical files directly from templates.
- Keep initial changes proposal-first.
- Preserve the normal `atrakta run` execution model.
- Prefer the smallest working set of files for the first proposal.
