package resolveguidanceprecedence

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	StrengthAuthoritative = "authoritative_constraint"
	StrengthOrchestration = "orchestration_constraint"
	StrengthExecutable    = "executable_guidance"
	StrengthAdvisory      = "advisory_map"
	StrengthHint          = "tool_hint"
)

// GuidanceItem is a single guidance input.
type GuidanceItem struct {
	ID                         string `json:"id"`
	Type                       string `json:"type"`
	MappedToCanonicalPolicy    bool   `json:"mapped_to_canonical_policy,omitempty"`
	ClaimsDecisionOverride     bool   `json:"claims_decision_override,omitempty"`
	ClaimsApprovalSubstitution bool   `json:"claims_approval_substitution,omitempty"`
}

// GuidanceDecision is a precedence-resolved guidance summary.
type GuidanceDecision struct {
	OrderedIDs      []string             `json:"ordered_ids"`
	Classifications []GuidanceResolution `json:"classifications"`
	Violations      []string             `json:"violations"`
}

// GuidanceResolution describes mapped strength/surfaces and precedence.
type GuidanceResolution struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Strength   string   `json:"strength"`
	Surfaces   []string `json:"surfaces"`
	Precedence int      `json:"precedence"`
}

type mappedGuidance struct {
	item GuidanceItem
	res  GuidanceResolution
}

// ResolveGuidancePrecedence sorts guidance by canonical precedence.
func ResolveGuidancePrecedence(items []GuidanceItem) common.ResolverOutput {
	mapped := make([]mappedGuidance, 0, len(items))
	violations := []string{}

	for _, it := range items {
		res := classify(it)
		if it.ClaimsDecisionOverride && (res.Strength == StrengthAdvisory || res.Strength == StrengthHint) {
			violations = append(violations, it.ID+":advisory_or_hint_cannot_override_decision")
		}
		if it.ClaimsApprovalSubstitution {
			violations = append(violations, it.ID+":approval_substitution_forbidden")
		}
		mapped = append(mapped, mappedGuidance{item: it, res: res})
	}

	sort.SliceStable(mapped, func(i, j int) bool {
		if mapped[i].res.Precedence != mapped[j].res.Precedence {
			return mapped[i].res.Precedence < mapped[j].res.Precedence
		}
		return mapped[i].res.ID < mapped[j].res.ID
	})

	decision := GuidanceDecision{
		OrderedIDs:      make([]string, 0, len(mapped)),
		Classifications: make([]GuidanceResolution, 0, len(mapped)),
		Violations:      violations,
	}
	for _, m := range mapped {
		decision.OrderedIDs = append(decision.OrderedIDs, m.res.ID)
		decision.Classifications = append(decision.Classifications, m.res)
	}

	next := "propose"
	reason := "guidance precedence resolved"
	if len(violations) > 0 {
		next = "deny"
		reason = "guidance policy violations detected"
	}

	evidence := []string{"canonical_policy_always_wins"}
	if len(violations) > 0 {
		evidence = append(evidence, "violations_present")
	}

	return common.NewOutput(items, decision, reason, evidence, next)
}

func classify(it GuidanceItem) GuidanceResolution {
	t := normalize(it.Type)
	res := GuidanceResolution{
		ID:   it.ID,
		Type: t,
	}

	switch t {
	case "policy":
		res.Strength = StrengthAuthoritative
		res.Surfaces = []string{"decision", "mutation"}
		res.Precedence = 1
	case "workflow":
		res.Strength = StrengthOrchestration
		res.Surfaces = []string{"orchestration", "mutation"}
		res.Precedence = 2
	case "skill":
		res.Strength = StrengthExecutable
		res.Surfaces = []string{"orchestration", "mutation"}
		res.Precedence = 3
	case "repo_map", "repo-map":
		res.Strength = StrengthAdvisory
		res.Surfaces = []string{"projection", "diagnostics"}
		res.Precedence = 4
	case "tool_hint", "tool-hint":
		res.Strength = StrengthHint
		res.Surfaces = []string{"diagnostics"}
		res.Precedence = 5
	case "legacy":
		if it.MappedToCanonicalPolicy {
			res.Strength = StrengthAuthoritative
			res.Surfaces = []string{"decision", "mutation"}
			res.Precedence = 1
		} else {
			res.Strength = StrengthAdvisory
			res.Surfaces = []string{"diagnostics"}
			res.Precedence = 4
		}
	default:
		res.Strength = StrengthAdvisory
		res.Surfaces = []string{"diagnostics"}
		res.Precedence = 4
	}

	return res
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
