package resolveguidanceprecedence

import "testing"

func TestResolveGuidancePrecedenceOrder(t *testing.T) {
	items := []GuidanceItem{
		{ID: "repo", Type: "repo_map"},
		{ID: "policy", Type: "policy"},
		{ID: "skill", Type: "skill"},
		{ID: "workflow", Type: "workflow"},
		{ID: "hint", Type: "tool_hint"},
	}

	out := ResolveGuidancePrecedence(items)
	decision, ok := out.Decision.(GuidanceDecision)
	if !ok {
		t.Fatalf("decision type mismatch")
	}
	want := []string{"policy", "workflow", "skill", "repo", "hint"}
	for i := range want {
		if decision.OrderedIDs[i] != want[i] {
			t.Fatalf("ordered[%d] = %q, want %q", i, decision.OrderedIDs[i], want[i])
		}
	}
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next = %q, want propose", out.NextAllowedAction)
	}
}

func TestResolveGuidancePrecedenceViolation(t *testing.T) {
	items := []GuidanceItem{
		{ID: "repo", Type: "repo_map", ClaimsDecisionOverride: true},
	}
	out := ResolveGuidancePrecedence(items)
	if out.NextAllowedAction != "deny" {
		t.Fatalf("next = %q, want deny", out.NextAllowedAction)
	}
	if out.Reason != "guidance policy violations detected" {
		t.Fatalf("reason = %q", out.Reason)
	}
}

func TestLegacyUnmappedIsAdvisory(t *testing.T) {
	items := []GuidanceItem{{ID: "legacy", Type: "legacy", MappedToCanonicalPolicy: false}}
	out := ResolveGuidancePrecedence(items)
	decision := out.Decision.(GuidanceDecision)
	if decision.Classifications[0].Strength != StrengthAdvisory {
		t.Fatalf("strength = %q, want %q", decision.Classifications[0].Strength, StrengthAdvisory)
	}
}

func TestLegacyMappedBecomesAuthoritative(t *testing.T) {
	items := []GuidanceItem{{ID: "legacy", Type: "legacy", MappedToCanonicalPolicy: true}}
	out := ResolveGuidancePrecedence(items)
	decision := out.Decision.(GuidanceDecision)
	if decision.Classifications[0].Strength != StrengthAuthoritative {
		t.Fatalf("strength = %q, want %q", decision.Classifications[0].Strength, StrengthAuthoritative)
	}
	if decision.Classifications[0].Precedence != 1 {
		t.Fatalf("precedence = %d, want 1", decision.Classifications[0].Precedence)
	}
}
