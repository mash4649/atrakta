# 07. Brownfield Integration

[English](./07_brownfield_integration.md) | [日本語](../../ja/03_運用/07_Brownfield導入.md)

## Goal

Integrate Atrakta into an existing repository without destructive overwrite.

## Baseline Flow

```bash
atrakta init \
  --mode brownfield \
  --interfaces cursor \
  --merge-strategy append \
  --agents-mode append \
  --no-overwrite
```

## Principles

- keep user-managed content untouched
- mutate only managed blocks / managed include files
- produce proposal patch when merge is ambiguous

## Post-check

```bash
atrakta doctor
atrakta doctor --parity
atrakta doctor --integration
atrakta projection status --json
```
