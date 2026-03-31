package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/fixtures"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
	"github.com/mash4649/atrakta/v0/resolvers/common"
	resolveextensionorder "github.com/mash4649/atrakta/v0/resolvers/extension/resolve-extension-order"
	resolveguidanceprecedence "github.com/mash4649/atrakta/v0/resolvers/guidance/resolve-guidance-precedence"
	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
)

var allowedNextActions = map[string]struct{}{
	"inspect":  {},
	"preview":  {},
	"simulate": {},
	"propose":  {},
	"apply":    {},
	"deny":     {},
}

const (
	schemaBundleInput      = "schemas/operations/bundle-input.schema.json"
	schemaBundleOutput     = "schemas/operations/bundle-output.schema.json"
	schemaFixturesReport   = "schemas/operations/fixtures-report.schema.json"
	schemaOnboardingBundle = "schemas/operations/onboarding-proposal-bundle.schema.json"
	schemaSessionState     = "schemas/operations/session-state.schema.json"
	schemaSessionProgress  = "schemas/operations/session-progress.schema.json"
	schemaMutationEnvelope = "schemas/operations/mutation-decision-envelope.schema.json"
	schemaMutationProposal = "schemas/operations/mutation-proposal.schema.json"
	schemaRunOutput        = "schemas/operations/run-output.schema.json"
	schemaMachineContract  = "schemas/operations/machine-contract.schema.json"
	schemaHarnessProfile   = "schemas/operations/harness-profile.schema.json"
	schemaStartLatency     = "schemas/operations/start-latency-benchmark.schema.json"
)

// DecodeAndValidateBundleInput decodes raw JSON and validates the pipeline input contract.
func DecodeAndValidateBundleInput(raw []byte) (pipeline.BundleInput, error) {
	if err := validateJSONAgainstSchema(raw, schemaBundleInput); err != nil {
		return pipeline.BundleInput{}, err
	}

	var input pipeline.BundleInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return pipeline.BundleInput{}, err
	}
	return input, ValidateBundleInput(input)
}

