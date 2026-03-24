package checkmutationscope

import (
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

const (
	ScopeManagedBlock        = "managed_block"
	ScopeManagedInclude      = "managed_include"
	ScopeGeneratedProjection = "generated_projection"
	ScopeUnmanagedUser       = "unmanaged_user_region"
	ScopeProposalPatchOnly   = "proposal_patch_only"
)

// Target describes a mutation target candidate.
type Target struct {
	Path            string `json:"path"`
	DeclaredScope   string `json:"declared_scope,omitempty"`
	AssetType       string `json:"asset_type,omitempty"`
	Operation       string `json:"operation,omitempty"`
	HasAmbiguity    bool   `json:"has_ambiguity,omitempty"`
	ManagedOnlyPath bool   `json:"managed_only_path,omitempty"`
}

// MutationDecision is the mutation scope resolution output.
type MutationDecision struct {
	Scope                   string   `json:"scope"`
	ImplicitMutationAllowed bool     `json:"implicit_mutation_allowed"`
	AllowedModes            []string `json:"allowed_modes"`
	ReverseSyncMode         string   `json:"reverse_sync_mode"`
	Policy                  string   `json:"policy"`
}

// CheckMutationScope classifies mutation scope and allowed action ceiling.
func CheckMutationScope(target Target) common.ResolverOutput {
	scope := resolveScope(target)
	decision := MutationDecision{
		Scope:                   scope,
		ImplicitMutationAllowed: false,
		AllowedModes:            []string{"inspect"},
		ReverseSyncMode:         "proposal_only",
		Policy:                  "default",
	}

	next := "inspect"
	reason := "mutation scope resolved"

	switch scope {
	case ScopeManagedBlock:
		decision.ImplicitMutationAllowed = true
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose", "apply"}
		decision.Policy = "managed apply allowed with decision envelope"
		next = "apply"
	case ScopeManagedInclude:
		decision.ImplicitMutationAllowed = false
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose", "apply"}
		decision.Policy = "append/include preferred"
		next = "propose"
	case ScopeGeneratedProjection:
		decision.ImplicitMutationAllowed = true
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose", "apply"}
		decision.Policy = "generated output replace allowed"
		next = "apply"
	case ScopeProposalPatchOnly:
		decision.ImplicitMutationAllowed = false
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose"}
		decision.Policy = "proposal-only explicit approval required"
		next = "propose"
	case ScopeUnmanagedUser:
		decision.ImplicitMutationAllowed = false
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose"}
		decision.Policy = "implicit mutate forbidden"
		next = "propose"
		reason = "unmanaged user region defaults to proposal-only"
	default:
		decision.ImplicitMutationAllowed = false
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose"}
		decision.Policy = "unknown scope fallback to proposal-only"
		next = "propose"
	}

	// Domain-specific mutation constraints.
	assetType := normalize(target.AssetType)
	op := normalize(target.Operation)
	if assetType == "repo_map" || assetType == "repo-map" {
		decision.Policy = "append/include preferred"
		next = "propose"
	}
	if assetType == "tool_config" || assetType == "tool-config" {
		decision.Policy = "include/proposal-only preferred"
		next = "propose"
	}
	if assetType == "policy" && op == "replace" && !target.ManagedOnlyPath {
		decision.Policy = "policy canonical replace disallowed except managed-only"
		decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose"}
		next = "propose"
	}
	if assetType == "existing_user_rules" || assetType == "existing-user-rules" {
		if target.HasAmbiguity {
			decision.Policy = "ambiguity fallback to proposal-only"
			decision.AllowedModes = []string{"inspect", "preview", "simulate", "propose"}
			next = "propose"
		}
	}

	evidence := []string{
		"scope=" + scope,
		"asset_type=" + assetType,
		"operation=" + op,
		"dual_read_single_write=true",
		"reverse_sync=proposal_only",
	}

	return common.NewOutput(target, decision, reason, evidence, next)
}

func resolveScope(target Target) string {
	if s := normalize(target.DeclaredScope); s != "" {
		return s
	}
	path := normalize(target.Path)
	switch {
	case strings.Contains(path, ".harness/generated/"), strings.Contains(path, "/generated/"):
		return ScopeGeneratedProjection
	case strings.Contains(path, ".harness/canonical/"), strings.Contains(path, "/canonical/"):
		return ScopeManagedBlock
	case strings.Contains(path, ".atrakta/generated/"):
		return ScopeGeneratedProjection
	case strings.Contains(path, ".atrakta/canonical/"), strings.Contains(path, ".atrakta/audit/"), strings.Contains(path, ".atrakta/state/"):
		return ScopeManagedBlock
	case strings.Contains(path, "docs/generated/"):
		return ScopeManagedInclude
	case strings.Contains(path, "agents.md"), strings.Contains(path, "rules"):
		return ScopeProposalPatchOnly
	default:
		return ScopeUnmanagedUser
	}
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
