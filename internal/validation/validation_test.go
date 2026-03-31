package validation

import (
	"encoding/json"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/fixtures"
	"github.com/mash4649/atrakta/v0/internal/harnessprofile"
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

func TestHarnessProfileValidation(t *testing.T) {
	report, err := harnessprofile.Generate(t.TempDir(), "current")
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}
	if err := ValidateHarnessProfile(report); err != nil {
		t.Fatalf("validate harness profile: %v", err)
	}
}

func TestHarnessProfileValidationRejectsAdditionalProperty(t *testing.T) {
	report, err := harnessprofile.Generate(t.TempDir(), "current")
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	out["unexpected"] = true
	if err := ValidateHarnessProfile(out); err == nil {
		t.Fatalf("expected harness profile validation error")
	}
}

func TestStartLatencyBenchmarkValidation(t *testing.T) {
	report := map[string]any{
		"schema_version": "start_latency_benchmark.v1",
		"scenario":       "start_fast_path",
		"command":        "start",
		"interface_id":   "generic-cli",
		"project_root":   "/tmp/bench",
		"iterations":     1,
		"samples": []any{
			map[string]any{
				"iteration":   1,
				"duration_ms": 12,
				"exit_code":   0,
				"fast_path":   true,
			},
		},
		"average_ms":     12,
		"min_ms":         12,
		"max_ms":         12,
		"median_ms":      12,
		"fast_path_hits": 1,
	}
	if err := ValidateStartLatencyBenchmark(report); err != nil {
		t.Fatalf("validate start latency benchmark: %v", err)
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
		"inspect_bundle":              map[string]any{},
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

func TestRunOutputValidationAcceptsErrorEnvelope(t *testing.T) {
	out := map[string]any{
		"status":          "error",
		"path":            "normal",
		"project_root":    "/tmp/repo",
		"canonical_state": "onboarding_complete",
		"interface": map[string]any{
			"interface_id": "generic-cli",
			"source":       "flag",
		},
		"apply_requested": false,
		"approved":        false,
		"message":         "command failed",
		"portability": map[string]any{
			"supported_targets":   []any{},
			"degraded_targets":    []any{},
			"unsupported_targets": []any{},
			"ingest_plan":         []any{},
			"projection_plan":     []any{},
			"portability_status":  "unsupported",
			"degrade_policy":      "proposal_only",
		},
		"resolved_projection_targets": []any{},
		"degraded_surfaces":           []any{},
		"missing_projection_targets":  []any{},
		"portability_status":          "unsupported",
		"portability_reason":          "command failed before portability evaluation",
		"error": map[string]any{
			"code":    "ERR_USAGE",
			"message": "wrap requires install, uninstall, or run",
			"recovery_steps": []any{
				"Run `atrakta wrap --help` to inspect the supported subcommands.",
			},
		},
	}
	if err := ValidateRunOutput(out); err != nil {
		t.Fatalf("validate error output: %v", err)
	}
}

func TestStartOutputValidationReusesRunSchema(t *testing.T) {
	out := map[string]any{
		"status":          "ok",
		"path":            "normal",
		"project_root":    "/tmp/repo",
		"canonical_state": "onboarding_complete",
		"interface": map[string]any{
			"interface_id": "generic-cli",
			"source":       "auto",
		},
		"apply_requested": false,
		"approved":        false,
		"message":         "start fast-path used; workspace unchanged",
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
	}
	if err := ValidateStartOutput(out); err != nil {
		t.Fatalf("validate start output: %v", err)
	}
}

func TestMachineContractValidation(t *testing.T) {
	contract := map[string]any{
		"v":          1,
		"project_id": "test",
		"interfaces": map[string]any{
			"supported": []any{"generic-cli"},
			"fallback":  "generic-cli",
		},
		"boundary": map[string]any{
			"managed_root": ".atrakta/",
		},
		"tools": map[string]any{
			"allow": []any{"create", "edit", "run"},
		},
		"security": map[string]any{
			"destructive":      "deny",
			"external_send":    "deny",
			"approval":         "explicit",
			"permission_model": "proposal_only",
		},
		"routing": map[string]any{
			"default": map[string]any{"worker": "general"},
		},
	}
	if err := ValidateMachineContract(contract); err != nil {
		t.Fatalf("validate machine contract: %v", err)
	}
}

func TestMachineContractValidationRejectsMissingFallback(t *testing.T) {
	raw := []byte(`{
  "v": 1,
  "interfaces": {"supported": ["generic-cli"]},
  "boundary": {"managed_root": ".atrakta/"},
  "tools": {"allow": ["create", "edit", "run"]},
  "security": {"destructive":"deny","external_send":"deny","approval":"explicit","permission_model":"proposal_only"},
  "routing": {"default": {"worker": "general"}}
}`)
	if err := ValidateMachineContractRaw(raw); err == nil {
		t.Fatalf("expected machine contract validation error")
	}
}

func TestMachineContractValidationRejectsUnsafeSecurityModel(t *testing.T) {
	raw := []byte(`{
  "v": 1,
  "interfaces": {"supported": ["generic-cli"], "fallback": "generic-cli"},
  "boundary": {"managed_root": ".atrakta/"},
  "tools": {"allow": ["create", "edit", "run"]},
  "security": {"destructive":"allow","external_send":"deny","approval":"explicit","permission_model":"proposal_only"},
  "routing": {"default": {"worker": "general"}}
}`)
	if err := ValidateMachineContractRaw(raw); err == nil {
		t.Fatalf("expected unsafe security model validation error")
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
