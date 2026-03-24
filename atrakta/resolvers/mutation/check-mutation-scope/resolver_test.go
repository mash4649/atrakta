package checkmutationscope

import "testing"

func TestScopeResolutionDefaults(t *testing.T) {
	cases := []struct {
		name string
		in   Target
		want string
		next string
	}{
		{name: "managed block", in: Target{Path: ".harness/canonical/policies/p.json"}, want: ScopeManagedBlock, next: "apply"},
		{name: "generated projection", in: Target{Path: ".harness/generated/repo-map.generated.json"}, want: ScopeGeneratedProjection, next: "apply"},
		{name: "proposal patch only", in: Target{Path: "AGENTS.md"}, want: ScopeProposalPatchOnly, next: "propose"},
		{name: "unmanaged user region", in: Target{Path: "src/main.go"}, want: ScopeUnmanagedUser, next: "propose"},
	}
	for _, tc := range cases {
		out := CheckMutationScope(tc.in)
		d := out.Decision.(MutationDecision)
		if d.Scope != tc.want {
			t.Fatalf("%s: scope=%q want=%q", tc.name, d.Scope, tc.want)
		}
		if out.NextAllowedAction != tc.next {
			t.Fatalf("%s: next=%q want=%q", tc.name, out.NextAllowedAction, tc.next)
		}
	}
}

func TestPolicyReplaceDisallowedOutsideManagedOnly(t *testing.T) {
	out := CheckMutationScope(Target{Path: ".harness/canonical/policies/p.json", AssetType: "policy", Operation: "replace", ManagedOnlyPath: false})
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next=%q want=propose", out.NextAllowedAction)
	}
	d := out.Decision.(MutationDecision)
	if d.Policy == "" {
		t.Fatalf("policy should not be empty")
	}
}

func TestExistingUserRulesAmbiguityFallsBackToProposalOnly(t *testing.T) {
	out := CheckMutationScope(Target{Path: "AGENTS.md", AssetType: "existing_user_rules", HasAmbiguity: true})
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next=%q want=propose", out.NextAllowedAction)
	}
	d := out.Decision.(MutationDecision)
	if d.ImplicitMutationAllowed {
		t.Fatalf("implicit mutation should be false")
	}
}
