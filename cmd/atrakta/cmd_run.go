package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/bindings"
	"github.com/mash4649/atrakta/v0/internal/entry"
	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
	"github.com/mash4649/atrakta/v0/internal/mutation"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/persist"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
	"github.com/mash4649/atrakta/v0/internal/startfast"
	"github.com/mash4649/atrakta/v0/internal/validation"
	resolveauditrequirements "github.com/mash4649/atrakta/v0/resolvers/audit/resolve-audit-requirements"
	"github.com/mash4649/atrakta/v0/resolvers/extension/resolve-extension-order"
	"github.com/mash4649/atrakta/v0/resolvers/failure/resolve-failure-tier"
	strictstatemachine "github.com/mash4649/atrakta/v0/resolvers/failure/strict-state-machine"
	"github.com/mash4649/atrakta/v0/resolvers/guidance/resolve-guidance-precedence"
	"github.com/mash4649/atrakta/v0/resolvers/layer/classify-layer"
	"github.com/mash4649/atrakta/v0/resolvers/legacy/resolve-legacy-status"
	"github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
	"github.com/mash4649/atrakta/v0/resolvers/operations/resolve-operation-capability"
	"github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
	"github.com/mash4649/atrakta/v0/resolvers/projection/check-projection-eligibility"
)

type runResponse struct {
	Status            string                         `json:"status"`
	Path              string                         `json:"path"`
	ProjectRoot       string                         `json:"project_root"`
	CanonicalState    string                         `json:"canonical_state"`
	CanonicalSummary  map[string]any                 `json:"canonical_summary,omitempty"`
	Interface         runpkg.InterfaceResolution     `json:"interface"`
	ApplyRequested    bool                           `json:"apply_requested"`
	Approved          bool                           `json:"approved"`
	Message           string                         `json:"message"`
	Portability       any                            `json:"portability"`
	ResolvedTargets   []string                       `json:"resolved_projection_targets"`
	DegradedSurfaces  []string                       `json:"degraded_surfaces"`
	MissingTargets    []string                       `json:"missing_projection_targets"`
	PortabilityStatus string                         `json:"portability_status"`
	PortabilityReason string                         `json:"portability_reason"`
	Error             *atraktaerrors.StructuredError `json:"error,omitempty"`
	NextAllowedAction string                         `json:"next_allowed_action,omitempty"`
	RequiredInputs    []string                       `json:"required_inputs,omitempty"`
	ApprovalScope     string                         `json:"approval_scope,omitempty"`
	Onboarding        *onboarding.ProposalBundle     `json:"onboarding_proposal,omitempty"`
	AcceptResult      *persist.AcceptResult          `json:"accept_result,omitempty"`
	InspectBundle     *pipeline.BundleOutput         `json:"inspect_bundle,omitempty"`
	PlannedMutations  []mutation.Proposal            `json:"planned_mutations,omitempty"`
	AppliedMutations  []mutation.DecisionEnvelope    `json:"applied_mutations,omitempty"`
}

func runCommand(args []string) (int, error) {
	return runLikeCommand("run", args)
}

