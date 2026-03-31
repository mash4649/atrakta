# detect_legacy_drift

Signature:

`detect_legacy_drift(signals) -> drift_decision`

Implemented in:

- `resolver.go` as `DetectLegacyDrift(Input) common.ResolverOutput`

Rules:

- detects known drift conditions only
- warn drift leads to proposal path
- strict drift leads to deny path
