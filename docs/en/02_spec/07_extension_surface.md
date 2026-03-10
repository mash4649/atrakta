# Extension Surface

[English](./07_extension_surface.md) | [日本語](../../ja/02_仕様/07_Extension Surface.md)

## Purpose

Define extension primitives as first-class contract entities.

## Extension Entities

- MCP
- plugin
- skill
- workflow
- hook

## Merge Strategy

- default: append-first
- explicit modes: `append`, `include`, `replace`

## Brownfield Modes

- append mode: managed block appended to existing files
- include mode: generated include file and explicit include pointer

## Unsupported Policy

- no silent ignore
- emit warning with field path and reason
- keep deterministic fallback outputs when native projection is unavailable

## Projection Constraints

- deterministic rendering from canonical contract
- interface-specific projection stays derived and reproducible

## Status

Draft specification for extension backlog implementation.
