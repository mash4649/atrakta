package mutation

import (
	"os"
	"path/filepath"
	"testing"

	checkmutationscope "github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
)

func TestInspectAndPropose(t *testing.T) {
	target := checkmutationscope.Target{
		Path:      ".atrakta/generated/repo-map.generated.json",
		AssetType: "repo_map",
	}
	env := Inspect(target)
	if env.Scope == "" {
		t.Fatalf("scope required")
	}
	proposal := Propose(target, `{"ok":true}`)
	if !proposal.Envelope.Allowed {
		t.Fatalf("proposal should be allowed")
	}
	if proposal.ProposedPatch == "" {
		t.Fatalf("proposal patch required")
	}
}

func TestApplyManagedTarget(t *testing.T) {
	root := t.TempDir()
	target := checkmutationscope.Target{
		Path:      ".atrakta/generated/repo-map.generated.json",
		AssetType: "repo_map",
	}
	env, err := Apply(root, target, `{"ok":true}`+"\n", true)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !env.Allowed {
		t.Fatalf("expected apply allowed")
	}
	if _, err := os.Stat(filepath.Join(root, ".atrakta/generated/repo-map.generated.json")); err != nil {
		t.Fatalf("applied file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".atrakta/audit/events/install-events.jsonl")); err != nil {
		t.Fatalf("audit log missing: %v", err)
	}
}

func TestApplyUnmanagedRejected(t *testing.T) {
	root := t.TempDir()
	target := checkmutationscope.Target{
		Path:      "src/main.go",
		AssetType: "existing_user_rules",
	}
	if _, err := Apply(root, target, "package main\n", true); err == nil {
		t.Fatalf("expected unmanaged apply rejection")
	}
}
