package resolvefailuretier

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	TierBlock         = "BLOCK"
	TierDegradeStrict = "DEGRADE_TO_STRICT"
	TierProposalOnly  = "PROPOSAL_ONLY"
	TierWarnOnly      = "WARN_ONLY"
)

var strictTriggers = map[string]struct{}{
	"stale_state":                    {},
	"unresolved_capability":          {},
	"unsupported_projection_surface": {},
	"missing_approval":               {},
	"workspace_mismatch":             {},
	"policy_ambiguity":               {},
	"instruction_conflict":           {},
	"audit_guarantee_shortfall":      {},
}

type failureRule struct {
	DefaultTier         string
	CanOverride         bool
	RequiresHumanReview bool
}

var failureRules = map[string]failureRule{
	"policy_failure":                {DefaultTier: TierBlock, CanOverride: false, RequiresHumanReview: true},
	"approval_failure":              {DefaultTier: TierDegradeStrict, CanOverride: false, RequiresHumanReview: true},
	"capability_resolution_failure": {DefaultTier: TierProposalOnly, CanOverride: true, RequiresHumanReview: false},
	"projection_failure":            {DefaultTier: TierWarnOnly, CanOverride: true, RequiresHumanReview: false},
	"adapter_execution_failure":     {DefaultTier: TierDegradeStrict, CanOverride: true, RequiresHumanReview: false},
	"provenance_failure":            {DefaultTier: TierProposalOnly, CanOverride: true, RequiresHumanReview: true},
	"audit_integrity_failure":       {DefaultTier: TierBlock, CanOverride: false, RequiresHumanReview: true},
	"legacy_conflict_failure":       {DefaultTier: TierDegradeStrict, CanOverride: true, RequiresHumanReview: true},
	"surface_portability_failure":   {DefaultTier: TierProposalOnly, CanOverride: true, RequiresHumanReview: false},
}

// Context contains failure routing context.
type Context struct {
	Scope             string   `json:"scope"`
	Triggers          []string `json:"triggers,omitempty"`
	RequestedOverride string   `json:"requested_override,omitempty"`
	IsDiagnosticsOnly bool     `json:"is_diagnostics_only,omitempty"`
}

// FailureDecision is a strict-compatible failure routing result.
type FailureDecision struct {
	FailureClass        string `json:"failure_class"`
	DefaultTier         string `json:"default_tier"`
	ResolvedTier        string `json:"resolved_tier"`
	CanOverride         bool   `json:"can_override"`
	RequiresHumanReview bool   `json:"requires_human_review"`
	Scope               string `json:"scope"`
	StrictTransition    string `json:"strict_transition"`
	ExecutionAllowed    bool   `json:"execution_allowed"`
	ProjectionAllowed   bool   `json:"projection_allowed"`
}

// ResolveFailureTier resolves failure routing tier and strict transition hints.
func ResolveFailureTier(failureClass string, ctx Context) common.ResolverOutput {
	fc := normalize(failureClass)
	rule, ok := failureRules[fc]
	if !ok {
		rule = failureRule{DefaultTier: TierBlock, CanOverride: false, RequiresHumanReview: true}
	}

	tier := rule.DefaultTier
	if ctx.IsDiagnosticsOnly && fc == "projection_failure" {
		tier = TierWarnOnly
	}
	if rule.CanOverride && isSupportedTier(ctx.RequestedOverride) {
		tier = ctx.RequestedOverride
	}

	strictTransition := "none"
	for _, tr := range ctx.Triggers {
		if _, exists := strictTriggers[normalize(tr)]; exists {
			strictTransition = "strict"
			break
		}
	}
	if strictTransition == "none" {
		switch tier {
		case TierDegradeStrict:
			strictTransition = "strict"
		case TierProposalOnly:
			strictTransition = "guarded"
		}
	}

	execAllowed, projAllowed := allowMatrix(fc, tier, ctx.IsDiagnosticsOnly)

	decision := FailureDecision{
		FailureClass:        fc,
		DefaultTier:         rule.DefaultTier,
		ResolvedTier:        tier,
		CanOverride:         rule.CanOverride,
		RequiresHumanReview: rule.RequiresHumanReview,
		Scope:               normalizeScope(ctx.Scope),
		StrictTransition:    strictTransition,
		ExecutionAllowed:    execAllowed,
		ProjectionAllowed:   projAllowed,
	}

	evidence := make([]string, 0, len(ctx.Triggers)+2)
	evidence = append(evidence, "default_tier="+rule.DefaultTier)
	if ctx.IsDiagnosticsOnly {
		evidence = append(evidence, "diagnostics_only=true")
	}
	for _, tr := range ctx.Triggers {
		evidence = append(evidence, "trigger="+normalize(tr))
	}
	sort.Strings(evidence)

	reason := "failure tier resolved"
	next := "inspect"
	switch tier {
	case TierBlock:
		next = "deny"
		reason = "blocked by failure tier"
	case TierDegradeStrict:
		next = "inspect"
		reason = "degraded to strict"
	case TierProposalOnly:
		next = "propose"
		reason = "proposal-only fallback"
	case TierWarnOnly:
		next = "inspect"
		reason = "warn-only fallback"
	}

	return common.NewOutput(map[string]any{"failure_class": fc, "context": ctx}, decision, reason, evidence, next)
}

func allowMatrix(failureClass, tier string, diagnosticsOnly bool) (executionAllowed bool, projectionAllowed bool) {
	executionAllowed = true
	projectionAllowed = true

	switch tier {
	case TierBlock:
		executionAllowed = false
		projectionAllowed = false
	case TierDegradeStrict:
		executionAllowed = false
		projectionAllowed = true
	case TierProposalOnly:
		executionAllowed = false
		projectionAllowed = true
	case TierWarnOnly:
		executionAllowed = true
		projectionAllowed = true
	}

	// Split handling for execution vs projection failure classes.
	if failureClass == "projection_failure" {
		executionAllowed = true
		if diagnosticsOnly {
			projectionAllowed = true
		} else {
			projectionAllowed = false
		}
	}
	if failureClass == "adapter_execution_failure" {
		executionAllowed = false
		projectionAllowed = true
	}

	return executionAllowed, projectionAllowed
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}

func normalizeScope(scope string) string {
	s := normalize(scope)
	switch s {
	case "request", "task", "workspace":
		return s
	default:
		return "task"
	}
}

func isSupportedTier(tier string) bool {
	switch tier {
	case TierBlock, TierDegradeStrict, TierProposalOnly, TierWarnOnly:
		return true
	default:
		return false
	}
}
