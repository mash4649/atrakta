# Extension Surface

[English](./07_extension_surface.md) | [日本語](../../ja/02_仕様/07_Extension Surface.md)

## Purpose

Define extension primitives as first-class contract entities.

## Extension Entities

- MCP: external capability provider via explicit server configuration.
- plugin: interface-native extension package or module.
- skill: reusable task guidance and local workflow recipe.
- workflow: ordered task sequence and runtime hooks.
- hook: event-driven integration points for shell/git/IDE/runtime lifecycle.

## Merge Strategy

- default: append-first
- explicit modes: `append`, `include`, `replace`

## Brownfield Modes

- append mode: managed block appended to existing files
- include mode: generated include file and explicit include pointer
- replace mode: allowed only for managed targets with explicit user intent

## Unsupported Policy

- no silent ignore
- emit warning with field path and reason
- keep deterministic fallback outputs when native projection is unavailable
- do not mutate unsupported targets in place

## Projection Constraints

- deterministic rendering from canonical contract
- interface-specific projection stays derived and reproducible
- extension objects (`mcp/plugins/skills/workflows/hooks`) are not reverse-synced automatically

## Repair Constraints

- repair mutates managed regions only
- append/include targets must remain idempotent
- existing user-managed content must be preserved

## Status

Normative baseline for extension backlog implementation.
