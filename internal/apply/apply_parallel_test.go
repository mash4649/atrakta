package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/state"
)

func TestParallelApplyAutoForNonDestructiveOps(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS: %v", err)
	}
	c := contract.Default(repo)
	st := state.Empty("sha256:contract")
	ops := make([]model.Operation, 0, 6)
	for i := 0; i < 6; i++ {
		ops = append(ops, model.Operation{
			Op:          "link",
			Path:        fmt.Sprintf(".p%d/AGENTS.md", i),
			Target:      "AGENTS.md",
			Source:      "AGENTS.md",
			Fingerprint: fmt.Sprintf("sha256:fp-%d", i),
			Interface:   fmt.Sprintf("if%d", i),
			TemplateID:  fmt.Sprintf("if%d:agents-md@1", i),
		})
	}
	pl := model.PlanResult{ID: "plan-p", FeatureID: "feat", Ops: ops}
	res := Run(Input{
		RepoRoot:           repo,
		Contract:           c,
		ContractHash:       "sha256:contract",
		State:              st,
		Plan:               pl,
		Approved:           true,
		SourceAGENTS:       "# x\n",
		ParallelMode:       "auto",
		ParallelMaxWorkers: 4,
	})
	if res.Result != "success" {
		t.Fatalf("expected success, got %s", res.Result)
	}
	if len(res.Ops) != len(ops) {
		t.Fatalf("ops count mismatch: got=%d want=%d", len(res.Ops), len(ops))
	}
}

func TestParallelApplyFallsBackWhenDestructiveIncluded(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS: %v", err)
	}
	c := contract.Default(repo)
	st := state.Empty("sha256:contract")
	if err := os.MkdirAll(filepath.Join(repo, ".p"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	managedPath := ".p/old.md"
	managedAbs := filepath.Join(repo, filepath.FromSlash(managedPath))
	managedContent := "# managed_by: atrakta\n# template_id: cursor:agents-md@1\n# fingerprint: sha256:fp-2\n\nlegacy\n"
	if err := os.WriteFile(managedAbs, []byte(managedContent), 0o644); err != nil {
		t.Fatalf("write managed file: %v", err)
	}
	st.ManagedPaths[managedPath] = state.ManagedRecord{
		Interface:   "cursor",
		Kind:        "copy",
		Fingerprint: "sha256:fp-2",
		TemplateID:  "cursor:agents-md@1",
	}
	ops := []model.Operation{
		{
			Op:          "link",
			Path:        ".p/AGENTS.md",
			Target:      "AGENTS.md",
			Source:      "AGENTS.md",
			Fingerprint: "sha256:fp-1",
			Interface:   "cursor",
			TemplateID:  "cursor:agents-md@1",
		},
		{Op: "delete", Path: managedPath, Fingerprint: "sha256:fp-2", Interface: "cursor", TemplateID: "cursor:agents-md@1"},
	}
	pl := model.PlanResult{ID: "plan-f", FeatureID: "feat", Ops: ops}
	res := Run(Input{
		RepoRoot:           repo,
		Contract:           c,
		ContractHash:       "sha256:contract",
		State:              st,
		Plan:               pl,
		Approved:           true,
		SourceAGENTS:       "# x\n",
		ParallelMode:       "on",
		ParallelMaxWorkers: 4,
	})
	if len(res.Ops) == 0 {
		t.Fatalf("expected operation results")
	}
	// Destructive op must fall back to sequential path and still succeed when managed-only proof passes.
	if res.Ops[1].Status != "ok" {
		t.Fatalf("expected destructive op handled in sequential fallback, got status=%s", res.Ops[1].Status)
	}
}

func TestShouldParallelApplyAutoSkipsCompileTimeNoops(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS: %v", err)
	}
	c := contract.Default(repo)
	st := state.Empty("sha256:contract")
	ops := make([]model.Operation, 0, 6)
	for i := 0; i < 6; i++ {
		dir := filepath.Join(repo, fmt.Sprintf(".p%d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		linkPath := filepath.Join(dir, "AGENTS.md")
		if err := os.Symlink("../AGENTS.md", linkPath); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		ops = append(ops, model.Operation{
			Op:          "link",
			Path:        fmt.Sprintf(".p%d/AGENTS.md", i),
			Target:      "AGENTS.md",
			Source:      "AGENTS.md",
			Fingerprint: fmt.Sprintf("sha256:fp-%d", i),
			Interface:   fmt.Sprintf("if%d", i),
			TemplateID:  fmt.Sprintf("if%d:agents-md@1", i),
		})
	}
	in := Input{
		RepoRoot:     repo,
		Contract:     c,
		ContractHash: "sha256:contract",
		State:        st,
		Plan:         model.PlanResult{ID: "plan-noop", FeatureID: "feat", Ops: ops},
		Approved:     true,
		SourceAGENTS: "# x\n",
	}
	resolver := newSourceResolver(repo, "# x\n")
	compiled := compileOps(in, ops, resolver)
	executable := countExecutableOps(compiled)
	if executable != 0 {
		t.Fatalf("expected zero executable ops, got %d", executable)
	}
	if shouldParallelApply(parallelModeAuto, in, compiled, executable) {
		t.Fatalf("expected auto parallel to be disabled for compile-time no-op set")
	}
}

func TestRunParallelHandlesCompileTimeNoopsOnly(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS: %v", err)
	}
	c := contract.Default(repo)
	st := state.Empty("sha256:contract")
	ops := make([]model.Operation, 0, 4)
	for i := 0; i < 4; i++ {
		dir := filepath.Join(repo, fmt.Sprintf(".n%d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		linkPath := filepath.Join(dir, "AGENTS.md")
		if err := os.Symlink("../AGENTS.md", linkPath); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		ops = append(ops, model.Operation{
			Op:          "link",
			Path:        fmt.Sprintf(".n%d/AGENTS.md", i),
			Target:      "AGENTS.md",
			Source:      "AGENTS.md",
			Fingerprint: fmt.Sprintf("sha256:fp-%d", i),
			Interface:   fmt.Sprintf("if%d", i),
			TemplateID:  fmt.Sprintf("if%d:agents-md@1", i),
		})
	}
	in := Input{
		RepoRoot:     repo,
		Contract:     c,
		ContractHash: "sha256:contract",
		State:        st,
		Plan:         model.PlanResult{ID: "plan-noop", FeatureID: "feat", Ops: ops},
		Approved:     true,
		SourceAGENTS: "# x\n",
	}
	resolver := newSourceResolver(repo, "# x\n")
	compiled := compileOps(in, ops, resolver)
	res := runParallel(in, compiled, resolver, 4)
	if res.Result != "success" {
		t.Fatalf("expected success, got %s", res.Result)
	}
	if len(res.Ops) != len(ops) {
		t.Fatalf("ops count mismatch: got=%d want=%d", len(res.Ops), len(ops))
	}
	for i, op := range res.Ops {
		if op.Status != "skipped" || op.Kind != "link" {
			t.Fatalf("op[%d] expected skipped link, got status=%s kind=%s", i, op.Status, op.Kind)
		}
	}
}
