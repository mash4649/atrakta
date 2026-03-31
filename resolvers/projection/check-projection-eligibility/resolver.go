package checkprojectioneligibility

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	EligibilityAllowed     = "allowed"
	EligibilityConditional = "conditional"
	EligibilityForbidden   = "forbidden"
)

// Source is a projection candidate.
type Source struct {
	Type               string `json:"type"`
	HasCanonicalAnchor bool   `json:"has_canonical_anchor,omitempty"`
}

// ProjectionDecision describes eligibility and operational constraints.
type ProjectionDecision struct {
	Eligibility       string   `json:"eligibility"`
	CanBeDecisionRoot bool     `json:"can_be_decision_root"`
	ProjectionTypes   []string `json:"projection_types"`
}

// CheckProjectionEligibility evaluates source eligibility.
func CheckProjectionEligibility(source Source) common.ResolverOutput {
	t := normalize(source.Type)
	decision := ProjectionDecision{
		Eligibility:       EligibilityForbidden,
		CanBeDecisionRoot: false,
		ProjectionTypes:   []string{"diagnostics"},
	}

	switch t {
	case "policy", "repo_map", "repo-map", "skill", "workflow":
		decision.Eligibility = EligibilityAllowed
		decision.CanBeDecisionRoot = true
		decision.ProjectionTypes = []string{"durable", "ephemeral", "diagnostics"}
	case "decision", "result":
		decision.Eligibility = EligibilityConditional
		decision.CanBeDecisionRoot = source.HasCanonicalAnchor
		decision.ProjectionTypes = []string{"ephemeral", "diagnostics"}
	case "task_state", "task-state", "audit_event", "audit-event":
		decision.Eligibility = EligibilityForbidden
		decision.CanBeDecisionRoot = false
		decision.ProjectionTypes = []string{"diagnostics"}
	default:
		decision.Eligibility = EligibilityForbidden
		decision.CanBeDecisionRoot = false
		decision.ProjectionTypes = []string{"diagnostics"}
	}

	evidence := []string{"type=" + t}
	if source.HasCanonicalAnchor {
		evidence = append(evidence, "has_canonical_anchor=true")
	} else {
		evidence = append(evidence, "has_canonical_anchor=false")
	}
	sort.Strings(evidence)

	next := "inspect"
	reason := "projection source eligible"
	switch decision.Eligibility {
	case EligibilityAllowed:
		next = "propose"
		reason = "projection source allowed"
	case EligibilityConditional:
		next = "inspect"
		reason = "projection source is conditional"
	case EligibilityForbidden:
		next = "deny"
		reason = "projection source forbidden"
	}

	if decision.Eligibility == EligibilityConditional && !decision.CanBeDecisionRoot {
		reason = "conditional source cannot be decision root alone"
	}

	return common.NewOutput(source, decision, reason, evidence, next)
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
