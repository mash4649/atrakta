# ADR-001 Layer Boundary

Status: Accepted

Core contains request, decision, result, and error only.
Canonical contains `canonical_policy`, capability, task state, and audit events.
Extension contains runtime profile, `repo_docs`, `skill_asset`, `workflow_binding`, and provenance.
Cross-layer ownership mutation is forbidden.
