# classify_layer

Signature:

`classify_layer(item) -> core | canonical | extension | unknown`

Implemented in:

- `resolver.go` as `ClassifyLayer(Item) common.ResolverOutput`

Rules:

- Kind-first classification with schema id fallback
- Unknown classification returns `next_allowed_action = deny`
- Output follows `schemas/operations/inspect-output.schema.json`
