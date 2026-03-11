package brownfield

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"atrakta/internal/bootstrap"
	"atrakta/internal/projection"
)

func TestBrownfieldFixtureExistingAgentsNoOverwriteForCodexSelfSource(t *testing.T) {
	repo := loadBrownfieldFixtureRepo(t, "existing-agents")
	d, err := Detect(repo)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !d.AGENTS {
		t.Fatalf("expected AGENTS detection")
	}
	conflicts, err := FindConflicts(repo, []projection.Desired{{Path: "AGENTS.md", Interface: "codex_cli"}}, true)
	if err != nil {
		t.Fatalf("find conflicts failed: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflict for codex self-source AGENTS, got %#v", conflicts)
	}
}

func TestBrownfieldFixtureExistingClaudeDetectsConflict(t *testing.T) {
	repo := loadBrownfieldFixtureRepo(t, "existing-claude")
	d, err := Detect(repo)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !d.CLAUDE {
		t.Fatalf("expected CLAUDE detection")
	}
	conflicts, err := FindConflicts(repo, []projection.Desired{{Path: "CLAUDE.md", Interface: "claude_code"}}, true)
	if err != nil {
		t.Fatalf("find conflicts failed: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected single claude conflict, got %#v", conflicts)
	}
}

func TestBrownfieldFixtureExistingCursorRulesDetectsConflict(t *testing.T) {
	repo := loadBrownfieldFixtureRepo(t, "existing-cursor-rules")
	fixtureRule := filepath.Join(repo, "cursor-rules", "00-atrakta.mdc")
	ruleBody, err := os.ReadFile(fixtureRule)
	if err != nil {
		t.Fatalf("read fixture cursor rule failed: %v", err)
	}
	actualRule := filepath.Join(repo, ".cursor", "rules", "00-atrakta.mdc")
	if err := os.MkdirAll(filepath.Dir(actualRule), 0o755); err != nil {
		t.Fatalf("mkdir cursor rules failed: %v", err)
	}
	if err := os.WriteFile(actualRule, ruleBody, 0o644); err != nil {
		t.Fatalf("write cursor rule failed: %v", err)
	}

	d, err := Detect(repo)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	if !d.CursorRules {
		t.Fatalf("expected cursor rules detection")
	}
	conflicts, err := FindConflicts(repo, []projection.Desired{{Path: ".cursor/rules/00-atrakta.mdc", Interface: "cursor"}}, true)
	if err != nil {
		t.Fatalf("find conflicts failed: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected single cursor rule conflict, got %#v", conflicts)
	}
}

func TestBrownfieldFixtureShellRCWithUserContentIsDetected(t *testing.T) {
	repo := loadBrownfieldFixtureRepo(t, "shell-rc-with-user-content")
	home := filepath.Join(repo, "home")
	t.Setenv("HOME", home)

	d, err := Detect(repo)
	if err != nil {
		t.Fatalf("detect failed: %v", err)
	}
	expected := filepath.Join(home, ".zshrc")
	if !slices.Contains(d.ShellRC, expected) {
		t.Fatalf("expected shell rc %s in detection list: %#v", expected, d.ShellRC)
	}
}

func TestBrownfieldFixtureConflictingManagedBlockRepairIsIdempotent(t *testing.T) {
	repo := loadBrownfieldFixtureRepo(t, "conflicting-managed-block")
	first, created, err := bootstrap.EnsureRootAGENTSWithMode(repo, "append", "")
	if err != nil {
		t.Fatalf("first repair failed: %v", err)
	}
	if created {
		t.Fatalf("expected existing AGENTS fixture, got created=true")
	}
	firstStart := strings.Count(first, "<!-- ATRAKTA_MANAGED:START -->")
	firstEnd := strings.Count(first, "<!-- ATRAKTA_MANAGED:END -->")
	if firstStart < 1 || firstEnd < 1 {
		t.Fatalf("expected at least one completed managed block after repair, got:\n%s", first)
	}

	second, _, err := bootstrap.EnsureRootAGENTSWithMode(repo, "append", "")
	if err != nil {
		t.Fatalf("second repair failed: %v", err)
	}
	secondStart := strings.Count(second, "<!-- ATRAKTA_MANAGED:START -->")
	secondEnd := strings.Count(second, "<!-- ATRAKTA_MANAGED:END -->")
	if secondStart != 1 || secondEnd != 1 {
		t.Fatalf("expected normalized single managed block on rerun, got start=%d end=%d", secondStart, secondEnd)
	}
	if secondStart > firstStart || secondEnd > firstEnd {
		t.Fatalf("managed block markers unexpectedly increased: first=(%d,%d) second=(%d,%d)", firstStart, firstEnd, secondStart, secondEnd)
	}
}

func loadBrownfieldFixtureRepo(t *testing.T, fixtureName string) string {
	t.Helper()
	src := filepath.Join(repoRootFromBrownfieldCaller(t), "testdata", "brownfield", fixtureName)
	dst := t.TempDir()
	copyBrownfieldTree(t, src, dst)
	return dst
}

func repoRootFromBrownfieldCaller(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyBrownfieldTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode())
	}); err != nil {
		t.Fatalf("copy fixture failed: %v", err)
	}
}
