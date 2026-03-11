# MEMORY_PROMOTION_RULES_ADDENDUM

## Promotion Prerequisites
Memory promotion requires both:
- operator decision
- review status = `approved`

Without either, promotion is denied.

## Review Statuses
- `pending`
- `approved`
- `rejected`

## TTL / Retention
Promotion metadata should include retention policy as an operator-managed rule.
Default posture for imported memory remains `reference_memory` until explicit review.

## Operator Override
Operator override is allowed only with explicit review record and event logging.
All overrides must append `memory_promotion_reviewed`.