func runLikeCommand(commandName string, args []string) (int, error) {
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var interfaceID string
	var artifactDir string
	var jsonOut bool
	var nonInteractive bool
	var apply bool
	var approve bool
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&interfaceID, "interface", "", "runtime interface id")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "disable approval prompt")
	fs.BoolVar(&apply, "apply", false, "request apply route")
	fs.BoolVar(&approve, "approve", false, "explicitly approve write path")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	if !flagWasProvided(fs, "non-interactive") {
		nonInteractive = isTruthyEnv(os.Getenv("ATRAKTA_NONINTERACTIVE"))
	}

	entryCode, entryOut, err := entry.Execute(entry.ExecuteInput{
		ProjectRoot:      projectRoot,
		InterfaceID:      interfaceID,
		TriggerInterface: strings.TrimSpace(os.Getenv("ATRAKTA_TRIGGER_INTERFACE")),
		ApplyRequested:   apply,
	})
	if err != nil {
		return exitRuntimeError, err
	}
	if entryCode == exitRuntimeError && entryOut.HardError != nil {
		resp := buildNeedsInputRunResponse(entryOut.HardError, apply)
		resp.Error = structuredErrorFromNeedsInput(entryOut.HardError.Message)
		if emitErr := emitRunResponse(commandName, resp, jsonOut, artifactDir); emitErr != nil {
			return exitRuntimeError, emitErr
		}
		return exitRuntimeError, atraktaerrors.NewExitError(exitRuntimeError, *resp.Error, true)
	}
	if entryCode == exitNeedsInput && entryOut.NeedsInput != nil {
		if emitErr := emitRunResponse(commandName, buildNeedsInputRunResponse(entryOut.NeedsInput, apply), jsonOut, artifactDir); emitErr != nil {
			return exitRuntimeError, emitErr
		}
		return exitNeedsInput, nil
	}
	resolved := entryOut.Decision
	if isSessionCommand(commandName) {
		beginEvent := runEventStartBegin
		if commandName == "resume" {
			beginEvent = runEventResumeBegin
		}
		if err := appendSessionLifecycleEvent(commandName, resolved.ProjectRoot, beginEvent, map[string]any{
			"path":             resolved.Path,
			"canonical_state":  resolved.CanonicalState,
			"interface_id":     resolved.Interface.InterfaceID,
			"interface_source": resolved.Interface.Source,
			"apply_requested":  apply,
		}); err != nil {
			return exitRuntimeError, err
		}
	}

	switch resolved.Path {
	case entry.PathOnboarding:
		return runOnboardingPath(commandName, resolved.ProjectRoot, resolved.Interface, nonInteractive, approve, jsonOut, artifactDir)
	case entry.PathNormal:
		contract, err := runpkg.LoadMachineContract(resolved.ProjectRoot)
		if err != nil {
			resp := runResponse{
				Status:            "needs_input",
				Path:              "normal",
				ProjectRoot:       resolved.ProjectRoot,
				CanonicalState:    resolved.CanonicalState,
				Interface:         resolved.Interface,
				ApplyRequested:    apply,
				Message:           "machine contract is missing or invalid",
				Portability:       unsupportedPortability("machine contract missing or invalid"),
				ResolvedTargets:   []string{},
				DegradedSurfaces:  []string{},
				MissingTargets:    []string{},
				PortabilityStatus: resolvesurfaceportability.PortabilityUnsupported,
				PortabilityReason: "machine contract missing or invalid",
				NextAllowedAction: "restore .atrakta/contract.json and re-run",
				RequiredInputs:    []string{"machine_contract"},
			}
			if auditErr := appendRunAuditEvent(commandName, resolved.ProjectRoot, map[string]any{
				"status":            resp.Status,
				"path":              resp.Path,
				"interface_id":      resolved.Interface.InterfaceID,
				"interface_source":  resolved.Interface.Source,
				"apply_requested":   apply,
				"contract_load_ok":  false,
				"contract_load_err": err.Error(),
			}); auditErr != nil {
				return exitRuntimeError, auditErr
			}
			if emitErr := emitRunResponse(commandName, resp, jsonOut, artifactDir); emitErr != nil {
				return exitRuntimeError, emitErr
			}
			return exitNeedsInput, nil
		}
		summary, err := loadCanonicalSummary(resolved.ProjectRoot)
		if err != nil {
			return exitRuntimeError, err
		}
		auditPreflight, err := preflightVerifyAudit(resolved.ProjectRoot)
		if err != nil {
			return exitRuntimeError, err
		}
		summary["audit_preflight"] = auditPreflight
		if version, ok := contract["v"].(float64); ok {
			summary["contract_version"] = int(version)
		}
		summary["contract_path"] = filepath.ToSlash(runpkg.ContractPath(resolved.ProjectRoot))
		return runNormalPath(commandName, resolved.ProjectRoot, resolved.CanonicalState, resolved.Interface, summary, apply, approve, nonInteractive, jsonOut, artifactDir)
	default:
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Runtime(
				fmt.Sprintf("unsupported entry path %q", resolved.Path),
				"Inspect the resolved entry path and retry after fixing the project state.",
			),
			false,
		)
	}
}

func hasDetectedAsset(set map[string]struct{}, names ...string) bool {
	for _, name := range names {
		if _, ok := set[name]; ok {
			return true
		}
	}
	return false
}

