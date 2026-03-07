package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
)

func TestEnsureDefaultAndLoadPromptMin(t *testing.T) {
	repo := t.TempDir()
	created, err := EnsureDefaultPromptMin(repo, DefaultPromptMinRef)
	if err != nil {
		t.Fatalf("ensure policy: %v", err)
	}
	if !created {
		t.Fatalf("expected default policy file to be created")
	}
	pol, err := LoadPromptMin(repo, contract.PromptMinRef{Ref: DefaultPromptMinRef})
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if pol.ID != "prompt-min@1" || pol.Apply != "conditional" {
		t.Fatalf("unexpected policy: %#v", pol)
	}
}

func TestPromptMinConditionalExclusionsAndPrefix(t *testing.T) {
	pol := DefaultPromptMin()
	if ShouldApplyPromptMin("json", false, pol) {
		t.Fatalf("json category must be excluded")
	}
	if !ShouldApplyPromptMin("sync", false, pol) {
		t.Fatalf("sync category should be eligible")
	}
	summary, _, applied := ApplyGoalPrefix("plan: 2 ops", "", pol)
	if !applied {
		t.Fatalf("expected goal prefix to apply")
	}
	if summary != "Goal: plan: 2 ops" {
		t.Fatalf("unexpected prefixed summary: %s", summary)
	}
}

func TestLoadPromptMinInvalidApplyFails(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".atrakta", "policies", "prompt-min.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := map[string]any{
		"id":                  "prompt-min@1",
		"apply":               "always",
		"require_goal_prefix": true,
	}
	b, _ := json.Marshal(body)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	_, err := LoadPromptMin(repo, contract.PromptMinRef{Ref: DefaultPromptMinRef})
	if err == nil {
		t.Fatalf("expected invalid apply to fail")
	}
}
