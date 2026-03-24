package validation

import (
	"encoding/json"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/fixtures"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
)

func TestBundleInputValidation(t *testing.T) {
	input := pipeline.DefaultInput("inspect")
	if err := ValidateBundleInput(input); err != nil {
		t.Fatalf("validate default input: %v", err)
	}

	bad := input
	bad.LayerItem.Kind = ""
	if err := ValidateBundleInput(bad); err == nil {
		t.Fatalf("expected validation error for missing layer kind")
	}
}

func TestDecodeAndValidateBundleInput(t *testing.T) {
	input := pipeline.DefaultInput("preview")
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	decoded, err := DecodeAndValidateBundleInput(raw)
	if err != nil {
		t.Fatalf("decode input: %v", err)
	}
	if decoded.FailureClass != input.FailureClass {
		t.Fatalf("decoded failure class mismatch")
	}
}

func TestDecodeAndValidateBundleInputRejectsSchemaTypeMismatch(t *testing.T) {
	badJSON := []byte(`{
  "layer_item":{"kind":"request","schema_id":"atrakta/schemas/core/request.schema.json"},
  "guidance_items":[{"id":"policy-1","type":"policy"}],
  "projection_source":{"type":"policy"},
  "failure_class":"approval_failure",
  "failure_context":{"scope":"task"},
  "mutation_target":{"path":"AGENTS.md"},
  "legacy_asset":{"asset_id":"legacy-1","ownership":"known","freshness":"acceptable","canonical_mapping":"exists"},
  "operation_input":{"command_or_alias":"doctor"},
  "extension_items":[{"id":"policy-1","kind":"policy"}],
  "audit_input":{"action":"inspect"},
  "strict_input":{"current_state":"normal","scope":"task"},
  "unexpected":"not allowed"
}`)
	if _, err := DecodeAndValidateBundleInput(badJSON); err == nil {
		t.Fatalf("expected schema validation error for additional property")
	}
}

func TestBundleOutputValidation(t *testing.T) {
	out, err := pipeline.ExecuteOrdered("inspect", pipeline.DefaultInput("inspect"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if err := ValidateBundleOutput(out); err != nil {
		t.Fatalf("validate output: %v", err)
	}
}

func TestFixtureReportValidation(t *testing.T) {
	report := fixtures.Report{Passed: 1, Failed: 0, Results: []fixtures.CaseResult{{Fixture: "fixtures/core/classify-layer.fixture.json", Case: 1, Passed: true}}}
	if err := ValidateFixtureReport(report); err != nil {
		t.Fatalf("validate fixture report: %v", err)
	}
}

func TestOnboardingProposalValidation(t *testing.T) {
	bundle := onboarding.ProposalBundle{
		DetectedAssets:       []string{"AGENTS.md"},
		DetectedRisks:        []string{},
		InferredMode:         onboarding.ModeBrownfield,
		InferredManagedScope: map[string]any{".atrakta/generated/**": "managed_block"},
		InferredCapabilities: []string{"inspect_repo"},
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
		Conflicts:            []string{},
		SuggestedNextActions: []string{"inspect details"},
	}
	if err := ValidateOnboardingProposal(bundle); err != nil {
		t.Fatalf("validate onboarding proposal: %v", err)
	}
}

func TestOnboardingProposalValidationRejectsMode(t *testing.T) {
	bundle := onboarding.ProposalBundle{
		DetectedAssets:       []string{},
		DetectedRisks:        []string{},
		InferredMode:         "invalid_mode",
		InferredManagedScope: map[string]any{".atrakta/generated/**": "managed_block"},
		InferredCapabilities: []string{"inspect_repo"},
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
		Conflicts:            []string{},
		SuggestedNextActions: []string{"accept defaults"},
	}
	if err := ValidateOnboardingProposal(bundle); err == nil {
		t.Fatalf("expected onboarding validation error for invalid mode")
	}
}

func TestRunOutputValidation(t *testing.T) {
	out := map[string]any{
		"status":          "ok",
		"path":            "normal",
		"project_root":    "/tmp/repo",
		"canonical_state": "onboarding_complete",
		"interface": map[string]any{
			"interface_id": "generic-cli",
			"source":       "flag",
		},
		"apply_requested": false,
		"approved":        false,
		"message":         "inspect pipeline executed",
		"portability": map[string]any{
			"supported_targets":   []any{"agents_md"},
			"degraded_targets":    []any{},
			"unsupported_targets": []any{},
			"ingest_plan":         []any{"canonical_policy"},
			"projection_plan": []any{
				map[string]any{
					"requested_target": "agents_md",
					"effective_target": "agents_md",
					"status":           "supported",
					"reason":           "binding supports requested target",
				},
			},
			"portability_status": "supported",
			"degrade_policy":     "proposal_only",
		},
		"resolved_projection_targets": []any{"agents_md"},
		"degraded_surfaces":           []any{},
		"missing_projection_targets":  []any{},
		"portability_status":          "supported",
		"portability_reason":          "requested targets supported without degradation",
		"inspect_bundle":  map[string]any{},
		"planned_mutations": []any{
			map[string]any{
				"envelope": map[string]any{
					"decision_id":         "id-1",
					"phase":               "propose",
					"target_path":         ".atrakta/generated/repo-map.generated.json",
					"scope":               "generated_projection",
					"policy":              "append/include preferred",
					"allowed_modes":       []any{"inspect", "propose", "apply"},
					"requested_action":    "propose",
					"allowed":             true,
					"reason":              "ok",
					"evidence":            []any{"scope=generated_projection"},
					"next_allowed_action": "propose",
				},
				"proposed_patch": "patch",
			},
		},
	}
	if err := ValidateRunOutput(out); err != nil {
		t.Fatalf("validate run output: %v", err)
	}
}

func TestRunOutputValidationRejectsMissingInterfaceID(t *testing.T) {
	raw := []byte(`{
  "status":"ok",
  "path":"normal",
  "project_root":"/tmp/repo",
  "canonical_state":"onboarding_complete",
  "interface":{"source":"flag"},
  "apply_requested":false,
  "approved":false,
  "message":"inspect pipeline executed"
}`)
	if err := ValidateRunOutputRaw(raw); err == nil {
		t.Fatalf("expected run output validation error")
	}
}

func TestMutationProposalValidation(t *testing.T) {
	p := map[string]any{
		"envelope": map[string]any{
			"decision_id":         "id-1",
			"phase":               "propose",
			"target_path":         ".atrakta/generated/repo-map.generated.json",
			"scope":               "generated_projection",
			"policy":              "append/include preferred",
			"allowed_modes":       []any{"inspect", "propose", "apply"},
			"requested_action":    "propose",
			"allowed":             true,
			"reason":              "ok",
			"evidence":            []any{"scope=generated_projection"},
			"next_allowed_action": "propose",
		},
		"proposed_patch": "patch",
	}
	if err := ValidateMutationProposal(p); err != nil {
		t.Fatalf("validate mutation proposal: %v", err)
	}
}
