package pipeline

import (
	"encoding/json"
	"fmt"

	resolveauditrequirements "github.com/mash4649/atrakta/v0/resolvers/audit/resolve-audit-requirements"
	"github.com/mash4649/atrakta/v0/resolvers/common"
	resolveextensionorder "github.com/mash4649/atrakta/v0/resolvers/extension/resolve-extension-order"
	resolvefailuretier "github.com/mash4649/atrakta/v0/resolvers/failure/resolve-failure-tier"
	strictstatemachine "github.com/mash4649/atrakta/v0/resolvers/failure/strict-state-machine"
	resolveguidanceprecedence "github.com/mash4649/atrakta/v0/resolvers/guidance/resolve-guidance-precedence"
	classifylayer "github.com/mash4649/atrakta/v0/resolvers/layer/classify-layer"
	resolvelegacystatus "github.com/mash4649/atrakta/v0/resolvers/legacy/resolve-legacy-status"
	checkmutationscope "github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
	resolveoperationcapability "github.com/mash4649/atrakta/v0/resolvers/operations/resolve-operation-capability"
	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
	checkprojectioneligibility "github.com/mash4649/atrakta/v0/resolvers/projection/check-projection-eligibility"
)

// BundleInput is a full ordered resolver pipeline input.
type BundleInput struct {
	LayerItem        classifylayer.Item                       `json:"layer_item"`
	GuidanceItems    []resolveguidanceprecedence.GuidanceItem `json:"guidance_items"`
	ProjectionSource checkprojectioneligibility.Source        `json:"projection_source"`
	FailureClass     string                                   `json:"failure_class"`
	FailureContext   resolvefailuretier.Context               `json:"failure_context"`
	MutationTarget   checkmutationscope.Target                `json:"mutation_target"`
	LegacyAsset      resolvelegacystatus.Asset                `json:"legacy_asset"`
	OperationInput   resolveoperationcapability.Input         `json:"operation_input"`
	ExtensionItems   []resolveextensionorder.Item             `json:"extension_items"`
	PortabilityInput resolvesurfaceportability.Input          `json:"portability_input"`
	AuditInput       resolveauditrequirements.Input           `json:"audit_input"`
	StrictInput      strictstatemachine.StateInput            `json:"strict_input"`
}

// StepResult is one resolver output in ordered execution.
type StepResult struct {
	Name   string                `json:"name"`
	Output common.ResolverOutput `json:"output"`
}

// BundleOutput is inspect/preview/simulate bundle output.
type BundleOutput struct {
	Mode               string       `json:"mode"`
	Steps              []StepResult `json:"steps"`
	FinalAllowedAction string       `json:"final_allowed_action"`
}

// ExecuteOrdered executes resolvers in deterministic order.
func ExecuteOrdered(mode string, in BundleInput) (BundleOutput, error) {
	mode = normalizeMode(mode)
	if mode == "" {
		return BundleOutput{}, fmt.Errorf("unsupported mode")
	}

	steps := make([]StepResult, 0, 11)

	layerOut := classifylayer.ClassifyLayer(in.LayerItem)
	steps = append(steps, StepResult{Name: "classify_layer", Output: layerOut})

	guidanceOut := resolveguidanceprecedence.ResolveGuidancePrecedence(in.GuidanceItems)
	steps = append(steps, StepResult{Name: "resolve_guidance_precedence", Output: guidanceOut})

	projectionOut := checkprojectioneligibility.CheckProjectionEligibility(in.ProjectionSource)
	steps = append(steps, StepResult{Name: "check_projection_eligibility", Output: projectionOut})

	failureOut := resolvefailuretier.ResolveFailureTier(in.FailureClass, in.FailureContext)
	steps = append(steps, StepResult{Name: "resolve_failure_tier", Output: failureOut})

	failureDecision, ok := failureOut.Decision.(resolvefailuretier.FailureDecision)
	if !ok {
		return BundleOutput{}, fmt.Errorf("unexpected failure decision type")
	}

	strictIn := in.StrictInput
	if strictIn.CurrentState == "" {
		strictIn.CurrentState = "normal"
	}
	if strictIn.Scope == "" {
		strictIn.Scope = failureDecision.Scope
	}
	if strictIn.Event == "" {
		switch failureDecision.StrictTransition {
		case "strict":
			strictIn.Event = "trigger_strict"
		case "guarded":
			strictIn.Event = "trigger_guarded"
		default:
			strictIn.Event = "reset"
		}
	}
	strictOut := strictstatemachine.Transition(strictIn)
	steps = append(steps, StepResult{Name: "strict_state_machine", Output: strictOut})

	mutationOut := checkmutationscope.CheckMutationScope(in.MutationTarget)
	steps = append(steps, StepResult{Name: "check_mutation_scope", Output: mutationOut})

	legacyOut := resolvelegacystatus.ResolveLegacyStatus(in.LegacyAsset)
	steps = append(steps, StepResult{Name: "resolve_legacy_status", Output: legacyOut})

	opIn := in.OperationInput
	opIn.FailureTier = failureDecision.ResolvedTier
	operationOut := resolveoperationcapability.ResolveOperationCapability(opIn)
	steps = append(steps, StepResult{Name: "resolve_operation_capability", Output: operationOut})

	extensionOut := resolveextensionorder.ResolveExtensionOrder(in.ExtensionItems)
	steps = append(steps, StepResult{Name: "resolve_extension_order", Output: extensionOut})

	portabilityOut := resolvesurfaceportability.ResolveSurfacePortability(in.PortabilityInput)
	steps = append(steps, StepResult{Name: "resolve_surface_portability", Output: portabilityOut})

	auditOut := resolveauditrequirements.ResolveAuditRequirements(in.AuditInput)
	steps = append(steps, StepResult{Name: "resolve_audit_requirements", Output: auditOut})

	final := "apply"
	for _, s := range steps {
		final = moreRestrictive(final, s.Output.NextAllowedAction)
	}
	final = clampByMode(final, mode)

	return BundleOutput{
		Mode:               mode,
		Steps:              steps,
		FinalAllowedAction: final,
	}, nil
}