func buildRunInspectInput(projectRoot string, iface runpkg.InterfaceResolution, apply, canonicalPresent bool) (pipeline.BundleInput, error) {
	assets, err := onboarding.DetectAssets(projectRoot)
	if err != nil {
		return pipeline.BundleInput{}, err
	}
	assetSet := make(map[string]struct{}, len(assets))
	for _, asset := range assets {
		assetSet[asset] = struct{}{}
	}

	guidanceItems := []resolveguidanceprecedence.GuidanceItem{
		{ID: "policy-default", Type: "policy"},
	}
	if hasDetectedAsset(assetSet, ".github/workflows") {
		guidanceItems = append(guidanceItems, resolveguidanceprecedence.GuidanceItem{ID: "workflow-detected", Type: "workflow"})
	}
	if hasDetectedAsset(assetSet, "AGENTS.md") {
		guidanceItems = append(guidanceItems, resolveguidanceprecedence.GuidanceItem{ID: "agents-guidance", Type: "skill"})
	}
	if hasDetectedAsset(assetSet, "docs") {
		guidanceItems = append(guidanceItems, resolveguidanceprecedence.GuidanceItem{ID: "repo-docs-map", Type: "repo_map"})
	}
	if hasDetectedAsset(assetSet, ".cursor", ".cursor/rules", ".vscode") {
		guidanceItems = append(guidanceItems, resolveguidanceprecedence.GuidanceItem{ID: "tool-hint-detected", Type: "tool_hint"})
	}

	failureClass := "projection_failure"
	triggers := []string{}
	if hasDetectedAsset(assetSet, "AGENTS.md") && hasDetectedAsset(assetSet, ".cursor", ".cursor/rules") {
		failureClass = "legacy_conflict_failure"
		triggers = append(triggers, "instruction_conflict", "policy_ambiguity")
	}
	if apply {
		failureClass = "approval_failure"
		triggers = append(triggers, "missing_approval")
	}

	extensionItems := []resolveextensionorder.Item{
		{ID: "policy-default", Kind: "policy"},
	}
	if hasDetectedAsset(assetSet, ".github/workflows") {
		extensionItems = append(extensionItems, resolveextensionorder.Item{ID: "workflow-detected", Kind: "workflow"})
	}
	if hasDetectedAsset(assetSet, "AGENTS.md") {
		extensionItems = append(extensionItems, resolveextensionorder.Item{ID: "agents-skill", Kind: "skill"})
	}
	if hasDetectedAsset(assetSet, ".cursor", ".cursor/rules", ".vscode") {
		extensionItems = append(extensionItems, resolveextensionorder.Item{ID: "tool-hint-detected", Kind: "tool_hint"})
	}
	extensionItems = append(extensionItems, resolveextensionorder.Item{ID: "runtime-hook", Kind: "hook"})
	extensionItems = append(extensionItems, resolveextensionorder.Item{ID: "projection-plugin", Kind: "projection_plugin"})

	opCommand := "doctor"
	if apply {
		opCommand = "repair"
	}
	portabilityInput, err := buildPortabilityInputFromAssets(projectRoot, iface, canonicalPresent, assets)
	if err != nil {
		return pipeline.BundleInput{}, err
	}

	input := pipeline.BundleInput{
		LayerItem:        classifylayer.Item{Kind: "request", SchemaID: "atrakta/schemas/core/request.schema.json"},
		GuidanceItems:    guidanceItems,
		ProjectionSource: checkprojectioneligibility.Source{Type: "policy", HasCanonicalAnchor: true},
		FailureClass:     failureClass,
		FailureContext: resolvefailuretier.Context{
			Scope:             "workspace",
			Triggers:          triggers,
			IsDiagnosticsOnly: false,
		},
		MutationTarget: checkmutationscope.Target{
			Path:            ".atrakta/generated/repo-map.generated.json",
			AssetType:       "repo_map",
			Operation:       "replace",
			ManagedOnlyPath: true,
		},
		LegacyAsset:      resolvelegacystatus.Asset{AssetID: "legacy-1", Ownership: "known", Freshness: "acceptable", CanonicalMapping: "exists", Integrity: "known"},
		OperationInput:   resolveoperationcapability.Input{CommandOrAlias: opCommand},
		ExtensionItems:   extensionItems,
		PortabilityInput: portabilityInput,
		AuditInput:       resolveauditrequirements.Input{Action: "inspect"},
		StrictInput:      strictstatemachine.StateInput{CurrentState: "normal", Scope: "workspace"},
	}
	if err := validation.ValidateBundleInput(input); err != nil {
		return pipeline.BundleInput{}, err
	}
	return input, nil
}

func buildPortabilityInput(projectRoot string, iface runpkg.InterfaceResolution, canonicalPresent bool) (resolvesurfaceportability.Input, error) {
	assets, err := onboarding.DetectAssets(projectRoot)
	if err != nil {
		return resolvesurfaceportability.Input{}, err
	}
	return buildPortabilityInputFromAssets(projectRoot, iface, canonicalPresent, assets)
}

func buildPortabilityInputFromAssets(projectRoot string, iface runpkg.InterfaceResolution, canonicalPresent bool, assets []string) (resolvesurfaceportability.Input, error) {
	caps, err := bindings.Load(iface.InterfaceID)
	if err != nil {
		return resolvesurfaceportability.Input{}, err
	}
	return resolvesurfaceportability.Input{
		InterfaceID:         iface.InterfaceID,
		RequestedTargets:    requestedPortabilityTargets(iface.InterfaceID, assets, caps),
		AvailableSources:    availablePortabilitySources(assets, canonicalPresent),
		BindingCapabilities: caps,
		DegradePolicy:       resolvesurfaceportability.DegradePolicyProposalOnly,
	}, nil
}

