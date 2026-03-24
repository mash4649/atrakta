# Resolve Surface Portability

`resolve_surface_portability(input) -> portability_decision`

Determines whether one interface can preserve canonical semantics for the
requested projection targets without forcing identical UX.

v1 target vocabulary:

- `agents_md`
- `ide_rules`
- `repo_docs`
- `skill_bundle`

Unsupported or degraded targets do not imply blanket block.
Default degradation policy is `proposal_only`.
