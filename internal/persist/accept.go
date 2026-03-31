package persist

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

// AcceptResult describes persisted artifacts for onboarding acceptance.
type AcceptResult struct {
	ProjectRoot string   `json:"project_root"`
	StoreRoot   string   `json:"store_root"`
	Written     []string `json:"written"`
	AuditLevel  string   `json:"audit_level"`
}

// AcceptOnboarding persists onboarding bundle into .atrakta store.
func AcceptOnboarding(projectRoot string, bundle onboarding.ProposalBundle) (AcceptResult, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return AcceptResult{}, err
	}
	storeRoot := filepath.Join(root, ".atrakta")
	written := make([]string, 0, 16)

	write := func(rel string, payload any) error {
		abs := filepath.Join(storeRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		b, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(abs, append(b, '\n'), 0o644); err != nil {
			return err
		}
		written = append(written, filepath.ToSlash(filepath.Join(".atrakta", rel)))
		return nil
	}

	spec, rubric := onboarding.BuildAcceptanceArtifacts(bundle)
	if err := write("generated/acceptance-spec.generated.json", spec); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/acceptance-rubric.generated.json", rubric); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/onboarding.proposal.accepted.json", bundle); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/repo-map.generated.json", map[string]any{
		"detected_assets": bundle.DetectedAssets,
		"mode":            bundle.InferredMode,
	}); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/capabilities.generated.json", bundle.InferredCapabilities); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/managed-scope.generated.json", bundle.InferredManagedScope); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/legacy-assets.generated.json", map[string]any{
		"conflicts": bundle.Conflicts,
		"risks":     bundle.DetectedRisks,
	}); err != nil {
		return AcceptResult{}, err
	}
	if err := write("generated/initial-policy.generated.json", bundle.InferredDefaultPolicy); err != nil {
		return AcceptResult{}, err
	}
	if err := write("canonical/policies/registry/index.json", map[string]any{
		"entries": []map[string]any{
			{"id": "default-policy", "status": "active"},
		},
	}); err != nil {
		return AcceptResult{}, err
	}
	if err := write("canonical/capabilities/registry/index.json", map[string]any{
		"entries": bundle.InferredCapabilities,
	}); err != nil {
		return AcceptResult{}, err
	}
	if err := write("contract.json", map[string]any{
		"v":          1,
		"project_id": filepath.Base(root),
		"interfaces": map[string]any{
			"supported": []string{"generic-cli", "cursor", "vscode", "mcp", "github-actions"},
			"fallback":  "generic-cli",
		},
		"boundary": map[string]any{
			"include":      []string{""},
			"exclude":      []string{".atrakta/"},
			"managed_root": ".atrakta/",
		},
		"tools": map[string]any{
			"allow": []string{"create", "edit", "run"},
			"deny":  []string{},
		},
		"security": map[string]any{
			"destructive":      "deny",
			"external_send":    "deny",
			"approval":         "explicit",
			"permission_model": "proposal_only",
		},
		"routing": map[string]any{
			"default": map[string]any{
				"worker":  "general",
				"quality": "quick",
			},
		},
	}); err != nil {
		return AcceptResult{}, err
	}
	if err := write("state/onboarding-state.json", map[string]any{
		"status":             "accepted",
		"inferred_mode":      bundle.InferredMode,
		"next_actions":       bundle.SuggestedNextActions,
		"failure_routing":    bundle.InferredFailure,
		"managed_scope_keys": len(bundle.InferredManagedScope),
	}); err != nil {
		return AcceptResult{}, err
	}

	auditLevel := audit.LevelA2
	if _, err := audit.AppendAndVerify(filepath.Join(storeRoot, "audit"), auditLevel, "accept_onboarding", map[string]any{
		"inferred_mode":  bundle.InferredMode,
		"conflicts":      bundle.Conflicts,
		"detected_risks": bundle.DetectedRisks,
	}); err != nil {
		return AcceptResult{}, err
	}
	if _, err := audit.AppendRunEventAndVerify(filepath.Join(storeRoot, "audit"), auditLevel, "onboarding.accepted", map[string]any{
		"inferred_mode":  bundle.InferredMode,
		"conflicts":      bundle.Conflicts,
		"detected_risks": bundle.DetectedRisks,
	}, audit.RunEventOptions{Actor: "kernel"}); err != nil {
		return AcceptResult{}, err
	}
	written = append(written, ".atrakta/audit/events/install-events.jsonl")
	written = append(written, ".atrakta/audit/events/run-events.jsonl")

	return AcceptResult{
		ProjectRoot: root,
		StoreRoot:   storeRoot,
		Written:     written,
		AuditLevel:  auditLevel,
	}, nil
}
