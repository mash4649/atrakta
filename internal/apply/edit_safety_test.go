package apply

import (
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/state"
)

func TestApplyBlocksInvalidGoWhenEditSafetyEnabled(t *testing.T) {
	repo := t.TempDir()
	srcPath := filepath.Join(repo, "invalid.go.src")
	if err := os.WriteFile(srcPath, []byte("package main\nfunc main(\n"), 0o644); err != nil {
		t.Fatalf("write source failed: %v", err)
	}
	c := contract.Default(repo)
	c.EditSafety = &contract.EditSafety{Mode: "anchor+optional_ast"}
	pl := model.PlanResult{
		ID: "plan-edit-safety",
		Ops: []model.Operation{{
			Op:          "write",
			Path:        "main.go",
			Source:      "invalid.go.src",
			TemplateID:  "custom:code@1",
			Fingerprint: "sha256:test",
		}},
	}
	res := Run(Input{
		RepoRoot:     repo,
		Contract:     c,
		ContractHash: "sha256:contract",
		State:        state.Empty("sha256:contract"),
		Plan:         pl,
		Approved:     true,
	})
	if res.Result != "fail" {
		t.Fatalf("expected fail, got %s", res.Result)
	}
	if len(res.Ops) == 0 || res.Ops[0].Status != "failed" {
		t.Fatalf("expected failed op result, got %#v", res.Ops)
	}
	if _, err := os.Stat(filepath.Join(repo, "main.go")); !os.IsNotExist(err) {
		t.Fatalf("expected target file not written")
	}
}
