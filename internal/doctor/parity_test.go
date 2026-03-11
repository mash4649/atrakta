package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/state"
)

func TestRunParityDetectsManifestMissing(t *testing.T) {
	repo := t.TempDir()
	rep, _ := RunParity(repo)
	if rep.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s", rep.Outcome)
	}
	if !hasFinding(rep.BlockingIssues, "manifest_missing") {
		t.Fatalf("expected manifest_missing finding, got %#v", rep.BlockingIssues)
	}
}

func TestRunParityDetectsRenderHashMismatchAndProjectionMissing(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta", "projections"), 0o755); err != nil {
		t.Fatalf("mkdir projections failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta", "extensions"), 0o755); err != nil {
		t.Fatalf("mkdir extensions failed: %v", err)
	}
	pm := model.ProjectionManifest{
		V: 1,
		Entries: []model.ProjectionManifestEntry{{
			Interface:  "cursor",
			Kind:       "link",
			Files:      []string{".cursor/AGENTS.md"},
			SourceHash: "sha256:s1",
			RenderHash: "sha256:r1",
			Status:     "ok",
		}},
	}
	pb, _ := json.MarshalIndent(pm, "", "  ")
	pb = append(pb, '\n')
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "projections", "manifest.json"), pb, 0o644); err != nil {
		t.Fatalf("write projection manifest failed: %v", err)
	}
	eb, _ := json.MarshalIndent(model.ExtensionManifest{V: 1, Entries: []model.ExtensionManifestEntry{}}, "", "  ")
	eb = append(eb, '\n')
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "extensions", "manifest.json"), eb, 0o644); err != nil {
		t.Fatalf("write extension manifest failed: %v", err)
	}

	if _, _, err := contract.LoadOrInit(repo); err != nil {
		t.Fatalf("load/init contract failed: %v", err)
	}
	if err := state.Save(repo, state.State{
		V:            1,
		ContractHash: "",
		ManagedPaths: map[string]state.ManagedRecord{},
		Projection: &state.ProjectionState{
			RenderHash: "sha256:state-mismatch",
			SourceHash: "sha256:s1",
			Status:     "ok",
		},
	}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	rep, err := RunParity(repo)
	if err != nil {
		t.Fatalf("run parity failed: %v", err)
	}
	if rep.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s", rep.Outcome)
	}
	if !hasFinding(rep.BlockingIssues, "projection_missing") {
		t.Fatalf("expected projection_missing finding")
	}
	if !hasFinding(rep.BlockingIssues, "render_hash_mismatch") {
		t.Fatalf("expected render_hash_mismatch finding")
	}
}

func TestRunParityDetectsManagedBlockCorruption(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".cursor"), 0o755); err != nil {
		t.Fatalf("mkdir cursor failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# Root\n"), 0o644); err != nil {
		t.Fatalf("write root AGENTS failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".cursor", "AGENTS.md"), []byte("tampered\n"), 0o644); err != nil {
		t.Fatalf("write managed file failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta", "projections"), 0o755); err != nil {
		t.Fatalf("mkdir projections failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta", "extensions"), 0o755); err != nil {
		t.Fatalf("mkdir extensions failed: %v", err)
	}
	pm := model.ProjectionManifest{
		V: 1,
		Entries: []model.ProjectionManifestEntry{{
			Interface:  "cursor",
			Kind:       "copy",
			Files:      []string{".cursor/AGENTS.md"},
			SourceHash: "sha256:s1",
			RenderHash: "sha256:r1",
			Status:     "ok",
		}},
	}
	pb, _ := json.MarshalIndent(pm, "", "  ")
	pb = append(pb, '\n')
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "projections", "manifest.json"), pb, 0o644); err != nil {
		t.Fatalf("write projection manifest failed: %v", err)
	}
	eb, _ := json.MarshalIndent(model.ExtensionManifest{V: 1, Entries: []model.ExtensionManifestEntry{}}, "", "  ")
	eb = append(eb, '\n')
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "extensions", "manifest.json"), eb, 0o644); err != nil {
		t.Fatalf("write extension manifest failed: %v", err)
	}
	if _, _, err := contract.LoadOrInit(repo); err != nil {
		t.Fatalf("load/init contract failed: %v", err)
	}
	if err := state.Save(repo, state.State{
		V: 1,
		ManagedPaths: map[string]state.ManagedRecord{
			".cursor/AGENTS.md": {
				Interface:   "cursor",
				Kind:        "copy",
				Fingerprint: "sha256:fp",
				TemplateID:  "cursor:agents-md@1",
				Target:      "AGENTS.md",
			},
		},
	}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	rep, err := RunParity(repo)
	if err != nil {
		t.Fatalf("run parity failed: %v", err)
	}
	if !hasFinding(rep.BlockingIssues, "managed_block_corruption") {
		t.Fatalf("expected managed_block_corruption finding, got %#v", rep.BlockingIssues)
	}
}

func TestRunParityWarnsOutputSurfaceMismatch(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("ATRAKTA_STATUS_JSON", "0")
	c := contract.Default(repo)
	c.Parity.OutputSurface.PlanFormat = "json"
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta", "projections"), 0o755); err != nil {
		t.Fatalf("mkdir projections failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta", "extensions"), 0o755); err != nil {
		t.Fatalf("mkdir extensions failed: %v", err)
	}
	pb, _ := json.MarshalIndent(model.ProjectionManifest{V: 1, Entries: []model.ProjectionManifestEntry{}}, "", "  ")
	pb = append(pb, '\n')
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "projections", "manifest.json"), pb, 0o644); err != nil {
		t.Fatalf("write projection manifest failed: %v", err)
	}
	eb, _ := json.MarshalIndent(model.ExtensionManifest{V: 1, Entries: []model.ExtensionManifestEntry{}}, "", "  ")
	eb = append(eb, '\n')
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "extensions", "manifest.json"), eb, 0o644); err != nil {
		t.Fatalf("write extension manifest failed: %v", err)
	}
	if err := state.Save(repo, state.State{V: 1, ManagedPaths: map[string]state.ManagedRecord{}}); err != nil {
		t.Fatalf("save state failed: %v", err)
	}
	rep, err := RunParity(repo)
	if err != nil {
		t.Fatalf("run parity failed: %v", err)
	}
	if rep.Outcome != "WARN" {
		t.Fatalf("expected WARN, got %s", rep.Outcome)
	}
	if !hasFinding(rep.Warnings, "output_surface_mismatch") {
		t.Fatalf("expected output_surface_mismatch warning")
	}
}

func hasFinding(list []ParityFinding, code string) bool {
	for _, f := range list {
		if f.Code == code {
			return true
		}
	}
	return false
}
