# EXPLORATION_INPUT_RETRIEVAL_RULES_ADDENDUM

## Managed Retrieval Rules
Exploration retrieval for imported assets must be:
- bounded (no unbounded recall)
- source-attributed
- scope-aware

## Source Scope Rules
- Source: capability registry and reference-memory surfaces only.
- Retrieval must preserve source attribution per item.

## Kind Filters
- Allowed kinds must be explicitly filtered.
- Unsupported/denied capabilities are excluded.

## Reviewed-only Option
- `reviewed-only` means the capability passed review gate (not just analyzed).
- `reviewed-only` must not be treated as synonymous with `analyzed`.
