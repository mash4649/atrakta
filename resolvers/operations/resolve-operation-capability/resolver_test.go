package resolveoperationcapability

import "testing"

func TestAliasMapping(t *testing.T) {
	out := ResolveOperationCapability(Input{CommandOrAlias: "doctor"})
	d := out.Decision.(CapabilityDecision)
	if d.CanonicalCapability != "inspect_health" {
		t.Fatalf("capability=%q", d.CanonicalCapability)
	}
	if d.ActionClass != ActionInspect {
		t.Fatalf("action=%q", d.ActionClass)
	}
}

func TestBlockCeilingToInspectOnly(t *testing.T) {
	out := ResolveOperationCapability(Input{CommandOrAlias: "apply_repair", FailureTier: "BLOCK"})
	d := out.Decision.(CapabilityDecision)
	if d.EffectiveActionClass != ActionInspect {
		t.Fatalf("effective=%q", d.EffectiveActionClass)
	}
	if out.NextAllowedAction != "inspect" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}

func TestProposalOnlyCeiling(t *testing.T) {
	out := ResolveOperationCapability(Input{CommandOrAlias: "apply_repair", FailureTier: "PROPOSAL_ONLY"})
	d := out.Decision.(CapabilityDecision)
	if d.EffectiveActionClass != ActionPropose {
		t.Fatalf("effective=%q", d.EffectiveActionClass)
	}
}
