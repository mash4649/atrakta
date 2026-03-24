package audit

import (
	"testing"
)

func TestAppendAndVerifyA0(t *testing.T) {
	root := t.TempDir()
	if _, err := AppendEvent(root, LevelA0, "inspect", map[string]any{"ok": true}); err != nil {
		t.Fatalf("append A0: %v", err)
	}
	if err := VerifyIntegrity(root, LevelA0); err != nil {
		t.Fatalf("verify A0: %v", err)
	}
}

func TestAppendAndVerifyA2(t *testing.T) {
	root := t.TempDir()
	if _, err := AppendEvent(root, LevelA2, "accept_onboarding", map[string]any{"mode": "brownfield_project"}); err != nil {
		t.Fatalf("append A2 #1: %v", err)
	}
	if _, err := AppendEvent(root, LevelA2, "apply_mutation", map[string]any{"target": ".atrakta/generated/x.json"}); err != nil {
		t.Fatalf("append A2 #2: %v", err)
	}
	if err := VerifyIntegrity(root, LevelA2); err != nil {
		t.Fatalf("verify A2: %v", err)
	}
}

func TestAppendAndVerifyA3Checkpoint(t *testing.T) {
	root := t.TempDir()
	if _, err := AppendEvent(root, LevelA3, "checkpoint_seed", map[string]any{"n": 1}); err != nil {
		t.Fatalf("append A3: %v", err)
	}
	if err := VerifyIntegrity(root, LevelA3); err != nil {
		t.Fatalf("verify A3: %v", err)
	}
}
