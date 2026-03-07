package wrapper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"atrakta/internal/contract"
)

func BenchmarkWrapperFastPath(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("shell script wrapper mock is unix-only")
	}

	repo := b.TempDir()
	home := b.TempDir()
	b.Setenv("HOME", home)
	b.Setenv("ATRAKTA_WRAP_DISABLE", "")
	b.Setenv("ATRAKTA_WRAP_ACTIVE", "")
	b.Setenv("ATRAKTA_WRAP_SKIP_LAUNCH", "1")

	mustWriteB(b, filepath.Join(repo, "AGENTS.md"), "# benchmark\n")
	writeDefaultContractB(b, repo)

	// Use stable binaries for benchmark timing. The benchmark targets wrapper fast-path
	// overhead, not shell script process startup variance.
	selfExe := "/usr/bin/true"
	realExe := "/usr/bin/true"

	restore := chdirB(b, repo)
	defer restore()

	if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
		b.Fatalf("warmup run failed: exit=%d", code)
	}

	b.ResetTimer()
	for b.Loop() {
		if code := Run(selfExe, "cursor", realExe, nil); code != 0 {
			b.Fatalf("fast path run failed: exit=%d", code)
		}
	}
}

func writeDefaultContractB(b *testing.B, repo string) {
	b.Helper()
	c := contract.Default(repo)
	cb, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		b.Fatalf("marshal contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta"), 0o755); err != nil {
		b.Fatalf("mkdir .atrakta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "contract.json"), append(cb, '\n'), 0o644); err != nil {
		b.Fatalf("write contract: %v", err)
	}
}

func writeScriptB(b *testing.B, path, content string) string {
	b.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		b.Fatalf("write script %s: %v", path, err)
	}
	return path
}

func mustWriteB(b *testing.B, path, body string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		b.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		b.Fatalf("write %s: %v", path, err)
	}
}

func chdirB(b *testing.B, dir string) func() {
	b.Helper()
	old, err := os.Getwd()
	if err != nil {
		b.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		b.Fatalf("chdir: %v", err)
	}
	return func() {
		_ = os.Chdir(old)
	}
}
