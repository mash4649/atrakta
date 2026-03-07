package context

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"atrakta/internal/contract"
)

func TestResolveNearestWithImport(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "root\nimport: docs/common.md\n")
	mustWrite(t, filepath.Join(repo, "pkg", "AGENTS.md"), "pkg\nimport: ../shared.md\n")
	mustWrite(t, filepath.Join(repo, "shared.md"), "shared\n")
	mustWrite(t, filepath.Join(repo, "docs", "common.md"), "common\n")

	text, report, err := Resolve(ResolveInput{
		RepoRoot: repo,
		StartDir: filepath.Join(repo, "pkg"),
		Config: &contract.Context{
			Resolution:     "nearest_with_import",
			MaxImportDepth: 6,
		},
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if report.Root != "pkg/AGENTS.md" {
		t.Fatalf("unexpected root: %s", report.Root)
	}
	wantResolved := []string{"pkg/AGENTS.md", "shared.md", "AGENTS.md", "docs/common.md"}
	if !reflect.DeepEqual(report.Resolved, wantResolved) {
		t.Fatalf("unexpected resolved order: got=%v want=%v", report.Resolved, wantResolved)
	}
	if report.Depth != 2 {
		t.Fatalf("unexpected depth: %d", report.Depth)
	}
	if text == "" || report.Fingerprint == "" {
		t.Fatalf("expected non-empty resolved text and fingerprint")
	}
}

func TestResolveDetectsImportCycle(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "import: a.md\n")
	mustWrite(t, filepath.Join(repo, "a.md"), "import: b.md\n")
	mustWrite(t, filepath.Join(repo, "b.md"), "import: a.md\n")

	_, _, err := Resolve(ResolveInput{
		RepoRoot: repo,
		Config: &contract.Context{
			Resolution:     "nearest_with_import",
			MaxImportDepth: 6,
		},
	})
	if err == nil {
		t.Fatalf("expected import cycle to fail")
	}
}

func TestResolveDetectsDepthLimit(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "import: a.md\n")
	mustWrite(t, filepath.Join(repo, "a.md"), "import: b.md\n")
	mustWrite(t, filepath.Join(repo, "b.md"), "deep\n")

	_, _, err := Resolve(ResolveInput{
		RepoRoot: repo,
		Config: &contract.Context{
			Resolution:     "nearest_with_import",
			MaxImportDepth: 1,
		},
	})
	if err == nil {
		t.Fatalf("expected depth limit error")
	}
}

func TestResolveLoadsConventions(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "root\n")
	mustWrite(t, filepath.Join(repo, "CONVENTIONS.md"), "no force-push\n")

	text, report, err := Resolve(ResolveInput{
		RepoRoot: repo,
		Config: &contract.Context{
			Resolution:          "nearest_with_import",
			MaxImportDepth:      6,
			Conventions:         []string{"CONVENTIONS.md"},
			ConventionsReadOnly: boolPtr(true),
		},
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if len(report.ConventionsLoaded) != 1 || report.ConventionsLoaded[0] != "CONVENTIONS.md" {
		t.Fatalf("unexpected conventions loaded: %#v", report.ConventionsLoaded)
	}
	if !reflect.DeepEqual(report.Resolved, []string{"AGENTS.md", "CONVENTIONS.md"}) {
		t.Fatalf("unexpected resolved order: %v", report.Resolved)
	}
	if text == "" || report.Fingerprint == "" {
		t.Fatalf("expected context text")
	}
}

func boolPtr(v bool) *bool {
	return &v
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
