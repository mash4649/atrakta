package resolveoperationcapability

import (
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	ActionInspect = "inspect_only"
	ActionPropose = "propose_only"
	ActionApply   = "apply_mutation"
)

// Input describes command/alias resolution context.
type Input struct {
	CommandOrAlias string `json:"command_or_alias"`
	FailureTier    string `json:"failure_tier,omitempty"`
}

// CapabilityDecision is operation capability resolution output.
type CapabilityDecision struct {
	CanonicalCapability  string `json:"canonical_capability"`
	ActionClass          string `json:"action_class"`
	CeilingActionClass   string `json:"ceiling_action_class"`
	EffectiveActionClass string `json:"effective_action_class"`
	AliasResolvedFrom    string `json:"alias_resolved_from,omitempty"`
}

var aliasMap = map[string]string{
	"doctor":      "inspect_health",
	"parity":      "inspect_parity",
	"integration": "inspect_integration",
	"repair":      "propose_repair",
}

var capabilityAction = map[string]string{
	"inspect_health":      ActionInspect,
	"inspect_drift":       ActionInspect,
	"inspect_parity":      ActionInspect,
	"inspect_integration": ActionInspect,
	"propose_repair":      ActionPropose,
	"apply_repair":        ActionApply,
}

// ResolveOperationCapability normalizes aliases to canonical capability names.
func ResolveOperationCapability(in Input) common.ResolverOutput {
	raw := normalize(in.CommandOrAlias)
	canonical := raw
	aliasFrom := ""
	if mapped, ok := aliasMap[raw]; ok {
		canonical = mapped
		aliasFrom = raw
	}

	action, ok := capabilityAction[canonical]
	if !ok {
		canonical = "inspect_health"
		action = ActionInspect
		aliasFrom = raw
	}

	ceiling := ceilingForTier(normalize(in.FailureTier))
	effective := minActionClass(action, ceiling)

	next := "inspect"
	reason := "operation capability resolved"
	switch effective {
	case ActionApply:
		next = "apply"
	case ActionPropose:
		next = "propose"
	case ActionInspect:
		next = "inspect"
	}
	if action != effective {
		reason = "capability clamped by failure tier action ceiling"
	}

	decision := CapabilityDecision{
		CanonicalCapability:  canonical,
		ActionClass:          action,
		CeilingActionClass:   ceiling,
		EffectiveActionClass: effective,
		AliasResolvedFrom:    aliasFrom,
	}

	evidence := []string{"canonical_capability=" + canonical, "ceiling=" + ceiling}
	return common.NewOutput(in, decision, reason, evidence, next)
}

func ceilingForTier(tier string) string {
	switch tier {
	case "block":
		return ActionInspect
	case "proposal_only":
		return ActionPropose
	case "degrade_to_strict":
		return ActionPropose
	case "warn_only", "":
		return ActionApply
	default:
		return ActionPropose
	}
}

func minActionClass(base, ceiling string) string {
	if rank(base) < rank(ceiling) {
		return base
	}
	return ceiling
}

func rank(action string) int {
	switch action {
	case ActionInspect:
		return 1
	case ActionPropose:
		return 2
	case ActionApply:
		return 3
	default:
		return 1
	}
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
