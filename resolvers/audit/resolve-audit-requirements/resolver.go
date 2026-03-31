package resolveauditrequirements

import (
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

// Input describes requested audit-sensitive action.
type Input struct {
	Action                  string `json:"action"`
	RequestedIntegrityLevel string `json:"requested_integrity_level,omitempty"`
	DestructiveCleanup      bool   `json:"destructive_cleanup,omitempty"`
}

// AuditDecision describes required integrity and retention constraints.
type AuditDecision struct {
	RequiredIntegrityLevel           string `json:"required_integrity_level"`
	AppendOnly                       bool   `json:"append_only"`
	PerEventHashRequired             bool   `json:"per_event_hash_required"`
	PrevHashChainRequired            bool   `json:"prev_hash_chain_required"`
	ChainVerificationRequired        bool   `json:"chain_verification_required"`
	CheckpointRequired               bool   `json:"checkpoint_required"`
	DryRunRequired                   bool   `json:"dry_run_required"`
	DestructiveCleanupMode           string `json:"destructive_cleanup_mode"`
	ProtectCoreAuditHead             bool   `json:"protect_core_audit_head"`
	ProtectManifestProjectionLinkage bool   `json:"protect_manifest_projection_linkage"`
	StrictTriggerOnShortfall         bool   `json:"strict_trigger_on_shortfall"`
}

// ResolveAuditRequirements returns action-specific audit requirements.
func ResolveAuditRequirements(in Input) common.ResolverOutput {
	action := normalizeAction(in.Action)
	level := requiredLevel(action, normalizeLevel(in.RequestedIntegrityLevel))

	decision := AuditDecision{
		RequiredIntegrityLevel:           level,
		AppendOnly:                       true,
		PerEventHashRequired:             levelRank(level) >= levelRank("A1"),
		PrevHashChainRequired:            levelRank(level) >= levelRank("A2"),
		ChainVerificationRequired:        levelRank(level) >= levelRank("A3"),
		CheckpointRequired:               levelRank(level) >= levelRank("A3"),
		DryRunRequired:                   action == "archive" || in.DestructiveCleanup,
		DestructiveCleanupMode:           "proposal_only",
		ProtectCoreAuditHead:             true,
		ProtectManifestProjectionLinkage: true,
		StrictTriggerOnShortfall:         true,
	}

	next := "inspect"
	reason := "audit requirements resolved"
	if in.DestructiveCleanup {
		next = "propose"
		reason = "destructive cleanup must be proposal-only"
	}

	evidence := []string{"append_only=true", "integrity_level=" + level}
	if decision.DryRunRequired {
		evidence = append(evidence, "dry_run_required=true")
	}

	return common.NewOutput(in, decision, reason, evidence, next)
}

func requiredLevel(action, requested string) string {
	base := "A1"
	switch action {
	case "inspect", "diagnostics":
		base = "A0"
	case "propose", "apply", "mutation":
		base = "A2"
	case "release", "archive", "checkpoint":
		base = "A3"
	}
	if isLevel(requested) && levelRank(requested) > levelRank(base) {
		return requested
	}
	return base
}

func isLevel(level string) bool {
	switch normalizeLevel(level) {
	case "A0", "A1", "A2", "A3":
		return true
	default:
		return false
	}
}

func levelRank(level string) int {
	switch normalizeLevel(level) {
	case "A0":
		return 0
	case "A1":
		return 1
	case "A2":
		return 2
	case "A3":
		return 3
	default:
		return 0
	}
}

func normalizeAction(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}

func normalizeLevel(v string) string {
	v = strings.TrimSpace(strings.ToUpper(v))
	v = strings.ReplaceAll(v, " ", "")
	return v
}
