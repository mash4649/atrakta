package progress

import "testing"

func TestLoadOrInitRoundtrip(t *testing.T) {
	repo := t.TempDir()
	p, existed, err := LoadOrInit(repo)
	if err != nil {
		t.Fatalf("load/init failed: %v", err)
	}
	if existed {
		t.Fatalf("expected not existed")
	}
	if p.CompletedFeatures == nil {
		t.Fatalf("expected completed features initialized")
	}

	feature := "feat-1"
	p.ActiveFeature = &feature
	if err := Save(repo, p); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	p2, existed2, err := LoadOrInit(repo)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if !existed2 {
		t.Fatalf("expected existed on reload")
	}
	if p2.ActiveFeature == nil || *p2.ActiveFeature != feature {
		t.Fatalf("unexpected active feature: %#v", p2.ActiveFeature)
	}
}
