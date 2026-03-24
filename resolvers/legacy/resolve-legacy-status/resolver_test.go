package resolvelegacystatus

import "testing"

func TestCanonicalizedWhenAllPromotionConditionsMet(t *testing.T) {
	out := ResolveLegacyStatus(Asset{AssetID: "a1", Ownership: "known", Freshness: "acceptable", CanonicalMapping: "exists", Integrity: "known"})
	d := out.Decision.(LegacyDecision)
	if d.Status != StatusCanonicalized {
		t.Fatalf("status=%q want=%q", d.Status, StatusCanonicalized)
	}
	if !d.CanAutoPromote {
		t.Fatalf("canAutoPromote should be true")
	}
}

func TestPartiallyMappedWhenMappingExistsButNotPromotable(t *testing.T) {
	out := ResolveLegacyStatus(Asset{AssetID: "a2", Ownership: "known", Freshness: "stale", CanonicalMapping: "exists"})
	d := out.Decision.(LegacyDecision)
	if d.Status != StatusPartiallyMapped {
		t.Fatalf("status=%q want=%q", d.Status, StatusPartiallyMapped)
	}
	if d.CanAutoPromote {
		t.Fatalf("canAutoPromote should be false")
	}
}

func TestReferenceOnlyWithoutMapping(t *testing.T) {
	out := ResolveLegacyStatus(Asset{AssetID: "a3", Ownership: "unknown", Freshness: "unknown", CanonicalMapping: "missing"})
	d := out.Decision.(LegacyDecision)
	if d.Status != StatusReferenceOnly {
		t.Fatalf("status=%q want=%q", d.Status, StatusReferenceOnly)
	}
	if d.CanAutoPromote {
		t.Fatalf("canAutoPromote should be false")
	}
}

func TestCanonicalConflictEscalatesStrict(t *testing.T) {
	out := ResolveLegacyStatus(Asset{AssetID: "a4", Ownership: "known", Freshness: "acceptable", CanonicalMapping: "exists", CanonicalConflict: true})
	d := out.Decision.(LegacyDecision)
	if d.Routing != "strict_escalate" {
		t.Fatalf("routing=%q want=strict_escalate", d.Routing)
	}
}

func TestUnknownMetadataCannotAutoPromote(t *testing.T) {
	out := ResolveLegacyStatus(Asset{AssetID: "a5", Ownership: "known", Freshness: "acceptable", CanonicalMapping: "exists", Integrity: "unknown"})
	d := out.Decision.(LegacyDecision)
	if d.CanAutoPromote {
		t.Fatalf("unknown metadata must not auto-promote")
	}
}