func requestedPortabilityTargets(interfaceID string, assets []string, caps resolvesurfaceportability.BindingCapabilities) []string {
	assetSet := make(map[string]struct{}, len(assets))
	for _, asset := range assets {
		assetSet[asset] = struct{}{}
	}
	targets := []string{}

	if hasDetectedAsset(assetSet, "AGENTS.md") {
		targets = append(targets, resolvesurfaceportability.TargetAgentsMD)
	}
	if hasDetectedAsset(assetSet, ".cursor", ".cursor/rules", ".vscode") {
		targets = append(targets, resolvesurfaceportability.TargetIDERules)
	}
	if hasDetectedAsset(assetSet, "docs") {
		targets = append(targets, resolvesurfaceportability.TargetRepoDocs)
	}
	if supportsProjectionTarget(caps, resolvesurfaceportability.TargetSkillBundle) && hasDetectedAsset(assetSet, "skills", ".codex/skills") {
		targets = append(targets, resolvesurfaceportability.TargetSkillBundle)
	}

	if len(targets) == 0 {
		if supportsProjectionTarget(caps, resolvesurfaceportability.TargetRepoDocs) && hasDetectedAsset(assetSet, "docs") {
			targets = append(targets, resolvesurfaceportability.TargetRepoDocs)
		}
		if len(targets) == 0 && supportsProjectionTarget(caps, resolvesurfaceportability.TargetAgentsMD) && hasDetectedAsset(assetSet, "AGENTS.md") {
			targets = append(targets, resolvesurfaceportability.TargetAgentsMD)
		}
	}
	if len(targets) == 0 {
		switch strings.TrimSpace(strings.ToLower(interfaceID)) {
		case "generic-cli", "claude-code":
			if supportsProjectionTarget(caps, resolvesurfaceportability.TargetAgentsMD) {
				targets = append(targets, resolvesurfaceportability.TargetAgentsMD)
			}
		case "cursor", "vscode", "copilot":
			if supportsProjectionTarget(caps, resolvesurfaceportability.TargetIDERules) {
				targets = append(targets, resolvesurfaceportability.TargetIDERules)
			}
		case "github-actions":
			if supportsProjectionTarget(caps, resolvesurfaceportability.TargetRepoDocs) {
				targets = append(targets, resolvesurfaceportability.TargetRepoDocs)
			}
		default:
			if supportsProjectionTarget(caps, resolvesurfaceportability.TargetRepoDocs) {
				targets = append(targets, resolvesurfaceportability.TargetRepoDocs)
			}
			if len(targets) == 0 && supportsProjectionTarget(caps, resolvesurfaceportability.TargetAgentsMD) {
				targets = append(targets, resolvesurfaceportability.TargetAgentsMD)
			}
		}
	}

	return uniqueStrings(targets)
}

func availablePortabilitySources(assets []string, canonicalPresent bool) []string {
	assetSet := make(map[string]struct{}, len(assets))
	for _, asset := range assets {
		assetSet[asset] = struct{}{}
	}
	sources := []string{}
	if canonicalPresent {
		sources = append(sources, resolvesurfaceportability.SourceCanonicalPolicy)
	}
	if hasDetectedAsset(assetSet, ".github/workflows") {
		sources = append(sources, resolvesurfaceportability.SourceWorkflowBinding)
	}
	if hasDetectedAsset(assetSet, "docs") {
		sources = append(sources, resolvesurfaceportability.SourceRepoDocs)
	}
	if hasDetectedAsset(assetSet, "AGENTS.md") {
		sources = append(sources, resolvesurfaceportability.SourceAgentsMD)
	}
	if hasDetectedAsset(assetSet, ".cursor", ".cursor/rules", ".vscode") {
		sources = append(sources, resolvesurfaceportability.SourceIDERules)
	}
	if hasDetectedAsset(assetSet, "skills", ".codex/skills") {
		sources = append(sources, resolvesurfaceportability.SourceSkillAsset)
	}
	return uniqueStrings(sources)
}

func supportsProjectionTarget(caps resolvesurfaceportability.BindingCapabilities, target string) bool {
	for _, candidate := range caps.ProjectionTargets {
		if strings.EqualFold(candidate, target) {
			return true
		}
	}
	return false
}