// DefaultInput returns a safe baseline input for the selected mode.
func DefaultInput(mode string) BundleInput {
	op := "doctor"
	action := "inspect"
	if mode == "preview" || mode == "simulate" {
		op = "repair"
		action = "propose"
	}

	return BundleInput{
		LayerItem: classifylayer.Item{Kind: "request", SchemaID: "atrakta/schemas/core/request.schema.json"},
		GuidanceItems: []resolveguidanceprecedence.GuidanceItem{
			{ID: "policy-1", Type: "policy"},
			{ID: "workflow-1", Type: "workflow"},
			{ID: "skill-1", Type: "skill"},
			{ID: "repo-1", Type: "repo_map"},
		},
		ProjectionSource: checkprojectioneligibility.Source{Type: "policy"},
		FailureClass:     "approval_failure",
		FailureContext:   resolvefailuretier.Context{Scope: "task", Triggers: []string{"missing_approval"}},
		MutationTarget:   checkmutationscope.Target{Path: "AGENTS.md", AssetType: "existing_user_rules", HasAmbiguity: true},
		LegacyAsset:      resolvelegacystatus.Asset{AssetID: "legacy-1", Ownership: "known", Freshness: "acceptable", CanonicalMapping: "exists", Integrity: "known"},
		OperationInput:   resolveoperationcapability.Input{CommandOrAlias: op},
		ExtensionItems: []resolveextensionorder.Item{
			{ID: "policy-1", Kind: "policy"},
			{ID: "workflow-1", Kind: "workflow"},
			{ID: "skill-1", Kind: "skill"},
			{ID: "hint-1", Kind: "tool_hint"},
			{ID: "hook-1", Kind: "hook"},
			{ID: "plugin-1", Kind: "projection_plugin"},
		},
		PortabilityInput: resolvesurfaceportability.Input{
			InterfaceID:      "generic-cli",
			RequestedTargets: []string{"agents_md", "repo_docs"},
			AvailableSources: []string{"canonical_policy", "repo_docs", "agents_md"},
			BindingCapabilities: resolvesurfaceportability.BindingCapabilities{
				InterfaceID:       "generic-cli",
				ProjectionTargets: []string{"agents_md", "repo_docs"},
				IngestSources:     []string{"canonical_policy", "repo_docs", "agents_md"},
				ApprovalChannel:   "cli_flag",
				PortabilityMode:   resolvesurfaceportability.PortabilityModeRequired,
			},
			DegradePolicy: resolvesurfaceportability.DegradePolicyProposalOnly,
		},
		AuditInput:  resolveauditrequirements.Input{Action: action},
		StrictInput: strictstatemachine.StateInput{CurrentState: "normal", Scope: "task"},
	}
}

// MarshalStable marshals pipeline output for deterministic replay checks.
func MarshalStable(out BundleOutput) ([]byte, error) {
	return json.Marshal(out)
}

func normalizeMode(mode string) string {
	switch mode {
	case "inspect", "preview", "simulate":
		return mode
	default:
		return ""
	}
}

func clampByMode(action, mode string) string {
	if action == "deny" {
		return action
	}
	switch mode {
	case "inspect":
		if action == "apply" || action == "propose" || action == "simulate" || action == "preview" {
			return "inspect"
		}
		return action
	case "preview", "simulate":
		if action == "apply" {
			return "propose"
		}
		return action
	default:
		return action
	}
}

func moreRestrictive(current, next string) string {
	if rank(next) < rank(current) {
		return next
	}
	return current
}

func rank(action string) int {
	switch action {
	case "deny":
		return 0
	case "inspect":
		return 1
	case "preview":
		return 2
	case "simulate":
		return 3
	case "propose":
		return 4
	case "apply":
		return 5
	default:
		return 1
	}
}
