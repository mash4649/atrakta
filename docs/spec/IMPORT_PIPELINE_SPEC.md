# IMPORT_PIPELINE_SPEC

## Pipeline Stages
1. Load (`local_directory` deterministic loader)
2. Normalize (classification + deny/quarantine-first)
3. Register (capability registry update)
4. Analyze (analyze-only hook; no auto-enable)
5. Review/convert/promotion (manual gates)

## Security Posture
- Default quarantine-first.
- Import and analyze do not grant executable trust.
- Conversion and promotion require explicit review path.

## Determinism
- Stable file order
- Stable content hash
- Stable batch id
- Same state -> same output