func extractPortabilityDecision(out pipeline.BundleOutput) (resolvesurfaceportability.PortabilityDecision, string, error) {
	for _, step := range out.Steps {
		if step.Name != "resolve_surface_portability" {
			continue
		}
		decision, ok := step.Output.Decision.(resolvesurfaceportability.PortabilityDecision)
		if !ok {
			return resolvesurfaceportability.PortabilityDecision{}, "", fmt.Errorf("unexpected portability decision type")
		}
		return decision, step.Output.Reason, nil
	}
	return resolvesurfaceportability.PortabilityDecision{}, "", fmt.Errorf("resolve_surface_portability step missing")
}

func resolvedProjectionTargets(decision resolvesurfaceportability.PortabilityDecision) []string {
	targets := make([]string, 0, len(decision.ProjectionPlan))
	for _, item := range decision.ProjectionPlan {
		if item.EffectiveTarget != "" {
			targets = append(targets, item.EffectiveTarget)
			continue
		}
		if item.Status == resolvesurfaceportability.PortabilitySupported {
			targets = append(targets, item.RequestedTarget)
		}
	}
	return uniqueStrings(targets)
}

func unsupportedPortability(reason string) resolvesurfaceportability.PortabilityDecision {
	return resolvesurfaceportability.PortabilityDecision{
		SupportedTargets:   []string{},
		DegradedTargets:    []string{},
		UnsupportedTargets: []string{},
		IngestPlan:         []string{},
		ProjectionPlan:     []resolvesurfaceportability.ProjectionPlanItem{},
		PortabilityStatus:  resolvesurfaceportability.PortabilityUnsupported,
		DegradePolicy:      resolvesurfaceportability.DegradePolicyProposalOnly,
	}
}

type runApplyPlan struct {
	Target   checkmutationscope.Target
	Content  string
	Proposal mutation.Proposal
}

func buildRunApplyPlans(projectRoot string, iface runpkg.InterfaceResolution, canonicalSummary map[string]any) ([]runApplyPlan, error) {
	assets, err := onboarding.DetectAssets(projectRoot)
	if err != nil {
		return nil, err
	}
	specs := []struct {
		target  checkmutationscope.Target
		payload map[string]any
	}{
		{
			target: checkmutationscope.Target{
				Path:      ".atrakta/generated/repo-map.generated.json",
				AssetType: "repo_map",
				Operation: "replace",
			},
			payload: map[string]any{
				"generator":         "atrakta.run",
				"interface_id":      iface.InterfaceID,
				"interface_source":  iface.Source,
				"detected_assets":   assets,
				"canonical_summary": canonicalSummary,
			},
		},
		{
			target: checkmutationscope.Target{
				Path:      ".atrakta/generated/capabilities.generated.json",
				AssetType: "capability",
				Operation: "replace",
			},
			payload: map[string]any{
				"generator":             "atrakta.run",
				"interface_id":          iface.InterfaceID,
				"inferred_capabilities": onboarding.InferCapabilities(assets),
			},
		},
		{
			target: checkmutationscope.Target{
				Path:      ".atrakta/generated/guidance.generated.json",
				AssetType: "guidance",
				Operation: "replace",
			},
			payload: map[string]any{
				"generator":                  "atrakta.run",
				"inferred_guidance_strength": onboarding.InferGuidanceStrength(assets),
				"inferred_managed_scope":     onboarding.InferManagedScope(assets),
			},
		},
	}

	plans := make([]runApplyPlan, 0, len(specs))
	for _, spec := range specs {
		b, err := json.MarshalIndent(spec.payload, "", "  ")
		if err != nil {
			return nil, err
		}
		content := string(append(b, '\n'))
		plans = append(plans, runApplyPlan{
			Target:   spec.target,
			Content:  content,
			Proposal: mutation.Propose(spec.target, content),
		})
	}
	return plans, nil
}

func loadCanonicalSummary(projectRoot string) (map[string]any, error) {
	policyPath := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry", "index.json")
	b, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("load canonical policy index: %w", err)
	}

	var policyIndex map[string]any
	if err := json.Unmarshal(b, &policyIndex); err != nil {
		return nil, fmt.Errorf("parse canonical policy index: %w", err)
	}

	entriesCount := 0
	if entries, ok := policyIndex["entries"].([]any); ok {
		entriesCount = len(entries)
	}

	summary := map[string]any{
		"policy_index_path":  filepath.ToSlash(policyPath),
		"policy_entry_count": entriesCount,
	}

	statePath := filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")
	if stateBytes, err := os.ReadFile(statePath); err == nil {
		var state map[string]any
		if err := json.Unmarshal(stateBytes, &state); err != nil {
			return nil, fmt.Errorf("parse onboarding state: %w", err)
		}
		if status, ok := state["status"].(string); ok {
			summary["onboarding_status"] = status
		}
	}

	return summary, nil
}

