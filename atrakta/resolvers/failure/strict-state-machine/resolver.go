package strictstatemachine

import (
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

// StateInput is strict lifecycle state machine input.
type StateInput struct {
	CurrentState    string `json:"current_state"`
	Event           string `json:"event"`
	Scope           string `json:"scope"`
	ReleaseApproved bool   `json:"release_approved,omitempty"`
}

// StateDecision is strict lifecycle transition output.
type StateDecision struct {
	CurrentState    string   `json:"current_state"`
	NextState       string   `json:"next_state"`
	Scope           string   `json:"scope"`
	AllowedActions  []string `json:"allowed_actions"`
	ReleaseRequired bool     `json:"release_required"`
}

// Transition applies strict lifecycle transition rules.
func Transition(in StateInput) common.ResolverOutput {
	current := normalizeState(in.CurrentState)
	event := normalize(in.Event)
	scope := normalizeScope(in.Scope)
	next := current
	reason := "state unchanged"

	switch current {
	case "normal":
		switch event {
		case "trigger_guarded":
			next = "guarded"
			reason = "entered guarded state"
		case "trigger_strict":
			next = "strict"
			reason = "entered strict state"
		}
	case "guarded":
		switch event {
		case "trigger_strict":
			next = "strict"
			reason = "escalated to strict"
		case "reset":
			next = "normal"
			reason = "guarded reset to normal"
		}
	case "strict":
		switch event {
		case "release_approved":
			if in.ReleaseApproved {
				next = "released"
				reason = "strict released"
			} else {
				reason = "release approval required"
			}
		case "request_release":
			reason = "release request pending approval"
		}
	case "released":
		if event == "reset" {
			next = "normal"
			reason = "released reset to normal"
		}
	}

	decision := StateDecision{
		CurrentState:    current,
		NextState:       next,
		Scope:           scope,
		AllowedActions:  allowedActions(next),
		ReleaseRequired: next == "strict",
	}

	nextAction := "inspect"
	if next == "guarded" || next == "strict" {
		nextAction = "propose"
	}
	if next == "normal" && current == "released" {
		nextAction = "apply"
	}

	return common.NewOutput(in, decision, reason, []string{"scope=" + scope, "event=" + event}, nextAction)
}

func allowedActions(state string) []string {
	switch state {
	case "normal":
		return []string{"inspect", "preview", "simulate", "propose", "apply"}
	case "guarded":
		return []string{"inspect", "preview", "simulate", "propose"}
	case "strict":
		return []string{"inspect", "preview", "simulate", "propose"}
	case "released":
		return []string{"inspect", "preview", "simulate", "propose", "apply"}
	default:
		return []string{"inspect"}
	}
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}

func normalizeState(state string) string {
	s := normalize(state)
	switch s {
	case "normal", "guarded", "strict", "released":
		return s
	default:
		return "normal"
	}
}

func normalizeScope(scope string) string {
	s := normalize(scope)
	switch s {
	case "request", "task", "workspace":
		return s
	default:
		return "task"
	}
}
