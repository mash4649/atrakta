package entry

import "errors"

import runpkg "github.com/mash4649/atrakta/v0/internal/run"

type ExecuteInput struct {
	ProjectRoot      string
	InterfaceID      string
	TriggerInterface string
	ApplyRequested   bool
}

type NeedsInputResponse struct {
	Status            string
	Path              string
	ProjectRoot       string
	CanonicalState    string
	Interface         runpkg.InterfaceResolution
	Message           string
	PortabilityReason string
	NextAllowedAction string
	RequiredInputs    []string
}

type ExecuteResponse struct {
	Decision   Decision
	NeedsInput *NeedsInputResponse
	HardError  *NeedsInputResponse
}

// Execute resolves canonical/interface entry routing and returns a deterministic exit contract.
func Execute(input ExecuteInput) (int, ExecuteResponse, error) {
	resolved, err := Resolve(Input{
		ProjectRoot:      input.ProjectRoot,
		InterfaceID:      input.InterfaceID,
		TriggerInterface: input.TriggerInterface,
	})
	if err != nil {
		var blocked *BlockedStateError
		if errors.As(err, &blocked) {
			resp := blocked.Response()
			return 1, ExecuteResponse{HardError: &resp}, nil
		}
		return 1, ExecuteResponse{}, err
	}
	if resolved.NeedsInput {
		return 2, ExecuteResponse{
			Decision: resolved,
			NeedsInput: &NeedsInputResponse{
				Status:            "needs_input",
				Path:              PathNormal,
				ProjectRoot:       resolved.ProjectRoot,
				CanonicalState:    resolved.CanonicalState,
				Interface:         resolved.Interface,
				Message:           "interface could not be resolved deterministically",
				PortabilityReason: "interface unresolved",
				NextAllowedAction: "re-run with --interface <id> or set ATRAKTA_TRIGGER_INTERFACE",
				RequiredInputs:    []string{"interface"},
			},
		}, nil
	}
	return 0, ExecuteResponse{Decision: resolved}, nil
}
