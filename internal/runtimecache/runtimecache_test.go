package runtimecache

import (
	"testing"
	"time"
)

func TestUpdateAndLoad(t *testing.T) {
	repo := t.TempDir()
	if err := Update(repo, func(st *State) error {
		st.Entries["k"] = Entry{
			UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
			Stamp:      "s1",
			ConfigHash: "c1",
			TTLSeconds: 60,
			Payload:    MarshalPayload(map[string]any{"x": 1}),
		}
		return nil
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	st, err := Load(repo)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if st.V != version {
		t.Fatalf("unexpected version: %d", st.V)
	}
	e, ok := st.Entries["k"]
	if !ok {
		t.Fatalf("missing key")
	}
	if !IsFresh(e, "s1", "c1", time.Now().UTC()) {
		t.Fatalf("expected fresh entry")
	}
}
