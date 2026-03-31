package main

import (
	"fmt"
	"os"

	"github.com/mash4649/atrakta/v0/internal/mutation"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/persist"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
	"github.com/mash4649/atrakta/v0/internal/startfast"
	"github.com/mash4649/atrakta/v0/internal/validation"
	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
)

func runOnboardingPath(commandName, root string, iface runpkg.InterfaceResolution, nonInteractive, approve, jsonOut bool, artifactDir string) (int, error) {
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
	_, approvalOutcome, err := evaluateApproval(approve, nonInteractive, "Accept onboarding proposal and persist canonical/state/audit?")
	if err != nil {
		return exitRuntimeError, err
	}
	switch approvalOutcome {
	case approvalOutcomeRequiresExplicit:
		resp.Status = "needs_approval"
		resp.Message = "explicit approval is required before onboarding persistence"
		resp.NextAllowedAction = "re-run with --approve, or run interactively and approve, or run `atrakta accept --project-root <dir>`"
		resp.ApprovalScope = "onboarding_accept"
		if err := appendSessionLifecycleEvent(commandName, root, "gate.result", map[string]any{
			"path":                resp.Path,
			"status":              resp.Status,
			"approval_scope":      resp.ApprovalScope,
			"interface_id":        iface.InterfaceID,
			"interface_source":    iface.Source,
			"next_allowed_action": resp.NextAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitNeedsApproval, nil
	case approvalOutcomeRejected:
		resp.Status = "needs_approval"
		resp.Message = "onboarding proposal was not approved"
		resp.NextAllowedAction = fmt.Sprintf("re-run `atrakta %s --approve` or approve persistence when prompted", commandName)
		resp.ApprovalScope = "onboarding_accept"
		if err := appendSessionLifecycleEvent(commandName, root, "gate.result", map[string]any{
			"path":                resp.Path,
			"status":              resp.Status,
			"approval_scope":      resp.ApprovalScope,
			"interface_id":        iface.InterfaceID,
			"interface_source":    iface.Source,
			"next_allowed_action": resp.NextAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
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
	endEvent := runEventStartEnd
	if commandName == "resume" {
		endEvent = runEventResumeEnd
	}
	if err := appendSessionLifecycleEvent(commandName, root, endEvent, map[string]any{
		"status":               resp.Status,
		"path":                 resp.Path,
		"interface_id":         iface.InterfaceID,
		"interface_source":     iface.Source,
		"final_allowed_action": resp.NextAllowedAction,
	}); err != nil {
		return exitRuntimeError, err
	}
	if err := saveStartFastSnapshot(commandName, root, iface, false); err != nil {
		return exitRuntimeError, err
	}
	if err := saveStartAutoState(commandName, root, iface); err != nil {
		return exitRuntimeError, err
	}
	if err := saveSessionRuntimeArtifacts(commandName, root, resp, nil, nil, false); err != nil {
		return exitRuntimeError, err
	}
	if err := saveStartHandoff(commandName, root, resp, false, nil, nil); err != nil {
		return exitRuntimeError, err
	}
	if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func runNormalPath(commandName, root, canonicalState string, iface runpkg.InterfaceResolution, canonicalSummary map[string]any, apply, approve, nonInteractive, jsonOut bool, artifactDir string) (int, error) {
	auditPreflight := ""
	if v, ok := canonicalSummary["audit_preflight"].(string); ok {
		auditPreflight = v
	}
	if isSessionCommand(commandName) && !apply {
		fastKey, err := startfast.ComputeKey(root, iface.InterfaceID, false)
		if err != nil {
			return exitRuntimeError, err
		}
		snapshot, err := startfast.LoadSnapshot(root)
		if err == nil && startfast.IsMatch(snapshot, fastKey) && snapshot.InterfaceID == iface.InterfaceID && !snapshot.ApplyRequested {
			portabilityInput, err := buildPortabilityInput(root, iface, true)
			if err != nil {
				return exitRuntimeError, err
			}
			portabilityOut := resolvesurfaceportability.ResolveSurfacePortability(portabilityInput)
			portabilityDecision := portabilityOut.Decision.(resolvesurfaceportability.PortabilityDecision)

			canonicalSummary["fast_path"] = true
			canonicalSummary["fast_path_key"] = fastKey.Key

			resp := runResponse{
				Status:            "ok",
				Path:              "normal",
				ProjectRoot:       root,
				CanonicalState:    canonicalState,
				CanonicalSummary:  canonicalSummary,
				Interface:         iface,
				ApplyRequested:    false,
				Approved:          false,
				Message:           fmt.Sprintf("%s fast-path used; workspace unchanged", commandName),
				Portability:       portabilityDecision,
				ResolvedTargets:   resolvedProjectionTargets(portabilityDecision),
				DegradedSurfaces:  portabilityDecision.DegradedTargets,
				MissingTargets:    portabilityDecision.UnsupportedTargets,
				PortabilityStatus: portabilityDecision.PortabilityStatus,
				PortabilityReason: portabilityOut.Reason,
				NextAllowedAction: "inspect",
			}
			if err := appendRunAuditEvent(commandName, root, map[string]any{
				"status":               resp.Status,
				"path":                 resp.Path,
				"interface_id":         iface.InterfaceID,
				"interface_source":     iface.Source,
				"apply_requested":      false,
				"fast_path":            true,
				"fast_path_key":        fastKey.Key,
				"audit_preflight":      auditPreflight,
				"final_allowed_action": "inspect",
			}); err != nil {
				return exitRuntimeError, err
			}
			if err := saveStartAutoState(commandName, root, iface); err != nil {
				return exitRuntimeError, err
			}
			if err := saveStartHandoff(commandName, root, resp, true, nil, nil); err != nil {
				return exitRuntimeError, err
			}
			if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
				return exitRuntimeError, err
			}
			endEvent := runEventStartEnd
			if commandName == "resume" {
				endEvent = runEventResumeEnd
			}
			if err := appendSessionLifecycleEvent(commandName, root, endEvent, map[string]any{
				"path":                 resp.Path,
				"status":               resp.Status,
				"interface_id":         iface.InterfaceID,
				"interface_source":     iface.Source,
				"final_allowed_action": "inspect",
			}); err != nil {
				return exitRuntimeError, err
			}
			return exitOK, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return exitRuntimeError, err
		}
	}

	input, err := buildRunInspectInput(root, iface, apply, true)
	if err != nil {
		return exitRuntimeError, err
	}
	out, err := executeBundle("inspect", input)
	if err != nil {
		return exitRuntimeError, err
	}
	if err := appendSessionLifecycleEvent(commandName, root, "detect.performed", map[string]any{
		"path":                 "normal",
		"canonical_state":      canonicalState,
		"interface_id":         iface.InterfaceID,
		"interface_source":     iface.Source,
		"apply_requested":      apply,
		"step_count":           len(out.Steps),
		"final_allowed_action": out.FinalAllowedAction,
	}); err != nil {
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
	plannedTargets := planTargetPaths(plans)
	if err := appendSessionLifecycleEvent(commandName, root, "plan.created", map[string]any{
		"path":                 "normal",
		"canonical_state":      canonicalState,
		"interface_id":         iface.InterfaceID,
		"interface_source":     iface.Source,
		"apply_requested":      apply,
		"planned_target_paths": plannedTargets,
		"planned_target_count": len(plannedTargets),
		"portability_status":   portabilityDecision.PortabilityStatus,
		"portability_reason":   portabilityReason,
		"final_allowed_action": resp.NextAllowedAction,
	}); err != nil {
		return exitRuntimeError, err
	}

	if apply {
		if err := appendRunAuditEvent(commandName, root, map[string]any{
			"path":                 resp.Path,
			"status":               resp.Status,
			"interface_id":         iface.InterfaceID,
			"interface_source":     iface.Source,
			"apply_begin":          true,
			"apply_requested":      true,
			"planned_target_count": len(plannedTargets),
			"final_allowed_action": resp.NextAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
	}

	if apply && portabilityDecision.PortabilityStatus != resolvesurfaceportability.PortabilitySupported {
		resp.Message = "apply disabled because requested surface portability is proposal-only"
		resp.NextAllowedAction = "propose"
		if err := appendSessionLifecycleEvent(commandName, root, "gate.result", map[string]any{
			"path":                resp.Path,
			"status":              resp.Status,
			"interface_id":        iface.InterfaceID,
			"interface_source":    iface.Source,
			"portability_status":  portabilityDecision.PortabilityStatus,
			"next_allowed_action": resp.NextAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := appendRunAuditEvent(commandName, root, map[string]any{
			"status":               resp.Status,
			"path":                 resp.Path,
			"interface_id":         iface.InterfaceID,
			"interface_source":     iface.Source,
			"apply_requested":      apply,
			"approved":             false,
			"portability_status":   portabilityDecision.PortabilityStatus,
			"audit_preflight":      auditPreflight,
			"final_allowed_action": out.FinalAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := saveStartFastSnapshot(commandName, root, iface, apply); err != nil {
			return exitRuntimeError, err
		}
		if err := saveStartAutoState(commandName, root, iface); err != nil {
			return exitRuntimeError, err
		}
		if err := saveSessionRuntimeArtifacts(commandName, root, resp, plannedTargets, nil, false); err != nil {
			return exitRuntimeError, err
		}
		if err := saveStartHandoff(commandName, root, resp, false, plannedTargets, nil); err != nil {
			return exitRuntimeError, err
		}
		if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if apply {
		_, approvalOutcome, err := evaluateApproval(approve, nonInteractive, "Apply managed mutation plan to generated managed targets?")
		if err != nil {
			return exitRuntimeError, err
		}
		switch approvalOutcome {
		case approvalOutcomeRequiresExplicit:
			resp.Status = "needs_approval"
			resp.Message = "apply route requires explicit managed mutation approval"
			resp.NextAllowedAction = "re-run with --apply --approve to execute managed apply"
			resp.ApprovalScope = "managed_apply"
			if err := appendSessionLifecycleEvent(commandName, root, "gate.result", map[string]any{
				"path":                resp.Path,
				"status":              resp.Status,
				"approval_scope":      resp.ApprovalScope,
				"interface_id":        iface.InterfaceID,
				"interface_source":    iface.Source,
				"next_allowed_action": resp.NextAllowedAction,
			}); err != nil {
				return exitRuntimeError, err
			}
			if err := appendRunAuditEvent(commandName, root, map[string]any{
				"status":               resp.Status,
				"path":                 resp.Path,
				"interface_id":         iface.InterfaceID,
				"interface_source":     iface.Source,
				"apply_requested":      apply,
				"approved":             false,
				"audit_preflight":      auditPreflight,
				"final_allowed_action": out.FinalAllowedAction,
			}); err != nil {
				return exitRuntimeError, err
			}
			if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
				return exitRuntimeError, err
			}
			return exitNeedsApproval, nil
		case approvalOutcomeRejected:
			resp.Status = "needs_approval"
			resp.Message = "apply plan was not approved"
			resp.NextAllowedAction = "re-run with --apply --approve when ready"
			resp.ApprovalScope = "managed_apply"
			if err := appendSessionLifecycleEvent(commandName, root, "gate.result", map[string]any{
				"path":                resp.Path,
				"status":              resp.Status,
				"approval_scope":      resp.ApprovalScope,
				"interface_id":        iface.InterfaceID,
				"interface_source":    iface.Source,
				"next_allowed_action": resp.NextAllowedAction,
			}); err != nil {
				return exitRuntimeError, err
			}
			if err := appendRunAuditEvent(commandName, root, map[string]any{
				"status":               resp.Status,
				"path":                 resp.Path,
				"interface_id":         iface.InterfaceID,
				"interface_source":     iface.Source,
				"apply_requested":      apply,
				"approved":             false,
				"audit_preflight":      auditPreflight,
				"final_allowed_action": out.FinalAllowedAction,
			}); err != nil {
				return exitRuntimeError, err
			}
			if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
				return exitRuntimeError, err
			}
			return exitNeedsApproval, nil
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
		if err := appendRunAuditEvent(commandName, root, map[string]any{
			"status":               resp.Status,
			"path":                 resp.Path,
			"interface_id":         iface.InterfaceID,
			"interface_source":     iface.Source,
			"apply_requested":      apply,
			"approved":             true,
			"apply_performed":      true,
			"audit_preflight":      auditPreflight,
			"applied_target_paths": appliedTargets,
			"applied_count":        len(appliedTargets),
			"final_allowed_action": out.FinalAllowedAction,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := saveStartFastSnapshot(commandName, root, iface, apply); err != nil {
			return exitRuntimeError, err
		}
		if err := saveStartAutoState(commandName, root, iface); err != nil {
			return exitRuntimeError, err
		}
		if err := saveSessionRuntimeArtifacts(commandName, root, resp, plannedTargets, appliedTargets, false); err != nil {
			return exitRuntimeError, err
		}
		if err := saveStartHandoff(commandName, root, resp, false, plannedTargets, appliedTargets); err != nil {
			return exitRuntimeError, err
		}
		if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if err := appendRunAuditEvent(commandName, root, map[string]any{
		"status":               resp.Status,
		"path":                 resp.Path,
		"interface_id":         iface.InterfaceID,
		"interface_source":     iface.Source,
		"apply_requested":      apply,
		"audit_preflight":      auditPreflight,
		"final_allowed_action": out.FinalAllowedAction,
	}); err != nil {
		return exitRuntimeError, err
	}
	if err := appendSessionLifecycleEvent(commandName, root, "gate.result", map[string]any{
		"path":                resp.Path,
		"status":              resp.Status,
		"interface_id":        iface.InterfaceID,
		"interface_source":    iface.Source,
		"next_allowed_action": resp.NextAllowedAction,
	}); err != nil {
		return exitRuntimeError, err
	}
	if err := saveStartFastSnapshot(commandName, root, iface, apply); err != nil {
		return exitRuntimeError, err
	}
	if err := saveStartAutoState(commandName, root, iface); err != nil {
		return exitRuntimeError, err
	}
	if err := saveSessionRuntimeArtifacts(commandName, root, resp, plannedTargets, nil, false); err != nil {
		return exitRuntimeError, err
	}
	if err := saveStartHandoff(commandName, root, resp, false, plannedTargets, nil); err != nil {
		return exitRuntimeError, err
	}
	if err := emitRunResponse(commandName, resp, jsonOut, artifactDir); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}
