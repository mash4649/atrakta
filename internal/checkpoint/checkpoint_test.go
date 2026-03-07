package checkpoint

import "testing"

func TestSaveLoadLatestRoundTrip(t *testing.T) {
	repo := t.TempDir()
	in := RunCheckpoint{
		FeatureID:    "feat-1",
		Interfaces:   "cursor",
		SyncLevel:    "1",
		Stage:        "plan_built",
		Outcome:      "running",
		DetectReason: "explicit",
		PlanID:       "plan-1",
		TaskGraphID:  "graph-1",
	}
	if err := SaveLatest(repo, in); err != nil {
		t.Fatalf("save checkpoint failed: %v", err)
	}
	out, err := LoadLatest(repo)
	if err != nil {
		t.Fatalf("load checkpoint failed: %v", err)
	}
	if out.V != 1 || out.Stage != in.Stage || out.FeatureID != in.FeatureID {
		t.Fatalf("unexpected checkpoint: %#v", out)
	}
	if out.UpdatedAt == "" {
		t.Fatalf("expected updated_at")
	}
}
