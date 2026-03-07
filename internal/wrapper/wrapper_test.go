package wrapper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"atrakta/internal/contract"
)

func TestA3WrapperFastPathSkipsSecondSync(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script wrapper mock is unix-only")
	}
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ATRAKTA_WRAP_DISABLE", "")
	t.Setenv("ATRAKTA_WRAP_ACTIVE", "")

	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "# test\n")
	writeDefaultContract(t, repo)

	logPath := filepath.Join(t.TempDir(), "self.log")
	selfExe := writeScript(t, filepath.Join(t.TempDir(), "self.sh"), "#!/bin/sh\necho called >> \""+logPath+"\"\nexit 0\n")
	realExe := writeScript(t, filepath.Join(t.TempDir(), "real.sh"), "#!/bin/sh\nexit 0\n")

	restore := chdir(t, repo)
	defer restore()

	if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
		t.Fatalf("first wrapper run exit=%d", code)
	}
	if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
		t.Fatalf("second wrapper run exit=%d", code)
	}

	calls := countLines(t, logPath)
	if calls != 1 {
		t.Fatalf("expected sync path once, got %d calls", calls)
	}
}

func TestA4WrapperDisableBypassesAtrakta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script wrapper mock is unix-only")
	}
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ATRAKTA_WRAP_DISABLE", "1")
	t.Setenv("ATRAKTA_WRAP_ACTIVE", "")

	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "# test\n")
	writeDefaultContract(t, repo)

	logPath := filepath.Join(t.TempDir(), "self.log")
	selfExe := writeScript(t, filepath.Join(t.TempDir(), "self.sh"), "#!/bin/sh\necho called >> \""+logPath+"\"\nexit 0\n")
	realExe := writeScript(t, filepath.Join(t.TempDir(), "real.sh"), "#!/bin/sh\nexit 0\n")

	restore := chdir(t, repo)
	defer restore()

	if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
		t.Fatalf("wrapper run exit=%d", code)
	}

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected self executable not to be invoked when disabled")
	}
}

func TestSLOWrapperFastPathHitRate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script wrapper mock is unix-only")
	}
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ATRAKTA_WRAP_DISABLE", "")
	t.Setenv("ATRAKTA_WRAP_ACTIVE", "")

	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "# test\n")
	writeDefaultContract(t, repo)

	logPath := filepath.Join(t.TempDir(), "self.log")
	selfExe := writeScript(t, filepath.Join(t.TempDir(), "self.sh"), "#!/bin/sh\necho called >> \""+logPath+"\"\nexit 0\n")
	realExe := writeScript(t, filepath.Join(t.TempDir(), "real.sh"), "#!/bin/sh\nexit 0\n")

	restore := chdir(t, repo)
	defer restore()

	if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
		t.Fatalf("warmup wrapper run exit=%d", code)
	}
	totalRuns := 40
	for i := 0; i < totalRuns; i++ {
		if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
			t.Fatalf("wrapper run %d exit=%d", i, code)
		}
	}
	calls := countLines(t, logPath)
	// first warmup must call start once; remaining calls should stay on fast-path.
	slowRuns := calls - 1
	if slowRuns < 0 {
		slowRuns = 0
	}
	hitRate := float64(totalRuns-slowRuns) / float64(totalRuns)
	if hitRate < 0.95 {
		t.Fatalf("wrapper fast-path hit rate too low: %.3f", hitRate)
	}
}

func writeDefaultContract(t *testing.T, repo string) {
	t.Helper()
	c := contract.Default(repo)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta"), 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "contract.json"), append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
}

func writeScript(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
	return path
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() {
		_ = os.Chdir(old)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestEnsurePathSnippetIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rc := filepath.Join(home, ".zshrc")
	binDir := filepath.Join(home, ".local", "bin")

	changed, err := ensurePathSnippet(rc, binDir)
	if err != nil {
		t.Fatalf("ensurePathSnippet first: %v", err)
	}
	if !changed {
		t.Fatalf("expected first ensurePathSnippet to change file")
	}
	b, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read rc: %v", err)
	}
	text := string(b)
	if !strings.Contains(text, "# >>> atrakta path >>>") || !strings.Contains(text, "$HOME/.local/bin") {
		t.Fatalf("missing atrakta path snippet: %q", text)
	}

	changed, err = ensurePathSnippet(rc, binDir)
	if err != nil {
		t.Fatalf("ensurePathSnippet second: %v", err)
	}
	if changed {
		t.Fatalf("expected second ensurePathSnippet to be no-op")
	}
}
