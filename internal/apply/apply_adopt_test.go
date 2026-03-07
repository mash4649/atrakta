package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/state"
)

func TestAdoptSkipsForEquivalentCopy(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	target := filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md"))
	if err := os.MkdirAll(filepath.Join(repo, ".cursor"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor failed: %v", err)
	}
	op := model.Operation{
		Op:          "adopt",
		Path:        target,
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:fp-copy",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
	}
	body := projection.ManagedContentForPath(target, op.TemplateID, op.Fingerprint, "# x\n")
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(target)), []byte(body), 0o644); err != nil {
		t.Fatalf("write managed body failed: %v", err)
	}
	res := Run(Input{
		RepoRoot:     repo,
		Contract:     contract.Default(repo),
		ContractHash: "sha256:contract",
		State:        state.Empty("sha256:contract"),
		Plan:         model.PlanResult{ID: "plan-1", FeatureID: "feat", Ops: []model.Operation{op}},
		Approved:     true,
		SourceAGENTS: "# x\n",
	})
	if res.Result != "success" {
		t.Fatalf("expected success, got %s", res.Result)
	}
	if len(res.Ops) != 1 {
		t.Fatalf("expected single op result, got %d", len(res.Ops))
	}
	if res.Ops[0].Status != "skipped" || res.Ops[0].Kind != "copy" {
		t.Fatalf("expected skipped copy adopt, got status=%s kind=%s", res.Ops[0].Status, res.Ops[0].Kind)
	}
}

func TestAdoptSkipsForEquivalentSymlink(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".cursor"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor failed: %v", err)
	}
	target := filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md"))
	if err := os.Symlink("../AGENTS.md", filepath.Join(repo, ".cursor", "AGENTS.md")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}
	op := model.Operation{
		Op:          "adopt",
		Path:        target,
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:fp-link",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
	}
	res := Run(Input{
		RepoRoot:     repo,
		Contract:     contract.Default(repo),
		ContractHash: "sha256:contract",
		State:        state.Empty("sha256:contract"),
		Plan:         model.PlanResult{ID: "plan-2", FeatureID: "feat", Ops: []model.Operation{op}},
		Approved:     true,
		SourceAGENTS: "# x\n",
	})
	if len(res.Ops) != 1 {
		t.Fatalf("expected single op result, got %d", len(res.Ops))
	}
	if res.Ops[0].Status != "skipped" || res.Ops[0].Kind != "link" {
		t.Fatalf("expected skipped link adopt, got status=%s kind=%s", res.Ops[0].Status, res.Ops[0].Kind)
	}
}

func TestAdoptFailsWhenEquivalentCheckBreaks(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	target := filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md"))
	if err := os.MkdirAll(filepath.Join(repo, ".cursor"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(target)), []byte("manual drift\n"), 0o644); err != nil {
		t.Fatalf("write drift file failed: %v", err)
	}
	op := model.Operation{
		Op:          "adopt",
		Path:        target,
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:fp-drift",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
	}
	res := Run(Input{
		RepoRoot:     repo,
		Contract:     contract.Default(repo),
		ContractHash: "sha256:contract",
		State:        state.Empty("sha256:contract"),
		Plan:         model.PlanResult{ID: "plan-3", FeatureID: "feat", Ops: []model.Operation{op}},
		Approved:     true,
		SourceAGENTS: "# x\n",
	})
	if len(res.Ops) != 1 {
		t.Fatalf("expected single op result, got %d", len(res.Ops))
	}
	if res.Ops[0].Status != "failed" {
		t.Fatalf("expected failed adopt, got %s", res.Ops[0].Status)
	}
	if !strings.Contains(res.Ops[0].Error, "adopt precondition failed") {
		t.Fatalf("unexpected error: %s", res.Ops[0].Error)
	}
}
