package resolvesurfaceportability

import "testing"

func TestResolveSurfacePortabilitySupported(t *testing.T) {
	out := ResolveSurfacePortability(Input{
		InterfaceID:      "generic-cli",
		RequestedTargets: []string{"agents_md", "repo_docs"},
		AvailableSources: []string{"canonical_policy", "repo_docs", "agents_md"},
		BindingCapabilities: BindingCapabilities{
			InterfaceID:       "generic-cli",
			ProjectionTargets: []string{"agents_md", "repo_docs"},
			IngestSources:     []string{"canonical_policy", "repo_docs", "agents_md"},
			ApprovalChannel:   "cli_flag",
			PortabilityMode:   PortabilityModeRequired,
		},
		DegradePolicy: DegradePolicyProposalOnly,
	})
	decision := out.Decision.(PortabilityDecision)
	if decision.PortabilityStatus != PortabilitySupported {
		t.Fatalf("status=%q", decision.PortabilityStatus)
	}
	if len(decision.SupportedTargets) != 2 {
		t.Fatalf("supported=%v", decision.SupportedTargets)
	}
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}

func TestResolveSurfacePortabilityDegraded(t *testing.T) {
	out := ResolveSurfacePortability(Input{
		InterfaceID:      "cursor",
		RequestedTargets: []string{"agents_md", "repo_docs"},
		AvailableSources: []string{"canonical_policy", "agents_md", "repo_docs", "ide_rules"},
		BindingCapabilities: BindingCapabilities{
			InterfaceID:       "cursor",
			ProjectionTargets: []string{"ide_rules", "repo_docs"},
			IngestSources:     []string{"canonical_policy", "repo_docs", "agents_md", "ide_rules"},
			ApprovalChannel:   "prompt",
			PortabilityMode:   PortabilityModeBestEffort,
		},
		DegradePolicy: DegradePolicyProposalOnly,
	})
	decision := out.Decision.(PortabilityDecision)
	if decision.PortabilityStatus != PortabilityDegraded {
		t.Fatalf("status=%q", decision.PortabilityStatus)
	}
	if len(decision.DegradedTargets) != 1 || decision.DegradedTargets[0] != "agents_md" {
		t.Fatalf("degraded=%v", decision.DegradedTargets)
	}
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}

func TestResolveSurfacePortabilityUnsupported(t *testing.T) {
	out := ResolveSurfacePortability(Input{
		InterfaceID:      "mcp",
		RequestedTargets: []string{"repo_docs"},
		AvailableSources: []string{"canonical_policy", "repo_docs"},
		BindingCapabilities: BindingCapabilities{
			InterfaceID:     "mcp",
			ApprovalChannel: "unsupported",
			PortabilityMode: PortabilityModeUnsupported,
		},
		DegradePolicy: DegradePolicyProposalOnly,
	})
	decision := out.Decision.(PortabilityDecision)
	if decision.PortabilityStatus != PortabilityUnsupported {
		t.Fatalf("status=%q", decision.PortabilityStatus)
	}
	if len(decision.UnsupportedTargets) != 1 || decision.UnsupportedTargets[0] != "repo_docs" {
		t.Fatalf("unsupported=%v", decision.UnsupportedTargets)
	}
	if out.NextAllowedAction != "inspect" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}
