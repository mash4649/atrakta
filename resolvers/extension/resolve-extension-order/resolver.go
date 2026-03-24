package resolveextensionorder

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

// Item is an extension candidate or binding node.
type Item struct {
	ID                             string `json:"id"`
	Kind                           string `json:"kind"`
	AttemptsCoreMutation           bool   `json:"attempts_core_mutation,omitempty"`
	DiagnosticsConstrainsExecution bool   `json:"diagnostics_constrains_execution,omitempty"`
	HookMutatesCanonical           bool   `json:"hook_mutates_canonical,omitempty"`
}

// OrderedItem includes resolved order metadata.
type OrderedItem struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	FirstPassGroup string `json:"first_pass_group"`
	Order          int    `json:"order"`
}

// ExtensionDecision is extension-order output.
type ExtensionDecision struct {
	Ordered    []OrderedItem `json:"ordered"`
	Violations []string      `json:"violations"`
}

// ResolveExtensionOrder resolves deterministic extension evaluation order.
func ResolveExtensionOrder(items []Item) common.ResolverOutput {
	ordered := make([]OrderedItem, 0, len(items))
	violations := []string{}

	for _, it := range items {
		k := normalize(it.Kind)
		group, order := classify(k)
		ordered = append(ordered, OrderedItem{ID: it.ID, Kind: k, FirstPassGroup: group, Order: order})

		if it.AttemptsCoreMutation {
			violations = append(violations, it.ID+":none_can_mutate_core_contracts_directly")
		}
		if it.DiagnosticsConstrainsExecution {
			violations = append(violations, it.ID+":diagnostics_asset_cannot_constrain_execution_policy")
		}
		if k == "hook" && it.HookMutatesCanonical {
			violations = append(violations, it.ID+":hook_cannot_directly_mutate_canonical_state")
		}
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Order != ordered[j].Order {
			return ordered[i].Order < ordered[j].Order
		}
		return ordered[i].ID < ordered[j].ID
	})

	next := "propose"
	reason := "extension order resolved"
	if len(violations) > 0 {
		next = "deny"
		reason = "extension boundary violation detected"
	}

	evidence := []string{"evaluation_order=canonical_policy>workflow>skill>tool_hint>runtime_hook>projection_plugin"}
	return common.NewOutput(items, ExtensionDecision{Ordered: ordered, Violations: violations}, reason, evidence, next)
}

func classify(kind string) (group string, order int) {
	switch kind {
	case "policy":
		return "orchestration_assets", 1
	case "workflow":
		return "orchestration_assets", 2
	case "skill":
		return "orchestration_assets", 3
	case "tool_hint", "tool-hint":
		return "orchestration_assets", 4
	case "hook":
		return "runtime_hooks", 5
	case "plugin", "projection_plugin", "projection-plugin":
		return "capability_adapters", 6
	case "mcp":
		return "capability_adapters", 7
	default:
		return "orchestration_assets", 8
	}
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
