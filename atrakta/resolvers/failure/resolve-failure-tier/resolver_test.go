package resolvefailuretier

import "testing"

func TestFailureDefaultsExist(t *testing.T) {
	classes := []string{
		"policy_failure",
		"approval_failure",
		"capability_resolution_failure",
		"projection_failure",
		"adapter_execution_failure",
		"provenance_failure",
		"audit_integrity_failure",
		"legacy_conflict_failure",
	}
	for _, c := range classes {
		out := ResolveFailureTier(c, Context{Scope: "task"})
		d := out.Decision.(FailureDecision)
		if d.DefaultTier == "" {
			t.Fatalf("class %s has empty default tier", c)
		}
		if d.Scope != "task" {
			t.Fatalf("scope = %q", d.Scope)
		}
	}
}

func TestStrictTriggerEscalation(t *testing.T) {
	out := ResolveFailureTier("legacy_conflict_failure", Context{Scope: "workspace", Triggers: []string{"policy_ambiguity"}})
	d := out.Decision.(FailureDecision)
	if d.StrictTransition != "strict" {
		t.Fatalf("strict transition = %q", d.StrictTransition)
	}
	if d.Scope != "workspace" {
		t.Fatalf("scope = %q", d.Scope)
	}
}

func TestProjectionFailureStopsProjectionNotExecution(t *testing.T) {
	out := ResolveFailureTier("projection_failure", Context{Scope: "request", IsDiagnosticsOnly: false})
	d := out.Decision.(FailureDecision)
	if !d.ExecutionAllowed {
		t.Fatalf("execution should stay allowed")
	}
	if d.ProjectionAllowed {
		t.Fatalf("projection should be stopped")
	}
}

func TestProposalOnlyCeiling(t *testing.T) {
	out := ResolveFailureTier("capability_resolution_failure", Context{Scope: "task"})
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next action = %q", out.NextAllowedAction)
	}
}
