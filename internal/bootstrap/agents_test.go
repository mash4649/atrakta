package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureRootAGENTSWithModeCreatesDefaultWhenMissing(t *testing.T) {
	repo := t.TempDir()
	got, created, err := EnsureRootAGENTSWithMode(repo, "append", "")
	if err != nil {
		t.Fatalf("ensure AGENTS failed: %v", err)
	}
	if !created {
		t.Fatalf("expected created=true for missing AGENTS")
	}
	if got != defaultAGENTS {
		t.Fatalf("unexpected default AGENTS content")
	}
}

func TestEnsureRootAGENTSWithModeAppendIsIdempotent(t *testing.T) {
	repo := t.TempDir()
	p := filepath.Join(repo, "AGENTS.md")
	if err := os.WriteFile(p, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, created, err := EnsureRootAGENTSWithMode(repo, "append", "")
	if err != nil {
		t.Fatalf("append ensure failed: %v", err)
	}
	if created {
		t.Fatalf("expected created=false for existing AGENTS")
	}
	if strings.Count(first, managedBlockStart) != 1 || strings.Count(first, managedBlockEnd) != 1 {
		t.Fatalf("expected one managed block after first run")
	}
	second, _, err := EnsureRootAGENTSWithMode(repo, "append", "")
	if err != nil {
		t.Fatalf("append ensure second run failed: %v", err)
	}
	if first != second {
		t.Fatalf("append mode should be idempotent")
	}
}

func TestEnsureRootAGENTSWithModeIncludeKeepsRootAndWritesAppendFile(t *testing.T) {
	repo := t.TempDir()
	rootPath := filepath.Join(repo, "AGENTS.md")
	root := "# Existing\n"
	if err := os.WriteFile(rootPath, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	got, created, err := EnsureRootAGENTSWithMode(repo, "include", ".atrakta/AGENTS.append.md")
	if err != nil {
		t.Fatalf("include ensure failed: %v", err)
	}
	if created {
		t.Fatalf("expected created=false for existing AGENTS")
	}
	if got != root {
		t.Fatalf("include mode should not rewrite root AGENTS")
	}
	appendPath := filepath.Join(repo, ".atrakta", "AGENTS.append.md")
	b, err := os.ReadFile(appendPath)
	if err != nil {
		t.Fatalf("read append file failed: %v", err)
	}
	if !strings.Contains(string(b), "Atrakta Managed Appendix") {
		t.Fatalf("append file content missing expected heading")
	}
}
