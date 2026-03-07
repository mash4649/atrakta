package state

import (
	"fmt"
	"sync"
	"testing"
)

func TestSaveConcurrentRemainsReadable(t *testing.T) {
	repo := t.TempDir()
	base := Empty("sha256:contract")
	const n = 24
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			s := base
			s.ManagedPaths = map[string]ManagedRecord{
				fmt.Sprintf(".cursor/AGENTS-%d.md", i): {
					Interface:   "cursor",
					Kind:        "copy",
					Fingerprint: fmt.Sprintf("sha256:fp-%d", i),
					TemplateID:  "cursor:agents-md@1",
				},
			}
			if err := Save(repo, s); err != nil {
				t.Errorf("save failed: %v", err)
			}
		}()
	}
	wg.Wait()

	got, _, err := LoadOrEmpty(repo, "sha256:contract")
	if err != nil {
		t.Fatalf("load after concurrent save failed: %v", err)
	}
	if got.V != 1 {
		t.Fatalf("unexpected state version: %d", got.V)
	}
}

func TestUpdateFromApplyTracksAdoptAsManaged(t *testing.T) {
	in := Empty("sha256:old")
	out := UpdateFromApply(in, "sha256:new", ApplyResult{
		Ops: []ApplyOpResult{{
			Path:        ".cursor/AGENTS.md",
			Op:          "adopt",
			Status:      "skipped",
			Interface:   "cursor",
			TemplateID:  "cursor:agents-md@1",
			Fingerprint: "sha256:fp",
			Kind:        "link",
			Target:      "AGENTS.md",
		}},
	})
	rec, ok := out.ManagedPaths[".cursor/AGENTS.md"]
	if !ok {
		t.Fatalf("expected managed path to be recorded")
	}
	if rec.Kind != "link" || rec.TemplateID != "cursor:agents-md@1" || rec.Fingerprint != "sha256:fp" {
		t.Fatalf("unexpected record: %#v", rec)
	}
}
