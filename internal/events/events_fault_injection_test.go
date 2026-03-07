package events

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFaultInjectionInterruptedAppendIsDetected(t *testing.T) {
	repo := t.TempDir()
	if _, err := Append(repo, "detect", "kernel", map[string]any{"reason": "explicit"}); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	ep := filepath.Join(repo, ".atrakta", "events.jsonl")
	f, err := os.OpenFile(ep, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("open events file failed: %v", err)
	}
	if _, err := f.Write([]byte("{\"v\":1")); err != nil {
		_ = f.Close()
		t.Fatalf("write interrupted payload failed: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close events file failed: %v", err)
	}
	if err := VerifyChain(repo); err == nil {
		t.Fatalf("expected chain verification to fail for interrupted append")
	}
}

func TestFaultInjectionGroupCommitRequiresFlush(t *testing.T) {
	repo := t.TempDir()
	t.Cleanup(func() {
		clearDirty(repo)
		commitMu.Lock()
		delete(lastSync, repo)
		commitMu.Unlock()
	})

	commitMu.Lock()
	lastSync[repo] = time.Now().UTC()
	commitMu.Unlock()

	_, err := AppendBatch(repo, []AppendInput{
		{Type: "plan", Actor: "kernel", Payload: map[string]any{"id": "p1"}},
		{Type: "apply", Actor: "worker", Payload: map[string]any{"result": "success"}},
	})
	if err != nil {
		t.Fatalf("append batch failed: %v", err)
	}
	if !isDirty(repo) {
		t.Fatalf("expected dirty events state before flush")
	}
	if err := Flush(repo); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if isDirty(repo) {
		t.Fatalf("expected dirty events state to be cleared by flush")
	}
}

func TestFaultInjectionBlockedOutcomeForcesImmediateSync(t *testing.T) {
	repo := t.TempDir()
	t.Cleanup(func() {
		clearDirty(repo)
		commitMu.Lock()
		delete(lastSync, repo)
		commitMu.Unlock()
	})

	commitMu.Lock()
	lastSync[repo] = time.Now().UTC()
	commitMu.Unlock()

	_, err := AppendBatch(repo, []AppendInput{
		{Type: "step", Actor: "worker", Payload: map[string]any{"outcome": "BLOCKED"}},
		{Type: "intent", Actor: "doctor", Payload: map[string]any{"text": "repair required"}},
	})
	if err != nil {
		t.Fatalf("append batch failed: %v", err)
	}
	if isDirty(repo) {
		t.Fatalf("expected blocked outcome batch to sync immediately")
	}
	if _, ok := getVerifyCache(repo); !ok {
		t.Fatalf("expected verify cache to be updated on urgent sync")
	}
}
