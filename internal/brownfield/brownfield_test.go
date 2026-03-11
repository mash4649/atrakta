package brownfield

import (
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/projection"
)

func TestDetectFindsKnownFiles(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "root")
	mustWrite(t, filepath.Join(repo, "CLAUDE.md"), "claude")
	if err := os.MkdirAll(filepath.Join(repo, ".cursor", "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, ".vscode", "tasks.json"), "{}")
	mustWrite(t, filepath.Join(repo, ".codex", "config.toml"), "x=1")

	d, err := Detect(repo)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !d.AGENTS || !d.CLAUDE || !d.CursorRules {
		t.Fatalf("expected AGENTS/CLAUDE/cursor_rules detection, got %#v", d)
	}
	if len(d.Tasks) == 0 {
		t.Fatalf("expected tasks detection")
	}
	if len(d.PluginConfig) == 0 {
		t.Fatalf("expected plugin config detection")
	}
}

func TestFindConflictsNoOverwrite(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "CLAUDE.md"), "user owned")
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "user owned")
	desired := []projection.Desired{
		{Path: "CLAUDE.md", Interface: "claude_code"},
		{Path: "AGENTS.md", Interface: "codex_cli"},
	}
	conflicts, err := FindConflicts(repo, desired, true)
	if err != nil {
		t.Fatalf("find conflicts failed: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict (CLAUDE only), got %#v", conflicts)
	}
	if conflicts[0].Path != "CLAUDE.md" {
		t.Fatalf("unexpected conflict path: %#v", conflicts[0])
	}
}

func TestWriteProposalPatch(t *testing.T) {
	repo := t.TempDir()
	path, err := WriteProposalPatch(repo, ProposalInput{
		Mode:          "brownfield",
		MergeStrategy: "append",
		AgentsMode:    "append",
		NoOverwrite:   true,
		Interfaces:    []string{"cursor"},
		Detection: Detection{
			AGENTS:      true,
			CursorRules: true,
		},
		Conflicts: []Conflict{{Path: ".cursor/AGENTS.md", Interface: "cursor", Reason: "existing user-managed file would be overwritten"}},
	})
	if err != nil {
		t.Fatalf("write proposal failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("proposal patch missing: %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
