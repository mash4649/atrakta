package resolveextensionorder

import "testing"

func TestDeterministicOrder(t *testing.T) {
	items := []Item{
		{ID: "skill-1", Kind: "skill"},
		{ID: "policy-1", Kind: "policy"},
		{ID: "workflow-1", Kind: "workflow"},
		{ID: "hint-1", Kind: "tool_hint"},
		{ID: "hook-1", Kind: "hook"},
		{ID: "plugin-1", Kind: "projection_plugin"},
	}
	out := ResolveExtensionOrder(items)
	d := out.Decision.(ExtensionDecision)
	want := []string{"policy-1", "workflow-1", "skill-1", "hint-1", "hook-1", "plugin-1"}
	for i := range want {
		if d.Ordered[i].ID != want[i] {
			t.Fatalf("ordered[%d]=%q want=%q", i, d.Ordered[i].ID, want[i])
		}
	}
}

func TestCoreMutationViolation(t *testing.T) {
	items := []Item{{ID: "hook-1", Kind: "hook", AttemptsCoreMutation: true, HookMutatesCanonical: true}}
	out := ResolveExtensionOrder(items)
	if out.NextAllowedAction != "deny" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
	d := out.Decision.(ExtensionDecision)
	if len(d.Violations) == 0 {
		t.Fatalf("expected violations")
	}
}
