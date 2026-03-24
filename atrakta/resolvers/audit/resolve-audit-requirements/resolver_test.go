package resolveauditrequirements

import "testing"

func TestIntegrityLevels(t *testing.T) {
	inspect := ResolveAuditRequirements(Input{Action: "inspect"}).Decision.(AuditDecision)
	if inspect.RequiredIntegrityLevel != "A0" {
		t.Fatalf("inspect level=%s", inspect.RequiredIntegrityLevel)
	}
	propose := ResolveAuditRequirements(Input{Action: "propose"}).Decision.(AuditDecision)
	if propose.RequiredIntegrityLevel != "A2" {
		t.Fatalf("propose level=%s", propose.RequiredIntegrityLevel)
	}
	release := ResolveAuditRequirements(Input{Action: "release"}).Decision.(AuditDecision)
	if release.RequiredIntegrityLevel != "A3" {
		t.Fatalf("release level=%s", release.RequiredIntegrityLevel)
	}
}

func TestArchiveDryRunFirst(t *testing.T) {
	out := ResolveAuditRequirements(Input{Action: "archive"})
	d := out.Decision.(AuditDecision)
	if !d.DryRunRequired {
		t.Fatalf("archive must require dry-run")
	}
}

func TestDestructiveCleanupProposalOnly(t *testing.T) {
	out := ResolveAuditRequirements(Input{Action: "apply", DestructiveCleanup: true})
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
	d := out.Decision.(AuditDecision)
	if d.DestructiveCleanupMode != "proposal_only" {
		t.Fatalf("cleanup mode=%q", d.DestructiveCleanupMode)
	}
}
