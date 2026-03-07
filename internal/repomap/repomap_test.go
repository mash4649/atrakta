package repomap

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrRefreshCachesWithinWindow(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "x\n")

	cfg := Config{MaxTokens: 100, RefreshSeconds: 3600, Includes: []string{""}}
	first, err := LoadOrRefresh(repo, cfg)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if !first.Refreshed || first.Cached {
		t.Fatalf("expected refreshed result")
	}
	second, err := LoadOrRefresh(repo, cfg)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if !second.Cached || second.Refreshed {
		t.Fatalf("expected cached result")
	}
}

func TestLoadOrRefreshHonorsBudget(t *testing.T) {
	repo := t.TempDir()
	for i := 0; i < 50; i++ {
		mustWrite(t, filepath.Join(repo, "dir", fmt.Sprintf("file-%03d.txt", i)), "data\n")
	}
	got, err := LoadOrRefresh(repo, Config{MaxTokens: 10, RefreshSeconds: 1, Includes: []string{""}})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if got.UsedTokens > 10 {
		t.Fatalf("budget exceeded: %d", got.UsedTokens)
	}
}

func mustWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
