package strictstatemachine

import "testing"

func TestStrictCanBeReleasedAndReset(t *testing.T) {
	toStrict := Transition(StateInput{CurrentState: "normal", Event: "trigger_strict", Scope: "task"})
	d1 := toStrict.Decision.(StateDecision)
	if d1.NextState != "strict" {
		t.Fatalf("next state = %q", d1.NextState)
	}

	toReleased := Transition(StateInput{CurrentState: d1.NextState, Event: "release_approved", Scope: "task", ReleaseApproved: true})
	d2 := toReleased.Decision.(StateDecision)
	if d2.NextState != "released" {
		t.Fatalf("next state = %q", d2.NextState)
	}

	toNormal := Transition(StateInput{CurrentState: d2.NextState, Event: "reset", Scope: "task"})
	d3 := toNormal.Decision.(StateDecision)
	if d3.NextState != "normal" {
		t.Fatalf("next state = %q", d3.NextState)
	}
}

func TestGuardedAllowsNoApply(t *testing.T) {
	out := Transition(StateInput{CurrentState: "normal", Event: "trigger_guarded", Scope: "request"})
	d := out.Decision.(StateDecision)
	for _, act := range d.AllowedActions {
		if act == "apply" {
			t.Fatalf("guarded should not allow apply")
		}
	}
}

func TestReleaseApprovalRequired(t *testing.T) {
	out := Transition(StateInput{CurrentState: "strict", Event: "release_approved", Scope: "workspace", ReleaseApproved: false})
	d := out.Decision.(StateDecision)
	if d.NextState != "strict" {
		t.Fatalf("next state = %q", d.NextState)
	}
}
