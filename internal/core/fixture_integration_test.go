package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/core"
	"atrakta/internal/doctor"
	"atrakta/internal/model"
)

func TestParityFixtureMinimalDeterministic(t *testing.T) {
	repo := loadFixtureRepo(t, "parity/minimal")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	firstHash := mustManifestHashForFile(t, repo, ".cursor/AGENTS.md")
	firstContent := mustRead(t, filepath.Join(repo, ".cursor", "AGENTS.md"))

	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	secondHash := mustManifestHashForFile(t, repo, ".cursor/AGENTS.md")
	secondContent := mustRead(t, filepath.Join(repo, ".cursor", "AGENTS.md"))

	if firstHash != secondHash {
		t.Fatalf("manifest hash changed across deterministic rerun: first=%s second=%s", firstHash, secondHash)
	}
	if firstContent != secondContent {
		t.Fatalf("projected content changed across deterministic rerun")
	}
}

func TestParityFixtureMultiInterfaceProjection(t *testing.T) {
	repo := loadFixtureRepo(t, "parity/multi-interface")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor,claude_code,codex_cli"}); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	mustExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
	mustExist(t, filepath.Join(repo, ".cursor", "rules", "00-atrakta.mdc"))
	mustExist(t, filepath.Join(repo, "CLAUDE.md"))
	mustExist(t, filepath.Join(repo, ".claude", "settings.json"))
	mustExist(t, filepath.Join(repo, ".codex", "config.toml"))
}

func TestParityFixtureDriftAndRepair(t *testing.T) {
	repo := loadFixtureRepo(t, "parity/drifted")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("initial start failed: %v", err)
	}
	p := filepath.Join(repo, ".cursor", "AGENTS.md")
	if err := os.Remove(p); err != nil {
		t.Fatalf("remove projection symlink failed: %v", err)
	}
	if err := os.WriteFile(p, []byte("manual drift\n"), 0o644); err != nil {
		t.Fatalf("replace projection with drifted file failed: %v", err)
	}

	rep, err := doctor.RunParity(repo)
	if err != nil {
		t.Fatalf("run parity failed: %v", err)
	}
	if !hasAnyParityFinding(rep.BlockingIssues, "managed_block_corruption", "render_hash_mismatch") {
		t.Fatalf("expected drift-related blocking finding, got %#v", rep.BlockingIssues)
	}

	if err := os.Remove(p); err != nil {
		t.Fatalf("remove corrupted projection failed: %v", err)
	}
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("repair start failed: %v", err)
	}
	rep2, err := doctor.RunParity(repo)
	if err != nil {
		t.Fatalf("run parity after repair failed: %v", err)
	}
	if len(rep2.BlockingIssues) != 0 {
		t.Fatalf("expected no blocking issues after repair, got %#v", rep2.BlockingIssues)
	}
}

func TestParityFixtureUnsupportedFieldFailsClosed(t *testing.T) {
	repo := loadFixtureRepo(t, "parity/unsupported-field")
	_, _, err := contract.LoadOrInit(repo)
	if err == nil {
		t.Fatalf("expected unsupported field contract to fail closed")
	}
	if !strings.Contains(err.Error(), "unsupported optional template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParityFixtureNoninteractiveMismatchWarning(t *testing.T) {
	repo := loadFixtureRepo(t, "parity/noninteractive")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	t.Setenv("ATRAKTA_NONINTERACTIVE", "1")
	rep, err := doctor.RunParity(repo)
	if err != nil {
		t.Fatalf("run parity failed: %v", err)
	}
	if !hasAnyParityFinding(rep.Warnings, "noninteractive_mismatch") {
		t.Fatalf("expected noninteractive_mismatch warning, got %#v", rep.Warnings)
	}
}

func hasAnyParityFinding(list []doctor.ParityFinding, codes ...string) bool {
	for _, f := range list {
		for _, code := range codes {
			if f.Code == code {
				return true
			}
		}
	}
	return false
}

func mustManifestHashForFile(t *testing.T, repo, relPath string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, ".atrakta", "projections", "manifest.json"))
	if err != nil {
		t.Fatalf("read projection manifest failed: %v", err)
	}
	var m model.ProjectionManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse projection manifest failed: %v", err)
	}
	for _, e := range m.Entries {
		for _, f := range e.Files {
			if f == relPath {
				if e.RenderHash == "" {
					t.Fatalf("empty render hash for %s", relPath)
				}
				return e.RenderHash
			}
		}
	}
	t.Fatalf("manifest entry not found for %s", relPath)
	return ""
}

func loadFixtureRepo(t *testing.T, fixtureRel string) string {
	t.Helper()
	src := filepath.Join(repoRootFromCaller(t), "testdata", filepath.FromSlash(fixtureRel))
	dst := t.TempDir()
	copyTree(t, src, dst)
	return dst
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyTree(t *testing.T, src, dst string) {
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

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s failed: %v", p, err)
	}
	return string(b)
}
