package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGitDirSupportsFilePointer(t *testing.T) {
	repo := t.TempDir()
	gitRoot := filepath.Join(repo, ".git-worktree")
	if err := os.MkdirAll(gitRoot, 0o755); err != nil {
		t.Fatalf("mkdir git root: %v", err)
	}
	gitMeta := filepath.Join(repo, ".git")
	if err := os.WriteFile(gitMeta, []byte("gitdir: .git-worktree\n"), 0o644); err != nil {
		t.Fatalf("write .git pointer: %v", err)
	}
	got, ok := resolveGitDir(gitMeta, repo)
	if !ok {
		t.Fatalf("expected gitdir resolution")
	}
	if got != gitRoot {
		t.Fatalf("unexpected gitdir: got=%s want=%s", got, gitRoot)
	}
}
