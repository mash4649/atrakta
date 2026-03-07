package runtimeobs

import (
	"testing"
	"time"
)

func TestRecordComputesPercentiles(t *testing.T) {
	repo := t.TempDir()
	for i := int64(1); i <= 20; i++ {
		if _, err := Record(repo, "start", time.Duration(i)*time.Millisecond); err != nil {
			t.Fatalf("record failed: %v", err)
		}
	}
	snap, err := Record(repo, "start", 25*time.Millisecond)
	if err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if snap.Command != "start" || snap.LastMs != 25 {
		t.Fatalf("unexpected snapshot: %#v", snap)
	}
	if snap.Count != 21 {
		t.Fatalf("unexpected sample count: %d", snap.Count)
	}
	if snap.P95Ms <= 0 || snap.P50Ms <= 0 || snap.P95Ms < snap.P50Ms {
		t.Fatalf("unexpected percentiles: p50=%d p95=%d", snap.P50Ms, snap.P95Ms)
	}
}
