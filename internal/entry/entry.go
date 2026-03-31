package entry

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

const (
	PathOnboarding = "onboarding"
	PathNormal     = "normal"
)

type Input struct {
	ProjectRoot      string
	InterfaceID      string
	TriggerInterface string
}

type Decision struct {
	Path           string
	ProjectRoot    string
	CanonicalState string
	Interface      runpkg.InterfaceResolution
	NeedsInput     bool
}

type BlockedStateError struct {
	ProjectRoot       string
	CanonicalState    string
	Interface         runpkg.InterfaceResolution
	Diagnostic        string
	NextAllowedAction string
	RequiredInputs    []string
}

func (e *BlockedStateError) Error() string {
	return e.Diagnostic
}

func (e *BlockedStateError) Response() NeedsInputResponse {
	return NeedsInputResponse{
		Status:            "needs_input",
		Path:              PathNormal,
		ProjectRoot:       e.ProjectRoot,
		CanonicalState:    e.CanonicalState,
		Interface:         e.Interface,
		Message:           e.Diagnostic,
		PortabilityReason: e.Diagnostic,
		NextAllowedAction: e.NextAllowedAction,
		RequiredInputs:    e.RequiredInputs,
	}
}

func Resolve(input Input) (Decision, error) {
	root, err := onboarding.DetectProjectRoot(input.ProjectRoot)
	if err != nil {
		return Decision{}, err
	}
	canonicalState, err := runpkg.DetectCanonicalState(root)
	if err != nil {
		return Decision{}, err
	}
	if canonicalState == runpkg.StatePartialState || canonicalState == runpkg.StateCorruptState {
		return Decision{}, blockedStateError(root, canonicalState)
	}

	iface, err := resolveInterface(root, canonicalState, input.InterfaceID, input.TriggerInterface)
	if err != nil {
		if errors.Is(err, runpkg.ErrInterfaceUnresolved) {
			if canonicalState == runpkg.StateNone {
				iface = runpkg.InterfaceResolution{InterfaceID: "generic-cli", Source: "default"}
			} else {
				return Decision{
					Path:           PathNormal,
					ProjectRoot:    root,
					CanonicalState: canonicalState,
					Interface:      runpkg.InterfaceResolution{InterfaceID: "unresolved", Source: "detect"},
					NeedsInput:     true,
				}, nil
			}
		} else {
			return Decision{}, err
		}
	}

	switch canonicalState {
	case runpkg.StateNone:
		return Decision{
			Path:           PathOnboarding,
			ProjectRoot:    root,
			CanonicalState: canonicalState,
			Interface:      iface,
		}, nil
	case runpkg.StateCanonicalPresent, runpkg.StateOnboardingComplete:
		return Decision{
			Path:           PathNormal,
			ProjectRoot:    root,
			CanonicalState: canonicalState,
			Interface:      iface,
		}, nil
	default:
		return Decision{}, fmt.Errorf("unsupported canonical state %q", canonicalState)
	}
}

func blockedStateError(root, canonicalState string) error {
	switch canonicalState {
	case runpkg.StatePartialState:
		return &BlockedStateError{
			ProjectRoot:       root,
			CanonicalState:    canonicalState,
			Interface:         runpkg.InterfaceResolution{InterfaceID: "unresolved", Source: "detect"},
			Diagnostic:        "run blocked: canonical state is partial_state; restore .atrakta/canonical/policies/registry/index.json or remove .atrakta/state/onboarding-state.json, then re-run atrakta start",
			NextAllowedAction: "restore .atrakta/canonical/policies/registry/index.json or delete .atrakta/state/onboarding-state.json, then re-run `atrakta start`",
			RequiredInputs:    []string{"canonical_state"},
		}
	case runpkg.StateCorruptState:
		return &BlockedStateError{
			ProjectRoot:       root,
			CanonicalState:    canonicalState,
			Interface:         runpkg.InterfaceResolution{InterfaceID: "unresolved", Source: "detect"},
			Diagnostic:        "run blocked: canonical state is corrupt_state; repair or delete the incomplete .atrakta/canonical/ and .atrakta/state/ directories, then re-run atrakta start",
			NextAllowedAction: "repair or delete the incomplete .atrakta/canonical/ and .atrakta/state/ directories, then re-run `atrakta start`",
			RequiredInputs:    []string{"canonical_state"},
		}
	default:
		return fmt.Errorf("unsupported canonical state %q", canonicalState)
	}
}

func resolveInterface(root, canonicalState, explicit, trigger string) (runpkg.InterfaceResolution, error) {
	trimmedExplicit := strings.TrimSpace(explicit)
	trimmedTrigger := strings.TrimSpace(trigger)
	if trimmedExplicit != "" || trimmedTrigger != "" {
		return runpkg.ResolveInterface(root, trimmedExplicit, trimmedTrigger)
	}
	if canonicalState != runpkg.StateNone {
		auto, err := runpkg.LoadAutoState(root)
		if err == nil && strings.TrimSpace(auto.InterfaceID) != "" {
			return runpkg.InterfaceResolution{InterfaceID: auto.InterfaceID, Source: "auto"}, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return runpkg.InterfaceResolution{}, err
		}
	}
	return runpkg.ResolveInterface(root, "", "")
}
