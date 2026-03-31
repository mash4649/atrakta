package main

import (
	"github.com/mash4649/atrakta/v0/internal/entry"
	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
)

func buildNeedsInputRunResponse(needsInput *entry.NeedsInputResponse, apply bool) runResponse {
	if needsInput == nil {
		return runResponse{}
	}
	return runResponse{
		Status:            needsInput.Status,
		Path:              needsInput.Path,
		ProjectRoot:       needsInput.ProjectRoot,
		CanonicalState:    needsInput.CanonicalState,
		Interface:         needsInput.Interface,
		ApplyRequested:    apply,
		Message:           needsInput.Message,
		Portability:       unsupportedPortability(needsInput.PortabilityReason),
		ResolvedTargets:   []string{},
		DegradedSurfaces:  []string{},
		MissingTargets:    []string{},
		PortabilityStatus: resolvesurfaceportability.PortabilityUnsupported,
		PortabilityReason: needsInput.PortabilityReason,
		NextAllowedAction: needsInput.NextAllowedAction,
		RequiredInputs:    needsInput.RequiredInputs,
	}
}
