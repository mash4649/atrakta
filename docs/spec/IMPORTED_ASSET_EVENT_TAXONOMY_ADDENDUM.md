# IMPORTED_ASSET_EVENT_TAXONOMY_ADDENDUM

This addendum extends event taxonomy without replacing existing events.

## Event Types
- `capability_imported`
- `capability_analyzed`
- `capability_quarantined`
- `capability_promoted`
- `recipe_candidate_created`
- `recipe_conversion_reviewed`
- `memory_surface_assigned`
- `memory_promotion_reviewed`

## Minimal Payload Requirements
Common minimum:
- `capability_id`
- `import_batch_id` (where applicable)

Type-specific minimum:
- `capability_imported`: `kind`, `path`, `source_type`, `content_hash`
- `capability_analyzed`: `risk`, `filesystem_access`, `network_access`, `secrets_access`, `bounded`
- `capability_quarantined`: `quarantine_reason`
- `capability_promoted`: `to_surface`, `review_status`
- `recipe_candidate_created`: `max_steps`, `timeout_sec`
- `recipe_conversion_reviewed`: `review_status`
- `memory_surface_assigned`: `memory_surface`
- `memory_promotion_reviewed`: `review_status`, `promoted`, `reason`

## Traceability
The event stream must allow tracing:
- import -> analyze -> quarantine status
- review -> conversion -> promotion decisions

No existing event type is overwritten by this addendum.
