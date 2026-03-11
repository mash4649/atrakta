package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/model"
)

func TestUpdateFromApplyCreatesAndUpdatesManifests(t *testing.T) {
	repo := t.TempDir()

	ap1 := model.ApplyResult{Ops: []model.OpResult{
		{
			Path:        ".cursor/AGENTS.md",
			Op:          "link",
			Status:      "ok",
			Interface:   "cursor",
			TemplateID:  "cursor:agents-md@1",
			Kind:        "link",
			Fingerprint: "sha256:fp1",
		},
	}}
	res1, err := UpdateFromApply(repo, ap1, "sha256:source1")
	if err != nil {
		t.Fatalf("update manifest (first): %v", err)
	}
	if res1.ProjectionEntries != 1 {
		t.Fatalf("expected 1 projection entry, got %d", res1.ProjectionEntries)
	}

	pmPath := filepath.Join(repo, ".atrakta", "projections", "manifest.json")
	if _, err := os.Stat(pmPath); err != nil {
		t.Fatalf("projection manifest missing: %v", err)
	}
	emPath := filepath.Join(repo, ".atrakta", "extensions", "manifest.json")
	if _, err := os.Stat(emPath); err != nil {
		t.Fatalf("extension manifest missing: %v", err)
	}

	ap2 := model.ApplyResult{Ops: []model.OpResult{
		{
			Path:      ".cursor/AGENTS.md",
			Op:        "unlink",
			Status:    "ok",
			Interface: "cursor",
			Kind:      "link",
		},
	}}
	res2, err := UpdateFromApply(repo, ap2, "sha256:source2")
	if err != nil {
		t.Fatalf("update manifest (second): %v", err)
	}
	if res2.ProjectionEntries != 0 {
		t.Fatalf("expected 0 projection entries after unlink, got %d", res2.ProjectionEntries)
	}

	b, err := os.ReadFile(pmPath)
	if err != nil {
		t.Fatalf("read projection manifest: %v", err)
	}
	var pm model.ProjectionManifest
	if err := json.Unmarshal(b, &pm); err != nil {
		t.Fatalf("parse projection manifest: %v", err)
	}
	if len(pm.Entries) != 0 {
		t.Fatalf("expected empty entries after unlink, got %d", len(pm.Entries))
	}
}

func TestReadStatusHandlesMissingAndExistingManifests(t *testing.T) {
	repo := t.TempDir()

	st0, err := ReadStatus(repo)
	if err != nil {
		t.Fatalf("read status (missing): %v", err)
	}
	if st0.ProjectionExists || st0.ExtensionExists {
		t.Fatalf("expected manifests to be missing initially")
	}
	if st0.Projection.V != 1 || st0.Extension.V != 1 {
		t.Fatalf("expected default manifest versions")
	}

	ap := model.ApplyResult{Ops: []model.OpResult{{
		Path:        ".cursor/AGENTS.md",
		Op:          "link",
		Status:      "ok",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
		Kind:        "link",
		Fingerprint: "sha256:fp1",
	}}}
	if _, err := UpdateFromApply(repo, ap, "sha256:source1"); err != nil {
		t.Fatalf("update manifest: %v", err)
	}

	st1, err := ReadStatus(repo)
	if err != nil {
		t.Fatalf("read status (existing): %v", err)
	}
	if !st1.ProjectionExists || !st1.ExtensionExists {
		t.Fatalf("expected manifests to exist after update")
	}
	if len(st1.Projection.Entries) != 1 {
		t.Fatalf("expected 1 projection entry, got %d", len(st1.Projection.Entries))
	}
}