// ValidateBundleInput validates the bundle input contract.
func ValidateBundleInput(in pipeline.BundleInput) error {
	errs := make([]string, 0)

	requireString(&errs, "layer_item.kind", in.LayerItem.Kind)
	requireString(&errs, "layer_item.schema_id", in.LayerItem.SchemaID)
	validateGuidanceItems(&errs, in.GuidanceItems)
	requireString(&errs, "projection_source.type", in.ProjectionSource.Type)
	requireString(&errs, "failure_class", in.FailureClass)
	requireString(&errs, "failure_context.scope", in.FailureContext.Scope)
	requireString(&errs, "mutation_target.path", in.MutationTarget.Path)
	requireString(&errs, "legacy_asset.asset_id", in.LegacyAsset.AssetID)
	requireString(&errs, "legacy_asset.ownership", in.LegacyAsset.Ownership)
	requireString(&errs, "legacy_asset.freshness", in.LegacyAsset.Freshness)
	requireString(&errs, "legacy_asset.canonical_mapping", in.LegacyAsset.CanonicalMapping)
	requireString(&errs, "operation_input.command_or_alias", in.OperationInput.CommandOrAlias)
	validateExtensionItems(&errs, in.ExtensionItems)
	validatePortabilityInput(&errs, in.PortabilityInput)
	requireString(&errs, "audit_input.action", in.AuditInput.Action)
	requireString(&errs, "strict_input.current_state", in.StrictInput.CurrentState)
	requireString(&errs, "strict_input.scope", in.StrictInput.Scope)

	if len(errs) > 0 {
		return fmt.Errorf("bundle input validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateBundleOutput validates the bundle output contract.
func ValidateBundleOutput(out pipeline.BundleOutput) error {
	raw, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("bundle output marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaBundleOutput); err != nil {
		return err
	}

	errs := make([]string, 0)

	requireString(&errs, "mode", out.Mode)
	if out.Steps == nil || len(out.Steps) == 0 {
		errs = append(errs, "steps: required at least one step")
	}
	for i, step := range out.Steps {
		if strings.TrimSpace(step.Name) == "" {
			errs = append(errs, fmt.Sprintf("steps[%d].name: required", i))
		}
		if step.Output.Input == nil {
			errs = append(errs, fmt.Sprintf("steps[%d].output.input: required", i))
		}
		if step.Output.Decision == nil {
			errs = append(errs, fmt.Sprintf("steps[%d].output.decision: required", i))
		}
		requireString(&errs, fmt.Sprintf("steps[%d].output.reason", i), step.Output.Reason)
		if step.Output.Evidence == nil {
			errs = append(errs, fmt.Sprintf("steps[%d].output.evidence: required", i))
		}
		if _, ok := allowedNextActions[step.Output.NextAllowedAction]; !ok {
			errs = append(errs, fmt.Sprintf("steps[%d].output.next_allowed_action: invalid %q", i, step.Output.NextAllowedAction))
		}
	}
	if _, ok := allowedNextActions[out.FinalAllowedAction]; !ok {
		errs = append(errs, fmt.Sprintf("final_allowed_action: invalid %q", out.FinalAllowedAction))
	}

	if len(errs) > 0 {
		return fmt.Errorf("bundle output validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateFixtureReport validates the fixture runner report.
func ValidateFixtureReport(report fixtures.Report) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("fixture report marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaFixturesReport); err != nil {
		return err
	}

	if report.Passed < 0 || report.Failed < 0 {
		return fmt.Errorf("fixture report validation failed: negative counters")
	}
	for i, result := range report.Results {
		if strings.TrimSpace(result.Fixture) == "" {
			return fmt.Errorf("fixture report validation failed: results[%d].fixture required", i)
		}
		if result.Case < 0 {
			return fmt.Errorf("fixture report validation failed: results[%d].case invalid", i)
		}
	}
	return nil
}

// ValidateOnboardingProposal validates the onboarding proposal bundle contract.
func ValidateOnboardingProposal(bundle onboarding.ProposalBundle) error {
	raw, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("onboarding proposal marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaOnboardingBundle); err != nil {
		return err
	}

	errs := make([]string, 0)
	requireString(&errs, "inferred_mode", bundle.InferredMode)
	if _, err := onboarding.MustMode(bundle.InferredMode); err != nil {
		errs = append(errs, err.Error())
	}
	if len(bundle.InferredManagedScope) == 0 {
		errs = append(errs, "inferred_managed_scope: required at least one item")
	}
	if bundle.DetectedRisks == nil {
		errs = append(errs, "detected_risks: required array")
	}
	if len(bundle.InferredCapabilities) == 0 {
		errs = append(errs, "inferred_capabilities: required at least one item")
	}
	if len(bundle.InferredGuidance) == 0 {
		errs = append(errs, "inferred_guidance_strength: required at least one item")
	}
	if len(bundle.InferredDefaultPolicy) == 0 {
		errs = append(errs, "inferred_default_policy: required at least one item")
	}
	requireString(&errs, "inferred_failure_routing.failure_class", bundle.InferredFailure.FailureClass)
	requireString(&errs, "inferred_failure_routing.scope", bundle.InferredFailure.Scope)
	requireString(&errs, "inferred_failure_routing.default_tier", bundle.InferredFailure.DefaultTier)
	requireString(&errs, "inferred_failure_routing.resolved_tier", bundle.InferredFailure.ResolvedTier)
	requireString(&errs, "inferred_failure_routing.strict_transition", bundle.InferredFailure.StrictTransition)
	if _, ok := allowedNextActions[bundle.InferredFailure.NextAllowedAction]; !ok {
		errs = append(errs, fmt.Sprintf("inferred_failure_routing.next_allowed_action invalid: %q", bundle.InferredFailure.NextAllowedAction))
	}
	if len(bundle.SuggestedNextActions) == 0 {
		errs = append(errs, "suggested_next_actions: required at least one item")
	}
	if len(errs) > 0 {
		return fmt.Errorf("onboarding proposal validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateHarnessProfile validates the harness profile report contract.
func ValidateHarnessProfile(report any) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("harness profile marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaHarnessProfile); err != nil {
		return err
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("harness profile decode failed: %w", err)
	}

	errs := make([]string, 0)
	requireNonEmptyStringAny(&errs, "schema_version", out["schema_version"])
	requireNonEmptyStringAny(&errs, "project_root", out["project_root"])
	requireNonEmptyStringAny(&errs, "model_generation", out["model_generation"])
	requireNonEmptyStringAny(&errs, "profile_name", out["profile_name"])
	requireNonNegativeIntAny(&errs, "threshold", out["threshold"])
	requireNonNegativeIntAny(&errs, "baseline_score", out["baseline_score"])
	if steps, ok := out["steps"].([]any); !ok || len(steps) == 0 {
		errs = append(errs, "steps: required at least one step")
	}
	if ablations, ok := out["ablations"].([]any); !ok || len(ablations) == 0 {
		errs = append(errs, "ablations: required at least one ablation")
	}
	if comps, ok := out["load_bearing_components"].([]any); !ok || len(comps) == 0 {
		errs = append(errs, "load_bearing_components: required at least one item")
	}
	if len(errs) > 0 {
		return fmt.Errorf("harness profile validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateStartLatencyBenchmark validates the start latency benchmark report contract.
func ValidateStartLatencyBenchmark(report any) error {
	raw, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("start latency benchmark marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaStartLatency); err != nil {
		return err
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("start latency benchmark decode failed: %w", err)
	}

	errs := make([]string, 0)
	requireNonEmptyStringAny(&errs, "schema_version", out["schema_version"])
	requireNonEmptyStringAny(&errs, "scenario", out["scenario"])
	requireNonEmptyStringAny(&errs, "command", out["command"])
	requireNonEmptyStringAny(&errs, "interface_id", out["interface_id"])
	requireNonEmptyStringAny(&errs, "project_root", out["project_root"])
	requireNonNegativeIntAny(&errs, "iterations", out["iterations"])
	if samples, ok := out["samples"].([]any); !ok || len(samples) == 0 {
		errs = append(errs, "samples: required at least one sample")
	}
	requireNonNegativeIntAny(&errs, "average_ms", out["average_ms"])
	requireNonNegativeIntAny(&errs, "min_ms", out["min_ms"])
	requireNonNegativeIntAny(&errs, "max_ms", out["max_ms"])
	requireNonNegativeIntAny(&errs, "median_ms", out["median_ms"])
	requireNonNegativeIntAny(&errs, "fast_path_hits", out["fast_path_hits"])
	if len(errs) > 0 {
		return fmt.Errorf("start latency benchmark validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateSessionStateRaw validates the session state persistence payload.
func ValidateSessionStateRaw(raw []byte) error {
	return validateJSONAgainstSchema(raw, schemaSessionState)
}

// ValidateSessionProgressRaw validates the session progress persistence payload.
func ValidateSessionProgressRaw(raw []byte) error {
	return validateJSONAgainstSchema(raw, schemaSessionProgress)
}

// ValidateRunOutput validates the run command output contract.
func ValidateRunOutput(payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("run output marshal failed: %w", err)
	}
	return ValidateRunOutputRaw(raw)
}

// ValidateRunOutputRaw validates run output from raw JSON bytes.
func ValidateRunOutputRaw(raw []byte) error {
	if err := validateJSONAgainstSchema(raw, schemaRunOutput); err != nil {
		return err
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("run output decode failed: %w", err)
	}

	errs := make([]string, 0)
	requireNonEmptyStringAny(&errs, "status", out["status"])
	requireNonEmptyStringAny(&errs, "path", out["path"])
	requireNonEmptyStringAny(&errs, "project_root", out["project_root"])
	requireNonEmptyStringAny(&errs, "canonical_state", out["canonical_state"])
	requireNonEmptyStringAny(&errs, "message", out["message"])
	if status, _ := out["status"].(string); status == "error" && out["error"] == nil {
		errs = append(errs, "error: required when status is error")
	}

	if ifaceRaw, ok := out["interface"]; ok {
		iface, ok := ifaceRaw.(map[string]any)
		if !ok {
			errs = append(errs, "interface: invalid object")
		} else {
			requireNonEmptyStringAny(&errs, "interface.interface_id", iface["interface_id"])
			requireNonEmptyStringAny(&errs, "interface.source", iface["source"])
		}
	}

	if v, ok := out["next_allowed_action"]; ok {
		if s, ok := v.(string); !ok || strings.TrimSpace(s) == "" {
			errs = append(errs, "next_allowed_action: invalid value")
		}
	}
	if v, ok := out["required_inputs"]; ok {
		items, ok := v.([]any)
		if !ok {
			errs = append(errs, "required_inputs: invalid array")
		} else {
			for i, item := range items {
				s, ok := item.(string)
				if !ok || strings.TrimSpace(s) == "" {
					errs = append(errs, fmt.Sprintf("required_inputs[%d]: required", i))
				}
			}
		}
	}
	if v, ok := out["approval_scope"]; ok {
		if s, ok := v.(string); !ok || strings.TrimSpace(s) == "" {
			errs = append(errs, "approval_scope: invalid value")
		}
	}
	if v, ok := out["error"]; ok {
		if err := validateStructuredErrorValue(v); err != nil {
			errs = append(errs, fmt.Sprintf("error: %v", err))
		}
	}
	if v, ok := out["planned_mutations"]; ok {
		items, ok := v.([]any)
		if !ok {
			errs = append(errs, "planned_mutations: invalid array")
		} else {
			for i, item := range items {
				if err := ValidateMutationProposal(item); err != nil {
					errs = append(errs, fmt.Sprintf("planned_mutations[%d]: %v", i, err))
				}
			}
		}
	}
	if v, ok := out["applied_mutations"]; ok {
		items, ok := v.([]any)
		if !ok {
			errs = append(errs, "applied_mutations: invalid array")
		} else {
			for i, item := range items {
				if err := ValidateMutationDecisionEnvelope(item); err != nil {
					errs = append(errs, fmt.Sprintf("applied_mutations[%d]: %v", i, err))
				}
			}
		}
	}
	if v, ok := out["portability"]; ok {
		portability, ok := v.(map[string]any)
		if !ok {
			errs = append(errs, "portability: invalid object")
		} else {
			validatePortabilityDecision(&errs, portability)
		}
	} else {
		errs = append(errs, "portability: required")
	}
	validateStringArrayField(&errs, out, "resolved_projection_targets")
	validateStringArrayField(&errs, out, "degraded_surfaces")
	validateStringArrayField(&errs, out, "missing_projection_targets")
	requireNonEmptyStringAny(&errs, "portability_status", out["portability_status"])
	requireNonEmptyStringAny(&errs, "portability_reason", out["portability_reason"])

	if len(errs) > 0 {
		return fmt.Errorf("run output validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

func validateStructuredErrorValue(v any) error {
	obj, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid object")
	}
	if s, ok := obj["code"].(string); !ok || strings.TrimSpace(s) == "" {
		return fmt.Errorf("code required")
	}
	if s, ok := obj["message"].(string); !ok || strings.TrimSpace(s) == "" {
		return fmt.Errorf("message required")
	}
	if stepsRaw, ok := obj["recovery_steps"]; ok {
		steps, ok := stepsRaw.([]any)
		if !ok {
			return fmt.Errorf("recovery_steps invalid array")
		}
		for i, step := range steps {
			if s, ok := step.(string); !ok || strings.TrimSpace(s) == "" {
				return fmt.Errorf("recovery_steps[%d] required", i)
			}
		}
	}
	return nil
}

// ValidateStartOutput validates the start command output contract.
// Phase 1 decision: start reuses run-output schema.
func ValidateStartOutput(payload any) error {
	return ValidateRunOutput(payload)
}

// ValidateStartOutputRaw validates start output from raw JSON bytes.
// Phase 1 decision: start reuses run-output schema.
func ValidateStartOutputRaw(raw []byte) error {
	return ValidateRunOutputRaw(raw)
}

// ValidateMachineContract validates the machine contract payload.
func ValidateMachineContract(payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("machine contract marshal failed: %w", err)
	}
	return ValidateMachineContractRaw(raw)
}

// ValidateMachineContractRaw validates machine contract from raw JSON bytes.
func ValidateMachineContractRaw(raw []byte) error {
	if err := validateJSONAgainstSchema(raw, schemaMachineContract); err != nil {
		return err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("machine contract decode failed: %w", err)
	}
	errs := make([]string, 0)
	requireNonEmptyStringAny(&errs, "interfaces.fallback", nestedObjectValue(out, "interfaces", "fallback"))
	requireNonEmptyStringAny(&errs, "boundary.managed_root", nestedObjectValue(out, "boundary", "managed_root"))
	requireNonEmptyStringAny(&errs, "security.destructive", nestedObjectValue(out, "security", "destructive"))
	requireNonEmptyStringAny(&errs, "security.external_send", nestedObjectValue(out, "security", "external_send"))
	requireNonEmptyStringAny(&errs, "security.approval", nestedObjectValue(out, "security", "approval"))
	requireNonEmptyStringAny(&errs, "security.permission_model", nestedObjectValue(out, "security", "permission_model"))

	if allow, ok := nestedArrayValue(out, "tools", "allow"); ok {
		if len(allow) == 0 {
			errs = append(errs, "tools.allow: required at least one item")
		}
		allowedTools := map[string]struct{}{
			"create": {},
			"edit":   {},
			"run":    {},
		}
		for i, item := range allow {
			s, ok := item.(string)
			if !ok || strings.TrimSpace(s) == "" {
				errs = append(errs, fmt.Sprintf("tools.allow[%d]: required string", i))
				continue
			}
			if _, ok := allowedTools[strings.TrimSpace(s)]; !ok {
				errs = append(errs, "tools.allow: unsafe tool disallowed: "+strings.TrimSpace(s))
			}
		}
	}

	if destructive, ok := nestedStringValue(out, "security", "destructive"); ok && destructive != "deny" {
		errs = append(errs, "security.destructive: must be deny")
	}
	if externalSend, ok := nestedStringValue(out, "security", "external_send"); ok && externalSend != "deny" {
		errs = append(errs, "security.external_send: must be deny")
	}
	if approval, ok := nestedStringValue(out, "security", "approval"); ok && approval != "explicit" {
		errs = append(errs, "security.approval: must be explicit")
	}
	if model, ok := nestedStringValue(out, "security", "permission_model"); ok && model != "proposal_only" {
		errs = append(errs, "security.permission_model: must be proposal_only")
	}
	if len(errs) > 0 {
		return fmt.Errorf("machine contract validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

// ValidateMutationDecisionEnvelope validates mutation decision envelope contract.
func ValidateMutationDecisionEnvelope(payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mutation decision envelope marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaMutationEnvelope); err != nil {
		return err
	}
	return nil
}

// ValidateMutationProposal validates mutation proposal contract.
func ValidateMutationProposal(payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mutation proposal marshal failed: %w", err)
	}
	if err := validateJSONAgainstSchema(raw, schemaMutationProposal); err != nil {
		return err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return fmt.Errorf("mutation proposal decode failed: %w", err)
	}
	envelope, ok := out["envelope"]
	if !ok {
		return fmt.Errorf("mutation proposal envelope required")
	}
	if err := ValidateMutationDecisionEnvelope(envelope); err != nil {
		return fmt.Errorf("mutation proposal envelope invalid: %w", err)
	}
	return nil
}

func requireString(errs *[]string, field, value string) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, field+": required")
	}
}

func requireNonEmptyStringAny(errs *[]string, field string, value any) {
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		*errs = append(*errs, field+": required")
	}
}

func requireNonNegativeIntAny(errs *[]string, field string, value any) {
	n, ok := value.(float64)
	if !ok || n < 0 || math.Trunc(n) != n {
		*errs = append(*errs, field+": required non-negative integer")
	}
}

func nestedObjectValue(root map[string]any, objectKey, fieldKey string) any {
	objRaw, ok := root[objectKey]
	if !ok {
		return nil
	}
	obj, ok := objRaw.(map[string]any)
	if !ok {
		return nil
	}
	return obj[fieldKey]
}

func nestedStringValue(root map[string]any, objectKey, fieldKey string) (string, bool) {
	v := nestedObjectValue(root, objectKey, fieldKey)
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(s), true
}

func nestedArrayValue(root map[string]any, objectKey, fieldKey string) ([]any, bool) {
	v := nestedObjectValue(root, objectKey, fieldKey)
	arr, ok := v.([]any)
	if !ok {
		return nil, false
	}
	return arr, true
}

func validateGuidanceItems(errs *[]string, items []resolveguidanceprecedence.GuidanceItem) {
	for i, item := range items {
		requireString(errs, fmt.Sprintf("guidance_items[%d].id", i), item.ID)
		requireString(errs, fmt.Sprintf("guidance_items[%d].type", i), item.Type)
	}
}

func validateExtensionItems(errs *[]string, items []resolveextensionorder.Item) {
	for i, item := range items {
		requireString(errs, fmt.Sprintf("extension_items[%d].id", i), item.ID)
		requireString(errs, fmt.Sprintf("extension_items[%d].kind", i), item.Kind)
	}
}

func validatePortabilityInput(errs *[]string, in resolvesurfaceportability.Input) {
	requireString(errs, "portability_input.interface_id", in.InterfaceID)
	if len(in.RequestedTargets) == 0 {
		*errs = append(*errs, "portability_input.requested_targets: required at least one item")
	}
	for i, item := range in.RequestedTargets {
		requireString(errs, fmt.Sprintf("portability_input.requested_targets[%d]", i), item)
	}
	if len(in.AvailableSources) == 0 {
		*errs = append(*errs, "portability_input.available_sources: required at least one item")
	}
	for i, item := range in.AvailableSources {
		requireString(errs, fmt.Sprintf("portability_input.available_sources[%d]", i), item)
	}
	requireString(errs, "portability_input.binding_capabilities.id", in.BindingCapabilities.InterfaceID)
	requireString(errs, "portability_input.binding_capabilities.approval_channel", in.BindingCapabilities.ApprovalChannel)
	requireString(errs, "portability_input.binding_capabilities.portability_mode", in.BindingCapabilities.PortabilityMode)
	requireString(errs, "portability_input.degrade_policy", in.DegradePolicy)
}

func validatePortabilityDecision(errs *[]string, portability map[string]any) {
	validateStringArrayAny(errs, "portability.supported_targets", portability["supported_targets"])
	validateStringArrayAny(errs, "portability.degraded_targets", portability["degraded_targets"])
	validateStringArrayAny(errs, "portability.unsupported_targets", portability["unsupported_targets"])
	validateStringArrayAny(errs, "portability.ingest_plan", portability["ingest_plan"])
	requireNonEmptyStringAny(errs, "portability.portability_status", portability["portability_status"])
	requireNonEmptyStringAny(errs, "portability.degrade_policy", portability["degrade_policy"])
	if v, ok := portability["projection_plan"]; !ok {
		*errs = append(*errs, "portability.projection_plan: required")
	} else if _, ok := v.([]any); !ok {
		*errs = append(*errs, "portability.projection_plan: invalid array")
	}
}

func validateStringArrayField(errs *[]string, out map[string]any, field string) {
	validateStringArrayAny(errs, field, out[field])
}

func validateStringArrayAny(errs *[]string, field string, value any) {
	items, ok := value.([]any)
	if !ok {
		*errs = append(*errs, field+": invalid array")
		return
	}
	for i, item := range items {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			*errs = append(*errs, fmt.Sprintf("%s[%d]: required", field, i))
		}
	}
}

// ValidateMode is a narrow helper for command validation.
func ValidateMode(mode string) error {
	switch mode {
	case "inspect", "preview", "simulate":
		return nil
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
}

// ValidateResolverOutput is a helper for contract tests and CLI hooks.
func ValidateResolverOutput(out common.ResolverOutput) error {
	if out.Input == nil {
		return fmt.Errorf("resolver output input required")
	}
	if out.Decision == nil {
		return fmt.Errorf("resolver output decision required")
	}
	if strings.TrimSpace(out.Reason) == "" {
		return fmt.Errorf("resolver output reason required")
	}
	if out.Evidence == nil {
		return fmt.Errorf("resolver output evidence required")
	}
	if _, ok := allowedNextActions[out.NextAllowedAction]; !ok {
		return fmt.Errorf("resolver output next_allowed_action invalid: %q", out.NextAllowedAction)
	}
	return nil
}

// ValidateResolverContracts ensures the core resolver outputs used in CLI remain valid.
func ValidateResolverContracts(outputs ...common.ResolverOutput) error {
	for i, out := range outputs {
		if err := ValidateResolverOutput(out); err != nil {
			return fmt.Errorf("resolver[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidatePipelineContext ensures the supporting typed inputs are populated enough for CLI execution.
func ValidatePipelineContext(in pipeline.BundleInput) error {
	return ValidateBundleInput(in)
}

// ValidatePipelineOutput ensures the CLI export-ready bundle is valid.
func ValidatePipelineOutput(out pipeline.BundleOutput) error {
	return ValidateBundleOutput(out)
}
