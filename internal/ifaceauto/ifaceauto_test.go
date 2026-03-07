package ifaceauto

import (
	"testing"
	"time"
)

func TestLoadMissingReturnsEmpty(t *testing.T) {
	repo := t.TempDir()
	st, err := Load(repo)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if st.V != 1 {
		t.Fatalf("expected v=1, got %d", st.V)
	}
	if len(st.LastTargetSet) != 0 {
		t.Fatalf("expected empty targets")
	}
}

func TestRecordSaveLoadAndResolve(t *testing.T) {
	repo := t.TempDir()
	st := Empty()
	st = Record(st, []string{"cursor"}, "trigger")
	if err := Save(repo, st); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(repo)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, ok := LastSingleTarget(loaded, map[string]struct{}{"cursor": {}, "trae": {}})
	if !ok || got != "cursor" {
		t.Fatalf("unexpected last single target: %q ok=%v", got, ok)
	}
}

func TestSuggestStale(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour).Format(time.RFC3339Nano)
	fresh := now.Add(-2 * time.Hour).Format(time.RFC3339Nano)
	st := State{
		V:             1,
		LastTargetSet: []string{"cursor"},
		Usage: map[string]UsageStat{
			"cursor": {Count: 3, LastUsedAt: fresh},
			"trae":   {Count: 1, LastUsedAt: old},
		},
	}
	candidates := SuggestStale(st, []string{"cursor"}, 24*time.Hour, now)
	if len(candidates) != 1 || candidates[0] != "trae" {
		t.Fatalf("unexpected stale suggestions: %#v", candidates)
	}
}
