# License Options (Proposal)

## Objective

Select an OSS license for Atrakta with a clear understanding of trade-offs.
This document compares a small set of common licenses and proposes a recommendation **pending owner choice**.

## Scope and Assumptions

- Atrakta is intended to be usable by individuals and organizations, including commercial users.
- We want straightforward redistribution and contribution flow.
- We may want patent safety for adopters (common concern for companies).

If any assumption is wrong, treat the recommendation section as void and re-evaluate.

## Options Compared

### Quick comparison

| License | Type | Patent grant | Copyleft | Notable obligations |
|---|---|---:|---:|---|
| MIT | Permissive | No explicit patent license | No | Keep copyright + license notice |
| Apache-2.0 | Permissive | **Yes (explicit)** + patent retaliation | No | Keep NOTICE (if any), preserve headers, state changes |
| MPL-2.0 | Weak copyleft | Yes (explicit) | **File-level** | Modified MPL files must remain MPL when distributed |

### MIT

- **What it optimizes for**: maximum simplicity and adoption, minimal obligations.
- **What you give up**: no explicit patent license; some companies prefer an explicit grant.
- **Good fit when**:
  - You want the shortest, simplest license.
  - Patent posture is intentionally left unspecified (or handled elsewhere).

### Apache License 2.0

- **What it optimizes for**: permissive adoption with stronger legal clarity, especially around patents.
- **Key features**:
  - **Explicit patent license grant** from contributors to users.
  - Patent retaliation clause (helps discourage patent aggression).
  - Clear requirements on preserving notices and stating modifications.
- **Good fit when**:
  - You want broad commercial adoption and clearer patent comfort.
  - You expect external contributions and want a default patent posture.

### Mozilla Public License 2.0 (MPL-2.0)

- **What it optimizes for**: encouraging improvements to be shared back **without** requiring full project copyleft.
- **How copyleft works**:
  - If someone modifies MPL-licensed files and distributes them, those modified files must remain MPL.
  - New files can be under other terms (subject to MPL’s rules), which makes it less restrictive than GPL-style licenses.
- **Good fit when**:
  - You want commercial use to remain possible but want stronger reciprocity on file modifications.
  - You are okay with some adopters avoiding it due to compliance overhead.

## Practical Implications (What will change for us)

### Contributions and inbound licensing

- **MIT**: simplest; inbound contributions typically accepted under the same terms via repository policy.
- **Apache-2.0**: common for contributor-heavy projects; pairs well with a `NOTICE` file if needed.
- **MPL-2.0**: requires discipline in file-level boundaries; contributors should understand file-level copyleft rules.

### Patents and corporate adoption

- **MIT**: fewer explicit assurances; some legal teams may request clarification.
- **Apache-2.0**: usually easiest for corporate legal review due to explicit patent grant.
- **MPL-2.0**: includes patent language but copyleft may reduce adoption in some orgs.

### Compatibility and ecosystem expectations

- All three are widely recognized and OSI-approved.
- **Apache-2.0** is especially common for infrastructure/CLI tooling in corporate environments.
- **MIT** is very common for libraries and small tools.
- **MPL-2.0** is common when maintainers want reciprocity without full copyleft.

## Recommendation (Proposal — owner decision required)

### Proposed default: Apache-2.0

**Rationale**:

- Aligns with likely adoption goals for a CLI/tooling project (broad commercial use).
- Provides **explicit patent grant**, reducing friction for organizations.
- Clear notice/attribution requirements without copyleft.

### Viable alternatives

- **Pick MIT** if we value maximum simplicity above patent clarity and want the shortest license text.
- **Pick MPL-2.0** if we want to encourage upstreaming of modifications (file-level reciprocity) while staying non-viral at the project level.

## Owner decision checklist

- Do we want an explicit patent grant by default? (If yes, prefer **Apache-2.0** or **MPL-2.0**.)
- Do we want any reciprocity requirement on modifications? (If yes, consider **MPL-2.0**.)
- Is minimizing compliance burden the top priority? (If yes, consider **MIT**.)

## Next step (after owner choice)

- Add a top-level `LICENSE` file matching the selected license.
- If Apache-2.0 is selected and we need it, add `NOTICE` (or confirm none is required).
- Update README/docs to declare the chosen license succinctly.

