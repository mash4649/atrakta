package persist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

func TestAcceptOnboarding(t *testing.T) {
	root := t.TempDir()
	bundle := onboarding.ProposalBundle{
		DetectedAssets:       []string{"AGENTS.md"},
		DetectedRisks:        []string{},
		InferredMode:         onboarding.ModeBrownfield,
		InferredManagedScope: map[string]any{".atrakta/generated/**": "managed_block"},
		InferredCapabilities: []string{"inspect_repo", "propose_repair"},
		InferredGuidance:     map[string]any{"canonical_policy": "authoritative_constraint"},
		InferredDefaultPolicy: map[string]any{
			"read_only": "allow",
		},
		InferredFailure: onboarding.FailurePreview{
			FailureClass:      "legacy_conflict_failure",
			Scope:             "workspace",
			Triggers:          []string{"instruction_conflict"},
			DefaultTier:       "DEGRADE_TO_STRICT",
			ResolvedTier:      "DEGRADE_TO_STRICT",
			StrictTransition:  "strict",
			ExecutionAllowed:  false,
			ProjectionAllowed: true,
			NextAllowedAction: "inspect",
		},
		Conflicts:            []string{"possible duplicate guidance"},
		SuggestedNextActions: []string{"inspect details"},
	}

	out, err := AcceptOnboarding(root, bundle)
	if err != nil {
		t.Fatalf("accept onboarding: %v", err)
	}
	if out.StoreRoot == "" || len(out.Written) == 0 {
		t.Fatalf("accept result incomplete")
	}
	if _, err := os.Stat(filepath.Join(root, ".atrakta/state/onboarding-state.json")); err != nil {
		t.Fatalf("missing onboarding-state.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".atrakta/audit/events/install-events.jsonl")); err != nil {
		t.Fatalf("missing audit log: %v", err)
	}
}