func appendRunAuditEvent(commandName, projectRoot string, payload map[string]any) error {
	auditRoot := filepath.Join(projectRoot, ".atrakta", "audit")
	if _, err := audit.AppendAndVerify(auditRoot, audit.LevelA2, "run_execute", payload); err != nil {
		return err
	}
	eventType := runEventTypeForPayload(commandName, payload)
	if eventType == "" {
		return nil
	}
	return appendRunLifecycleEvent(auditRoot, eventType, payload)
}

func appendSessionLifecycleEvent(commandName, projectRoot, eventType string, payload map[string]any) error {
	if !isSessionCommand(commandName) {
		return nil
	}
	auditRoot := filepath.Join(projectRoot, ".atrakta", "audit")
	return appendRunLifecycleEvent(auditRoot, eventType, payload)
}

func appendRunLifecycleEvent(auditRoot, eventType string, payload map[string]any) error {
	opts := audit.RunEventOptions{
		Actor:     "kernel",
		Interface: mapString(payload, "interface_id"),
	}
	_, err := audit.AppendRunEventAndVerify(auditRoot, audit.LevelA2, eventType, payload, opts)
	return err
}

func runEventTypeForPayload(commandName string, payload map[string]any) string {
	if !isSessionCommand(commandName) && !mapBool(payload, "apply_requested") && !mapBool(payload, "apply_performed") {
		return ""
	}
	if hasBoolKey(payload, "contract_load_ok") && !mapBool(payload, "contract_load_ok") {
		return "error.raised"
	}
	if mapString(payload, "status") == "needs_approval" {
		return "gate.result"
	}
	if mapBool(payload, "fast_path") {
		return ""
	}
	if mapBool(payload, "apply_begin") {
		return runEventApplyBegin
	}
	if mapBool(payload, "apply_performed") {
		return runEventApplyPerformed
	}
	if commandName == "resume" {
		return runEventResumeEnd
	}
	if isSessionCommand(commandName) {
		return runEventStartEnd
	}
	return ""
}

func mapBool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func hasBoolKey(m map[string]any, key string) bool {
	_, ok := m[key].(bool)
	return ok
}

