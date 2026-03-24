package checkprojectioneligibility

import "testing"

func TestAllowedSources(t *testing.T) {
	tests := []Source{
		{Type: "policy"},
		{Type: "repo_map"},
		{Type: "skill"},
		{Type: "workflow"},
	}
	for _, in := range tests {
		out := CheckProjectionEligibility(in)
		decision := out.Decision.(ProjectionDecision)
		if decision.Eligibility != EligibilityAllowed {
			t.Fatalf("type %q eligibility = %q", in.Type, decision.Eligibility)
		}
		if !decision.CanBeDecisionRoot {
			t.Fatalf("type %q should be decision root", in.Type)
		}
	}
}

func TestConditionalNeedsAnchor(t *testing.T) {
	out := CheckProjectionEligibility(Source{Type: "decision", HasCanonicalAnchor: false})
	decision := out.Decision.(ProjectionDecision)
	if decision.Eligibility != EligibilityConditional {
		t.Fatalf("eligibility = %q", decision.Eligibility)
	}
	if decision.CanBeDecisionRoot {
		t.Fatalf("conditional source should not be decision root without anchor")
	}
	if out.NextAllowedAction != "inspect" {
		t.Fatalf("next = %q", out.NextAllowedAction)
	}
}

func TestForbiddenSources(t *testing.T) {
	tests := []Source{{Type: "task_state"}, {Type: "audit_event"}}
	for _, in := range tests {
		out := CheckProjectionEligibility(in)
		decision := out.Decision.(ProjectionDecision)
		if decision.Eligibility != EligibilityForbidden {
			t.Fatalf("type %q eligibility = %q", in.Type, decision.Eligibility)
		}
		if out.NextAllowedAction != "deny" {
			t.Fatalf("next = %q", out.NextAllowedAction)
		}
	}
}
