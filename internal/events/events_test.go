package events

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestA7CorruptedChainBlocked(t *testing.T) {
	repo := t.TempDir()
	if _, err := Append(repo, "detect", "kernel", map[string]any{"target_set": []string{"cursor"}, "prune_allowed": false, "reason": "unknown", "signals": map[string]any{}}); err != nil {
		t.Fatal(err)
	}
	ep := filepath.Join(repo, ".atrakta", "events.jsonl")
	b, err := os.ReadFile(ep)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 10 {
		t.Fatalf("unexpected short events file")
	}
	b[len(b)-2] = 'x'
	if err := os.WriteFile(ep, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChain(repo); err == nil {
		t.Fatalf("expected hash-chain verification failure")
	}
}

func TestAppendConcurrentChainSafe(t *testing.T) {
	repo := t.TempDir()
	const n = 48
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			if _, err := Append(repo, "detect", "kernel", map[string]any{
				"target_set":    []string{"cursor"},
				"prune_allowed": false,
				"reason":        "explicit",
				"signals":       map[string]any{"idx": i},
			}); err != nil {
				t.Errorf("append failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if err := VerifyChain(repo); err != nil {
		t.Fatalf("expected valid chain after concurrent append: %v", err)
	}
	ev, err := ReadAll(repo)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(ev) != n {
		t.Fatalf("event count mismatch: got=%d want=%d", len(ev), n)
	}
	sv, ok := ev[0].Raw["schema_version"].(float64)
	if !ok || int(sv) != SchemaVersion {
		t.Fatalf("unexpected schema_version: %#v", ev[0].Raw["schema_version"])
	}
}

func TestAppendBatchProducesValidChain(t *testing.T) {
	repo := t.TempDir()
	_, err := AppendBatch(repo, []AppendInput{
		{Type: "detect", Actor: "kernel", Payload: map[string]any{"reason": "explicit"}},
		{Type: "plan", Actor: "kernel", Payload: map[string]any{"id": "p1"}},
		{Type: "apply", Actor: "worker", Payload: map[string]any{"result": "success"}},
	})
	if err != nil {
		t.Fatalf("append batch failed: %v", err)
	}
	if err := VerifyChain(repo); err != nil {
		t.Fatalf("expected valid chain: %v", err)
	}
}

func TestVerifyChainCached(t *testing.T) {
	repo := t.TempDir()
	if _, err := Append(repo, "detect", "kernel", map[string]any{"reason": "explicit"}); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if err := VerifyChainCached(repo); err != nil {
		t.Fatalf("first verify cached failed: %v", err)
	}
	if err := VerifyChainCached(repo); err != nil {
		t.Fatalf("second verify cached failed: %v", err)
	}
}