func mapString(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func writeRunState(projectRoot string, payload map[string]any) error {
	path := filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func saveStartFastSnapshot(commandName, projectRoot string, iface runpkg.InterfaceResolution, applyRequested bool) error {
	if !isSessionCommand(commandName) {
		return nil
	}
	key, err := startfast.ComputeKey(projectRoot, iface.InterfaceID, applyRequested)
	if err != nil {
		return err
	}
	return startfast.SaveSnapshot(projectRoot, startfast.Snapshot{
		Key:                 key.Key,
		ContractHash:        key.ContractHash,
		CanonicalPolicyHash: key.CanonicalPolicyHash,
		WorkspaceStamp:      key.WorkspaceStamp,
		InterfaceID:         iface.InterfaceID,
		ApplyRequested:      applyRequested,
	})
}

func saveStartAutoState(commandName, projectRoot string, iface runpkg.InterfaceResolution) error {
	if !isSessionCommand(commandName) {
		return nil
	}
	return runpkg.SaveAutoState(projectRoot, runpkg.AutoState{
		InterfaceID:     iface.InterfaceID,
		InterfaceSource: iface.Source,
	})
}

func saveStartHandoff(commandName, projectRoot string, resp runResponse, fastPath bool, plannedTargets, appliedTargets []string) error {
	if !isSessionCommand(commandName) {
		return nil
	}

	handoff := runpkg.HandoffBundle{
		Command:           commandName,
		CanonicalState:    resp.CanonicalState,
		Status:            resp.Status,
		Message:           resp.Message,
		InterfaceID:       resp.Interface.InterfaceID,
		InterfaceSource:   resp.Interface.Source,
		ApplyRequested:    resp.ApplyRequested,
		Approved:          resp.Approved,
		FastPath:          fastPath,
		PortabilityStatus: resp.PortabilityStatus,
		PortabilityReason: resp.PortabilityReason,
		NextAllowedAction: resp.NextAllowedAction,
		NextAction: runpkg.HandoffNextAction{
			Command: resp.NextAllowedAction,
			Hint:    buildHandoffNextHint(resp),
		},
		FeatureSpec: runpkg.HandoffFeatureSpec{
			Summary:           buildHandoffFeatureSummary(resp, plannedTargets, appliedTargets),
			ResolvedTargets:   resp.ResolvedTargets,
			DegradedSurfaces:  resp.DegradedSurfaces,
			MissingTargets:    resp.MissingTargets,
			PortabilityStatus: resp.PortabilityStatus,
			PortabilityReason: resp.PortabilityReason,
		},
		Acceptance:     buildHandoffAcceptance(resp, plannedTargets, appliedTargets),
		PlannedTargets: plannedTargets,
		AppliedTargets: appliedTargets,
		Checkpoint: runpkg.HandoffCheckpoint{
			AutoStatePath:   existingFileOrEmpty(runpkg.AutoStatePath(projectRoot)),
			StartFastPath:   existingFileOrEmpty(startfast.SnapshotPath(projectRoot)),
			RunStatePath:    existingFileOrEmpty(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")),
			StatePath:       existingFileOrEmpty(runpkg.SessionStatePath(projectRoot)),
			ProgressPath:    existingFileOrEmpty(runpkg.SessionProgressPath(projectRoot)),
			TaskGraphPath:   existingFileOrEmpty(runpkg.SessionTaskGraphPath(projectRoot)),
			AuditHeadPath:   existingFileOrEmpty(filepath.Join(projectRoot, ".atrakta", "audit", "checkpoints", "head.json")),
			OnboardingState: existingFileOrEmpty(filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")),
		},
	}
	return runpkg.SaveHandoff(projectRoot, handoff)
}

func saveSessionRuntimeArtifacts(commandName, projectRoot string, resp runResponse, plannedTargets, appliedTargets []string, fastPath bool) error {
	if !isSessionCommand(commandName) || fastPath {
		return nil
	}

	state := runpkg.SessionState{
		Command:           commandName,
		CanonicalState:    resp.CanonicalState,
		Status:            resp.Status,
		InterfaceID:       resp.Interface.InterfaceID,
		InterfaceSource:   resp.Interface.Source,
		PortabilityStatus: resp.PortabilityStatus,
		PortabilityReason: resp.PortabilityReason,
		ApplyRequested:    resp.ApplyRequested,
		Approved:          resp.Approved,
		PlannedCount:      len(plannedTargets),
		AppliedCount:      len(appliedTargets),
	}
	if err := runpkg.SaveSessionState(projectRoot, state); err != nil {
		return err
	}

	progress := runpkg.SessionProgress{
		Command:           commandName,
		Status:            statusForProgress(resp.Status, plannedTargets, appliedTargets),
		NextAllowedAction: resp.NextAllowedAction,
		PlannedCount:      len(plannedTargets),
		AppliedCount:      len(appliedTargets),
	}
	if err := runpkg.SaveSessionProgress(projectRoot, progress); err != nil {
		return err
	}

	graph := runpkg.SessionTaskGraph{
		Command: commandName,
		Nodes:   buildSessionTaskNodes(plannedTargets, appliedTargets),
	}
	return runpkg.SaveSessionTaskGraph(projectRoot, graph)
}

func statusForProgress(respStatus string, plannedTargets, appliedTargets []string) string {
	if respStatus != "ok" {
		return respStatus
	}
	if len(appliedTargets) > 0 {
		return "applied"
	}
	if len(plannedTargets) > 0 {
		return "planned"
	}
	return "ok"
}

func buildSessionTaskNodes(plannedTargets, appliedTargets []string) []runpkg.SessionTaskNode {
	nodes := make([]runpkg.SessionTaskNode, 0, len(plannedTargets))
	appliedSet := make(map[string]struct{}, len(appliedTargets))
	for _, target := range appliedTargets {
		appliedSet[target] = struct{}{}
	}
	for i, target := range plannedTargets {
		status := "planned"
		if _, ok := appliedSet[target]; ok {
			status = "applied"
		}
		nodes = append(nodes, runpkg.SessionTaskNode{
			ID:     fmt.Sprintf("task-%d", i+1),
			Kind:   "managed_mutation",
			Target: target,
			Status: status,
		})
	}
	return nodes
}

func buildHandoffFeatureSummary(resp runResponse, plannedTargets, appliedTargets []string) string {
	switch {
	case len(appliedTargets) > 0:
		return fmt.Sprintf("managed apply completed for %d targets", len(appliedTargets))
	case len(plannedTargets) > 0:
		return fmt.Sprintf("managed plan prepared for %d targets", len(plannedTargets))
	case len(resp.ResolvedTargets) > 0:
		return fmt.Sprintf("session prepared for %d projection targets", len(resp.ResolvedTargets))
	default:
		return strings.TrimSpace(resp.Message)
	}
}

func buildHandoffNextHint(resp runResponse) string {
	if strings.TrimSpace(resp.Message) == "" {
		return ""
	}
	return resp.Message
}

func buildHandoffAcceptance(resp runResponse, plannedTargets, appliedTargets []string) []string {
	hints := make([]string, 0, 4)
	switch resp.PortabilityStatus {
	case resolvesurfaceportability.PortabilitySupported:
		hints = append(hints, "requested surface portability is supported")
	case resolvesurfaceportability.PortabilityDegraded:
		hints = append(hints, "requested surface portability is degraded and should remain proposal-only")
	case resolvesurfaceportability.PortabilityUnsupported:
		hints = append(hints, "requested surface portability is unsupported until the interface or contract changes")
	}
	if len(plannedTargets) > 0 {
		hints = append(hints, fmt.Sprintf("%d managed targets are planned", len(plannedTargets)))
	}
	if len(appliedTargets) > 0 {
		hints = append(hints, fmt.Sprintf("%d managed targets were applied", len(appliedTargets)))
	}
	if resp.NextAllowedAction != "" {
		hints = append(hints, fmt.Sprintf("next allowed action remains %s", resp.NextAllowedAction))
	}
	return hints
}

func planTargetPaths(plans []runApplyPlan) []string {
	paths := make([]string, 0, len(plans))
	for _, plan := range plans {
		paths = append(paths, plan.Target.Path)
	}
	return paths
}

func existingFileOrEmpty(path string) string {
	if path == "" {
		return ""
	}
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		return path
	}
	return ""
}

func isSessionCommand(commandName string) bool {
	return commandName == "start" || commandName == "resume"
}

func preflightVerifyAudit(projectRoot string) (string, error) {
	auditRoot := filepath.Join(projectRoot, ".atrakta", "audit")
	if _, err := os.Stat(auditRoot); err != nil {
		if os.IsNotExist(err) {
			return "missing_bootstrap", nil
		}
		return "", err
	}
	if err := audit.VerifyIntegrity(auditRoot, audit.LevelA2); err != nil {
		return "", fmt.Errorf("preflight audit verify: %w", err)
	}
	if err := audit.VerifyRunEventsIntegrity(auditRoot, audit.LevelA2); err != nil {
		return "", fmt.Errorf("preflight run-events verify: %w", err)
	}
	return "verified", nil
}

type approvalOutcome string

const (
	approvalOutcomeApproved         approvalOutcome = "approved"
	approvalOutcomeRequiresExplicit approvalOutcome = "requires_explicit"
	approvalOutcomeRejected         approvalOutcome = "rejected"
)

func evaluateApproval(explicitApprove, nonInteractive bool, prompt string) (bool, approvalOutcome, error) {
	if explicitApprove {
		return true, approvalOutcomeApproved, nil
	}
	if nonInteractive || !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return false, approvalOutcomeRequiresExplicit, nil
	}
	approved, err := promptApproval(prompt)
	if err != nil {
		return false, approvalOutcomeRejected, err
	}
	if !approved {
		return false, approvalOutcomeRejected, nil
	}
	return true, approvalOutcomeApproved, nil
}

func emitRunResponse(commandName string, resp runResponse, jsonOut bool, artifactDir string) error {
	switch commandName {
	case "start", "resume":
		if err := validation.ValidateStartOutput(resp); err != nil {
			return err
		}
	default:
		if err := validation.ValidateRunOutput(resp); err != nil {
			return err
		}
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
	} else {
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("path: %s\n", resp.Path)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("canonical_state: %s\n", resp.CanonicalState)
		if resp.Interface.InterfaceID != "" {
			fmt.Printf("interface: %s (%s)\n", resp.Interface.InterfaceID, resp.Interface.Source)
		}
		fmt.Printf("message: %s\n", resp.Message)
		if resp.NextAllowedAction != "" {
			fmt.Printf("next: %s\n", resp.NextAllowedAction)
		}
		if resp.Onboarding != nil {
			fmt.Printf("inferred_mode: %s\n", resp.Onboarding.InferredMode)
			fmt.Printf("detected_assets: %d\n", len(resp.Onboarding.DetectedAssets))
			fmt.Printf("conflicts: %d\n", len(resp.Onboarding.Conflicts))
		}
		if resp.AcceptResult != nil {
			fmt.Printf("written: %d\n", len(resp.AcceptResult.Written))
		}
		if resp.Error != nil {
			fmt.Printf("error_code: %s\n", resp.Error.Code)
			fmt.Printf("error_message: %s\n", resp.Error.Message)
			if len(resp.Error.RecoverySteps) > 0 {
				fmt.Printf("recovery_steps: %s\n", strings.Join(resp.Error.RecoverySteps, " | "))
			}
		}
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, commandName+".result.json", resp); err != nil {
			return err
		}
	}
	return nil
}

func structuredErrorFromNeedsInput(message string) *atraktaerrors.StructuredError {
	err := atraktaerrors.NewStructured(
		"ERR_BLOCKED",
		message,
		"Inspect the reported canonical state and repair the incomplete workspace state before retrying.",
	)
	return &err
}
