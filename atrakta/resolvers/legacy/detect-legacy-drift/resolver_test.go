package detectlegacydrift

import "testing"

func TestNoDrift(t *testing.T) {
	out := DetectLegacyDrift(Input{Signals: []string{"unknown_signal"}})
	d := out.Decision.(DriftDecision)
	if len(d.Detected) != 0 {
		t.Fatalf("detected should be empty")
	}
	if out.NextAllowedAction != "inspect" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}

func TestWarnDrift(t *testing.T) {
	out := DetectLegacyDrift(Input{Signals: []string{"stale_review"}})
	d := out.Decision.(DriftDecision)
	if d.Severity != "warn" {
		t.Fatalf("severity=%q", d.Severity)
	}
	if out.NextAllowedAction != "propose" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}

func TestStrictDrift(t *testing.T) {
	out := DetectLegacyDrift(Input{Signals: []string{"canonical_conflict"}})
	d := out.Decision.(DriftDecision)
	if d.Severity != "strict" {
		t.Fatalf("severity=%q", d.Severity)
	}
	if out.NextAllowedAction != "deny" {
		t.Fatalf("next=%q", out.NextAllowedAction)
	}
}
