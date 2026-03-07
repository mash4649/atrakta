# Overview

[English](./01_overview.md) | [日本語](../../ja/01_全体/01_概要.md)

Atrakta is a deterministic harness for keeping AI development workflows consistent and safe across multiple interfaces.

## Problems It Solves

- Config/state drift between editors and CLIs
- Implicit destructive changes
- Poor traceability of execution history

## Core Characteristics

- Single binary CLI
- Deterministic Detect -> Plan -> Apply flow
- `managed-only` guard for destructive mutation
- Append-only hash-chained `events.jsonl`
- Recovery and rebuild path via `doctor`

## Current Subcommands

- `init`
- `start`
- `doctor`
- `gc`
- `wrap install|uninstall|run`
- `hook install|uninstall`
- `ide-autostart install|uninstall|status`
- `migrate check`
- `resume`
