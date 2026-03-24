package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/bindings"
	"github.com/mash4649/atrakta/v0/internal/extensions"
	"github.com/mash4649/atrakta/v0/internal/fixtures"
	"github.com/mash4649/atrakta/v0/internal/mutation"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/persist"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
	"github.com/mash4649/atrakta/v0/internal/validation"
	resolveauditrequirements "github.com/mash4649/atrakta/v0/resolvers/audit/resolve-audit-requirements"
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

const (
	exitOK            = 0
	exitRuntimeError  = 1
	exitNeedsInput    = 2
	exitNeedsApproval = 3
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		code, err := runCommand(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			if code == exitOK {
				code = exitRuntimeError
			}
			os.Exit(code)
		}
		if code != exitOK {
			os.Exit(code)
		}
	case "inspect", "preview", "simulate":
		if err := runMode(cmd, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "run-fixtures":
		if err := runFixtures(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "accept":
		if err := runAccept(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "mutate":
		if err := runMutate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "audit":
		if err := runAudit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "doctor", "parity", "integration":
		if err := runAlias(cmd, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "extensions":
		if err := runExtensions(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "onboard":
		if err := runOnboard(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "export-snapshots":
		if err := runExportSnapshots(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "verify-coverage":
		if err := runVerifyCoverage(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func runMode(mode string, args []string) error {
	fs := flag.NewFlagSet(mode, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var inputPath string
	var artifactDir string
	var onboardRoot string
	fs.StringVar(&inputPath, "input", "", "bundle input JSON path")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.StringVar(&onboardRoot, "onboard-root", "", "project root for onboarding-derived failure routing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var input pipeline.BundleInput
	if inputPath != "" {
		b, err := os.ReadFile(inputPath)
		if err != nil {
			return err
		}
		input, err = validation.DecodeAndValidateBundleInput(b)
		if err != nil {
			return err
		}
	} else {
		var err error
		input, err = buildDefaultInput(mode)
		if err != nil {
			return err
		}
	}
	if onboardRoot != "" {
		onboardingBundle, err := onboarding.BuildOnboardingProposal(onboardRoot)
		if err != nil {
			return err
		}
		if err := validation.ValidateOnboardingProposal(onboardingBundle); err != nil {
			return err
		}
		input = applyOnboardingFailure(input, onboardingBundle)
		if err := validation.ValidateBundleInput(input); err != nil {
			return err
		}
	}

	out, err := executeBundle(mode, input)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, mode+".bundle.json", out); err != nil {
			return err
		}
	}
	return nil
}

func runFixtures(args []string) error {
	fs := flag.NewFlagSet("run-fixtures", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var artifactDir string
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	report, err := runFixturesReport()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "fixtures.report.json", report); err != nil {
			return err
		}
	}
	if report.Failed > 0 {
		return fmt.Errorf("fixture failures: %d", report.Failed)
	}
	return nil
}

func runExportSnapshots(args []string) error {
	fs := flag.NewFlagSet("export-snapshots", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var dir string
	fs.StringVar(&dir, "dir", "fixtures/snapshots", "snapshot output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	onboardingBundle, err := onboarding.BuildOnboardingProposal("")
	if err != nil {
		return err
	}
	if err := validation.ValidateOnboardingProposal(onboardingBundle); err != nil {
		return err
	}
	if err := writeArtifact(dir, "onboarding.proposal.json", onboardingBundle); err != nil {
		return err
	}

	inspectOnboardInput, err := buildDefaultInput("inspect")
	if err != nil {
		return err
	}
	inspectOnboardInput = applyOnboardingFailure(inspectOnboardInput, onboardingBundle)
	inspectOnboardOut, err := executeBundle("inspect", inspectOnboardInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "inspect.onboard.bundle.json", inspectOnboardOut); err != nil {
		return err
	}

	previewOnboardInput, err := buildDefaultInput("preview")
	if err != nil {
		return err
	}
	previewOnboardInput = applyOnboardingFailure(previewOnboardInput, onboardingBundle)
	previewOnboardOut, err := executeBundle("preview", previewOnboardInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "preview.onboard.bundle.json", previewOnboardOut); err != nil {
		return err
	}

	simulateOnboardInput, err := buildDefaultInput("simulate")
	if err != nil {
		return err
	}
	simulateOnboardInput = applyOnboardingFailure(simulateOnboardInput, onboardingBundle)
	simulateOnboardOut, err := executeBundle("simulate", simulateOnboardInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "simulate.onboard.bundle.json", simulateOnboardOut); err != nil {
		return err
	}

	inspectInput, err := buildDefaultInput("inspect")
	if err != nil {
		return err
	}
	inspectOut, err := executeBundle("inspect", inspectInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "inspect.bundle.json", inspectOut); err != nil {
		return err
	}

	previewInput, err := buildDefaultInput("preview")
	if err != nil {
		return err
	}
	previewOut, err := executeBundle("preview", previewInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "preview.bundle.json", previewOut); err != nil {
		return err
	}

	simulateInput, err := buildDefaultInput("simulate")
	if err != nil {
		return err
	}
	simulateOut, err := executeBundle("simulate", simulateInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "simulate.bundle.json", simulateOut); err != nil {
		return err
	}

	report, err := runFixturesReport()
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "fixtures.report.json", report); err != nil {
		return err
	}

	return nil
}

func runOnboard(args []string) error {
	fs := flag.NewFlagSet("onboard", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root to inspect")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	bundle, err := onboarding.BuildOnboardingProposal(projectRoot)
	if err != nil {
		return err
	}
	if err := validation.ValidateOnboardingProposal(bundle); err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(bundle); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "onboarding.proposal.json", bundle); err != nil {
			return err
		}
	}
	return nil
}

func runAccept(args []string) error {
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var proposalPath string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&proposalPath, "proposal", "", "onboarding proposal JSON path")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var bundle onboarding.ProposalBundle
	if proposalPath != "" {
		b, err := os.ReadFile(proposalPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &bundle); err != nil {
			return err
		}
	} else {
		var err error
		bundle, err = onboarding.BuildOnboardingProposal(projectRoot)
		if err != nil {
			return err
		}
	}
	if err := validation.ValidateOnboardingProposal(bundle); err != nil {
		return err
	}

	result, err := persist.AcceptOnboarding(projectRoot, bundle)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "accept.result.json", result); err != nil {
			return err
		}
	}
	return nil
}

type runResponse struct {
	Status            string                      `json:"status"`
	Path              string                      `json:"path"`
	ProjectRoot       string                      `json:"project_root"`
	CanonicalState    string                      `json:"canonical_state"`
	CanonicalSummary  map[string]any              `json:"canonical_summary,omitempty"`
	Interface         runpkg.InterfaceResolution  `json:"interface"`
	ApplyRequested    bool                        `json:"apply_requested"`
	Approved          bool                        `json:"approved"`
	Message           string                      `json:"message"`
	Portability       any                         `json:"portability"`
	ResolvedTargets   []string                    `json:"resolved_projection_targets"`
	DegradedSurfaces  []string                    `json:"degraded_surfaces"`
	MissingTargets    []string                    `json:"missing_projection_targets"`
	PortabilityStatus string                      `json:"portability_status"`
	PortabilityReason string                      `json:"portability_reason"`
	NextAllowedAction string                      `json:"next_allowed_action,omitempty"`
	RequiredInputs    []string                    `json:"required_inputs,omitempty"`
	ApprovalScope     string                      `json:"approval_scope,omitempty"`
	Onboarding        *onboarding.ProposalBundle  `json:"onboarding_proposal,omitempty"`
	AcceptResult      *persist.AcceptResult       `json:"accept_result,omitempty"`
	InspectBundle     *pipeline.BundleOutput      `json:"inspect_bundle,omitempty"`
	PlannedMutations  []mutation.Proposal         `json:"planned_mutations,omitempty"`
	AppliedMutations  []mutation.DecisionEnvelope `json:"applied_mutations,omitempty"`
}

func runCommand(args []string) (int, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
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

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}
	canonicalState, err := runpkg.DetectCanonicalState(root)
	if err != nil {
		return exitRuntimeError, err
	}
	if canonicalState == runpkg.StatePartialState || canonicalState == runpkg.StateCorruptState {
		return exitRuntimeError, fmt.Errorf("run blocked: canonical state is %s", canonicalState)
	}

	iface, err := runpkg.ResolveInterface(root, interfaceID, strings.TrimSpace(os.Getenv("ATRAKTA_TRIGGER_INTERFACE")))
	if err != nil {
		if errors.Is(err, runpkg.ErrInterfaceUnresolved) {
			if canonicalState == runpkg.StateNone {
				iface = runpkg.InterfaceResolution{InterfaceID: "generic-cli", Source: "default"}
			} else {
				resp := runResponse{
					Status:            "needs_input",
					Path:              "normal",
					ProjectRoot:       root,
					CanonicalState:    canonicalState,
					Interface:         runpkg.InterfaceResolution{InterfaceID: "unresolved", Source: "detect"},
					ApplyRequested:    apply,
					Message:           "interface could not be resolved deterministically",
					Portability:       unsupportedPortability("interface unresolved"),
					ResolvedTargets:   []string{},
					DegradedSurfaces:  []string{},
					MissingTargets:    []string{},
					PortabilityStatus: resolvesurfaceportability.PortabilityUnsupported,
					PortabilityReason: "interface unresolved",
					NextAllowedAction: "re-run with --interface <id> or set ATRAKTA_TRIGGER_INTERFACE",
					RequiredInputs:    []string{"interface"},
				}
				if emitErr := emitRunResponse(resp, jsonOut, artifactDir); emitErr != nil {
					return exitRuntimeError, emitErr
				}
				return exitNeedsInput, nil
			}
		} else {
			return exitRuntimeError, err
		}
	}

	switch canonicalState {
	case runpkg.StateNone:
		return runOnboardingPath(root, iface, nonInteractive, approve, jsonOut, artifactDir)
	case runpkg.StateCanonicalPresent, runpkg.StateOnboardingComplete:
		summary, err := loadCanonicalSummary(root)
		if err != nil {
			return exitRuntimeError, err
		}
		return runNormalPath(root, canonicalState, iface, summary, apply, approve, nonInteractive, jsonOut, artifactDir)
	default:
		return exitRuntimeError, fmt.Errorf("unsupported canonical state %q", canonicalState)
	}
}

func runOnboardingPath(root string, iface runpkg.InterfaceResolution, nonInteractive, approve, jsonOut bool, artifactDir string) (int, error) {
	bundle, err := onboarding.BuildOnboardingProposal(root)
	if err != nil {
		return exitRuntimeError, err
	}
	if err := validation.ValidateOnboardingProposal(bundle); err != nil {
		return exitRuntimeError, err
	}
	portabilityInput, err := buildPortabilityInput(root, iface, false)
	if err != nil {
		return exitRuntimeError, err
	}
	portabilityOut := resolvesurfaceportability.ResolveSurfacePortability(portabilityInput)
	portabilityDecision := portabilityOut.Decision.(resolvesurfaceportability.PortabilityDecision)

	resp := runResponse{
		Status:            "ok",
		Path:              "onboarding",
		ProjectRoot:       root,
		CanonicalState:    runpkg.StateNone,
		Interface:         iface,
		Message:           "onboarding proposal generated",
		Portability:       portabilityDecision,
		ResolvedTargets:   resolvedProjectionTargets(portabilityDecision),
		DegradedSurfaces:  portabilityDecision.DegradedTargets,
		MissingTargets:    portabilityDecision.UnsupportedTargets,
		PortabilityStatus: portabilityDecision.PortabilityStatus,
		PortabilityReason: portabilityOut.Reason,
		NextAllowedAction: portabilityOut.NextAllowedAction,
		Onboarding:        &bundle,
	}

	resp.Approved = approve
	if !approve && (nonInteractive || !isTerminal(os.Stdin) || !isTerminal(os.Stdout)) {
		resp.Status = "needs_approval"
		resp.Message = "explicit approval is required before onboarding persistence"
		resp.NextAllowedAction = "re-run with --approve, or run interactively and approve, or run `atrakta accept --project-root <dir>`"
		resp.ApprovalScope = "onboarding_accept"
		if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitNeedsApproval, nil
	}

	approved := approve
	if !approved {
		approved, err = promptApproval("Accept onboarding proposal and persist canonical/state/audit?")
		if err != nil {
			return exitRuntimeError, err
		}
	}
	if !approved {
		resp.Status = "needs_approval"
		resp.Message = "onboarding proposal was not approved"
		resp.NextAllowedAction = "re-run `atrakta run --approve` or approve persistence when prompted"
		resp.ApprovalScope = "onboarding_accept"
		if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitNeedsApproval, nil
	}
	resp.Approved = true

	result, err := persist.AcceptOnboarding(root, bundle)
	if err != nil {
		return exitRuntimeError, err
	}
	resp.Message = "onboarding accepted and persisted"
	resp.AcceptResult = &result
	if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func runNormalPath(root, canonicalState string, iface runpkg.InterfaceResolution, canonicalSummary map[string]any, apply, approve, nonInteractive, jsonOut bool, artifactDir string) (int, error) {
	input, err := buildRunInspectInput(root, iface, apply, true)
	if err != nil {
		return exitRuntimeError, err
	}
	out, err := executeBundle("inspect", input)
	if err != nil {
		return exitRuntimeError, err
	}
	portabilityDecision, portabilityReason, err := extractPortabilityDecision(out)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := runResponse{
		Status:            "ok",
		Path:              "normal",
		ProjectRoot:       root,
		CanonicalState:    canonicalState,
		CanonicalSummary:  canonicalSummary,
		Interface:         iface,
		ApplyRequested:    apply,
		Approved:          approve,
		Message:           "inspect pipeline executed",
		Portability:       portabilityDecision,
		ResolvedTargets:   resolvedProjectionTargets(portabilityDecision),
		DegradedSurfaces:  portabilityDecision.DegradedTargets,
		MissingTargets:    portabilityDecision.UnsupportedTargets,
		PortabilityStatus: portabilityDecision.PortabilityStatus,
		PortabilityReason: portabilityReason,
		InspectBundle:     &out,
	}
	if portabilityDecision.PortabilityStatus != resolvesurfaceportability.PortabilitySupported {
		resp.Message = "inspect pipeline executed with proposal-only portability degradation"
		resp.NextAllowedAction = "propose"
	} else {
		resp.NextAllowedAction = out.FinalAllowedAction
	}
	plans, err := buildRunApplyPlans(root, iface, canonicalSummary)
	if err != nil {
		return exitRuntimeError, err
	}
	resp.PlannedMutations = make([]mutation.Proposal, 0, len(plans))
	for _, plan := range plans {
		resp.PlannedMutations = append(resp.PlannedMutations, plan.Proposal)
	}

	if apply && portabilityDecision.PortabilityStatus != resolvesurfaceportability.PortabilitySupported {
		resp.Message = "apply disabled because requested surface portability is proposal-only"
		resp.NextAllowedAction = "propose"
		if err := appendRunAuditEvent(root, map[string]any{
			"status":               resp.Status,
			"path":                 resp.Path,
			"interface_id":         iface.InterfaceID,
			"interface_source":     iface.Source,
			"apply_requested":      apply,
			"approved":             false,
			"portability_status":   portabilityDecision.PortabilityStatus,
			"final_allowed_action": out.FinalAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if apply {
		approved := approve
		if !approved {
			if nonInteractive || !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
				resp.Status = "needs_approval"
				resp.Message = "apply route requires explicit managed mutation approval"
				resp.NextAllowedAction = "re-run with --apply --approve to execute managed apply"
				resp.ApprovalScope = "managed_apply"
				if err := appendRunAuditEvent(root, map[string]any{
					"status":               resp.Status,
					"path":                 resp.Path,
					"interface_id":         iface.InterfaceID,
					"interface_source":     iface.Source,
					"apply_requested":      apply,
					"approved":             false,
					"final_allowed_action": out.FinalAllowedAction,
				}); err != nil {
					return exitRuntimeError, err
				}
				if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
					return exitRuntimeError, err
				}
				return exitNeedsApproval, nil
			}
			approved, err = promptApproval("Apply managed mutation plan to generated managed targets?")
			if err != nil {
				return exitRuntimeError, err
			}
			if !approved {
				resp.Status = "needs_approval"
				resp.Message = "apply plan was not approved"
				resp.NextAllowedAction = "re-run with --apply --approve when ready"
				resp.ApprovalScope = "managed_apply"
				if err := appendRunAuditEvent(root, map[string]any{
					"status":               resp.Status,
					"path":                 resp.Path,
					"interface_id":         iface.InterfaceID,
					"interface_source":     iface.Source,
					"apply_requested":      apply,
					"approved":             false,
					"final_allowed_action": out.FinalAllowedAction,
				}); err != nil {
					return exitRuntimeError, err
				}
				if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
					return exitRuntimeError, err
				}
				return exitNeedsApproval, nil
			}
		}
		resp.Approved = true
		resp.AppliedMutations = make([]mutation.DecisionEnvelope, 0, len(plans))
		appliedTargets := make([]string, 0, len(plans))
		decisionIDs := make([]string, 0, len(plans))
		for _, plan := range plans {
			applied, err := mutation.Apply(root, plan.Target, plan.Content, true)
			if err != nil {
				return exitRuntimeError, err
			}
			resp.AppliedMutations = append(resp.AppliedMutations, applied)
			appliedTargets = append(appliedTargets, plan.Target.Path)
			decisionIDs = append(decisionIDs, applied.DecisionID)
		}
		resp.Message = fmt.Sprintf("inspect pipeline executed and managed apply completed (%d targets)", len(resp.AppliedMutations))
		if err := writeRunState(root, map[string]any{
			"status":               "applied",
			"interface_id":         iface.InterfaceID,
			"applied_target_paths": appliedTargets,
			"decision_ids":         decisionIDs,
			"applied_count":        len(resp.AppliedMutations),
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := appendRunAuditEvent(root, map[string]any{
			"status":               resp.Status,
			"path":                 resp.Path,
			"interface_id":         iface.InterfaceID,
			"interface_source":     iface.Source,
			"apply_requested":      apply,
			"approved":             true,
			"apply_performed":      true,
			"applied_target_paths": appliedTargets,
			"applied_count":        len(appliedTargets),
			"final_allowed_action": out.FinalAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if err := appendRunAuditEvent(root, map[string]any{
		"status":               resp.Status,
		"path":                 resp.Path,
		"interface_id":         iface.InterfaceID,
		"interface_source":     iface.Source,
		"apply_requested":      apply,
		"final_allowed_action": out.FinalAllowedAction,
	}); err != nil {
		return exitRuntimeError, err
	}
	if err := emitRunResponse(resp, jsonOut, artifactDir); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
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

func hasDetectedAsset(set map[string]struct{}, names ...string) bool {
	for _, name := range names {
		if _, ok := set[name]; ok {
			return true
		}
	}
	return false
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

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
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

func appendRunAuditEvent(projectRoot string, payload map[string]any) error {
	auditRoot := filepath.Join(projectRoot, ".atrakta", "audit")
	if _, err := audit.AppendEvent(auditRoot, audit.LevelA2, "run_execute", payload); err != nil {
		return err
	}
	return audit.VerifyIntegrity(auditRoot, audit.LevelA2)
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

func emitRunResponse(resp runResponse, jsonOut bool, artifactDir string) error {
	if err := validation.ValidateRunOutput(resp); err != nil {
		return err
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
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "run.result.json", resp); err != nil {
			return err
		}
	}
	return nil
}

func promptApproval(prompt string) (bool, error) {
	fmt.Fprintf(os.Stdout, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func isTerminal(file *os.File) bool {
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func flagWasProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}

func isTruthyEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func runMutate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: atrakta mutate <inspect|propose|apply> [flags]")
	}
	phase := args[0]
	phaseArgs := args[1:]

	fs := flag.NewFlagSet("mutate "+phase, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var targetPath string
	var declaredScope string
	var assetType string
	var operation string
	var content string
	var contentFile string
	var projectRoot string
	var allow bool
	var artifactDir string
	fs.StringVar(&targetPath, "target", "", "target path")
	fs.StringVar(&declaredScope, "declared-scope", "", "declared scope override")
	fs.StringVar(&assetType, "asset-type", "", "asset type")
	fs.StringVar(&operation, "operation", "", "operation type")
	fs.StringVar(&content, "content", "", "inline content")
	fs.StringVar(&contentFile, "content-file", "", "content file path")
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&allow, "allow", false, "explicitly allow apply")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(phaseArgs); err != nil {
		return err
	}
	if targetPath == "" {
		return fmt.Errorf("--target is required")
	}

	target := checkmutationscope.Target{
		Path:          targetPath,
		DeclaredScope: declaredScope,
		AssetType:     assetType,
		Operation:     operation,
	}

	resolveContent := func() (string, error) {
		if contentFile != "" {
			b, err := os.ReadFile(contentFile)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
		return content, nil
	}

	var output any
	var err error

	switch phase {
	case "inspect":
		output = mutation.Inspect(target)
	case "propose":
		var body string
		body, err = resolveContent()
		if err != nil {
			return err
		}
		output = mutation.Propose(target, body)
	case "apply":
		var body string
		body, err = resolveContent()
		if err != nil {
			return err
		}
		output, err = mutation.Apply(projectRoot, target, body, allow)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported mutate phase %q", phase)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "mutate."+phase+".json", output); err != nil {
			return err
		}
	}
	return nil
}

func runAudit(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: atrakta audit <append|verify> [flags]")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "append":
		fs := flag.NewFlagSet("audit append", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var projectRoot string
		var level string
		var action string
		var payloadPath string
		var artifactDir string
		fs.StringVar(&projectRoot, "project-root", "", "project root")
		fs.StringVar(&level, "level", audit.LevelA2, "audit integrity level")
		fs.StringVar(&action, "action", "", "audit action")
		fs.StringVar(&payloadPath, "payload-file", "", "payload json file")
		fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		if action == "" {
			return fmt.Errorf("--action is required")
		}

		root, err := onboarding.DetectProjectRoot(projectRoot)
		if err != nil {
			return err
		}
		payload := map[string]any{}
		if payloadPath != "" {
			b, err := os.ReadFile(payloadPath)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &payload); err != nil {
				return err
			}
		}

		event, err := audit.AppendEvent(filepath.Join(root, ".atrakta", "audit"), level, action, payload)
		if err != nil {
			return err
		}
		if err := audit.VerifyIntegrity(filepath.Join(root, ".atrakta", "audit"), level); err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(event); err != nil {
			return err
		}
		if artifactDir != "" {
			if err := writeArtifact(artifactDir, "audit.append.json", event); err != nil {
				return err
			}
		}
		return nil

	case "verify":
		fs := flag.NewFlagSet("audit verify", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var projectRoot string
		var level string
		var artifactDir string
		fs.StringVar(&projectRoot, "project-root", "", "project root")
		fs.StringVar(&level, "level", audit.LevelA2, "audit integrity level")
		fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		root, err := onboarding.DetectProjectRoot(projectRoot)
		if err != nil {
			return err
		}
		err = audit.VerifyIntegrity(filepath.Join(root, ".atrakta", "audit"), level)
		out := map[string]any{
			"project_root": root,
			"level":        level,
			"ok":           err == nil,
		}
		if err != nil {
			out["error"] = err.Error()
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
		if artifactDir != "" {
			if err := writeArtifact(artifactDir, "audit.verify.json", out); err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported audit subcommand %q", sub)
	}
}

func runAlias(alias string, args []string) error {
	fs := flag.NewFlagSet(alias, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var failureTier string
	var execute bool
	var onboardRoot string
	var artifactDir string
	fs.StringVar(&failureTier, "failure-tier", "", "failure tier ceiling")
	fs.BoolVar(&execute, "execute", false, "execute mapped pipeline mode")
	fs.StringVar(&onboardRoot, "onboard-root", "", "project root for onboarding-derived failure routing")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	capOut := resolveoperationcapability.ResolveOperationCapability(resolveoperationcapability.Input{
		CommandOrAlias: alias,
		FailureTier:    failureTier,
	})
	response := map[string]any{
		"alias":      alias,
		"capability": capOut,
	}

	if execute {
		decision := capOut.Decision.(resolveoperationcapability.CapabilityDecision)
		mode := "inspect"
		switch decision.EffectiveActionClass {
		case resolveoperationcapability.ActionPropose:
			mode = "preview"
		case resolveoperationcapability.ActionApply:
			mode = "simulate"
		}
		input, err := buildDefaultInput(mode)
		if err != nil {
			return err
		}
		if onboardRoot != "" {
			onboardingBundle, err := onboarding.BuildOnboardingProposal(onboardRoot)
			if err != nil {
				return err
			}
			if err := validation.ValidateOnboardingProposal(onboardingBundle); err != nil {
				return err
			}
			input = applyOnboardingFailure(input, onboardingBundle)
		}
		bundle, err := executeBundle(mode, input)
		if err != nil {
			return err
		}
		response["mode"] = mode
		response["bundle"] = bundle
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(response); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, alias+".alias.json", response); err != nil {
			return err
		}
	}
	return nil
}

func runExtensions(args []string) error {
	fs := flag.NewFlagSet("extensions", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	out, err := extensions.Resolve(projectRoot)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "extensions.resolve.json", out); err != nil {
			return err
		}
	}
	return nil
}

func runVerifyCoverage(args []string) error {
	fs := flag.NewFlagSet("verify-coverage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validation.VerifyOperationsSchemaCoverage(""); err != nil {
		return err
	}
	if err := fixtures.VerifyResolverFixtureCoverage(""); err != nil {
		return err
	}
	return nil
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  primary:")
	fmt.Println("  atrakta run [--project-root dir] [--interface id] [--non-interactive] [--json] [--apply] [--approve]")
	fmt.Println("  debug/auxiliary:")
	fmt.Println("  atrakta inspect [--input bundle.json] [--onboard-root dir] [--artifact-dir dir]")
	fmt.Println("  atrakta preview [--input bundle.json] [--onboard-root dir] [--artifact-dir dir]")
	fmt.Println("  atrakta simulate [--input bundle.json] [--onboard-root dir] [--artifact-dir dir]")
	fmt.Println("  atrakta run-fixtures [--artifact-dir dir]")
	fmt.Println("  atrakta onboard [--project-root dir] [--artifact-dir dir]")
	fmt.Println("  atrakta accept [--project-root dir] [--proposal proposal.json] [--artifact-dir dir]")
	fmt.Println("  atrakta mutate <inspect|propose|apply> --target path [--content text|--content-file file] [--allow]")
	fmt.Println("  atrakta audit <append|verify> [flags]")
	fmt.Println("  atrakta doctor [--execute]")
	fmt.Println("  atrakta parity [--execute]")
	fmt.Println("  atrakta integration [--execute]")
	fmt.Println("  atrakta extensions [--project-root dir]")
	fmt.Println("  atrakta export-snapshots [--dir fixtures/snapshots]")
	fmt.Println("  atrakta verify-coverage")
}

func writeArtifact(dir, name string, payload any) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func buildDefaultInput(mode string) (pipeline.BundleInput, error) {
	if err := validation.ValidateMode(mode); err != nil {
		return pipeline.BundleInput{}, err
	}
	input := pipeline.DefaultInput(mode)
	if err := validation.ValidateBundleInput(input); err != nil {
		return pipeline.BundleInput{}, err
	}
	return input, nil
}

func executeBundle(mode string, input pipeline.BundleInput) (pipeline.BundleOutput, error) {
	out, err := pipeline.ExecuteOrdered(mode, input)
	if err != nil {
		return pipeline.BundleOutput{}, err
	}
	if err := validation.ValidateBundleOutput(out); err != nil {
		return pipeline.BundleOutput{}, err
	}
	return out, nil
}

func runFixturesReport() (fixtures.Report, error) {
	fixturesDir, err := resolveFixturesDir()
	if err != nil {
		return fixtures.Report{}, err
	}
	report, err := fixtures.RunAll(fixturesDir)
	if err != nil {
		return fixtures.Report{}, err
	}
	if err := validation.ValidateFixtureReport(report); err != nil {
		return fixtures.Report{}, err
	}
	return report, nil
}

func resolveFixturesDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "fixtures")
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("fixtures directory not found from current path")
}

func applyOnboardingFailure(input pipeline.BundleInput, bundle onboarding.ProposalBundle) pipeline.BundleInput {
	input.FailureClass = bundle.InferredFailure.FailureClass
	input.FailureContext.Scope = bundle.InferredFailure.Scope
	input.FailureContext.Triggers = append([]string{}, bundle.InferredFailure.Triggers...)
	input.FailureContext.IsDiagnosticsOnly = len(bundle.Conflicts) == 0
	return input
}
