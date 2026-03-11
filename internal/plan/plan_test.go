package plan

import (
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/state"
)

func TestBuildSetsRequiredPermissionFromOps(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	pl, err := Build(Input{
		RepoRoot:  repo,
		Contract:  c,
		Detect:    model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:     state.Empty(""),
		FeatureID: "feat",
		Projections: []projection.Desired{{
			Path:        filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md")),
			Source:      "AGENTS.md",
			Target:      "AGENTS.md",
			Fingerprint: "fp",
			Interface:   "cursor",
			TemplateID:  "cursor:agents-md@1",
		}},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if pl.RequiredPermission != model.PermissionWorkspaceWrite {
		t.Fatalf("expected workspace_write, got %s", pl.RequiredPermission)
	}
}

func TestBuildReadOnlyPermissionWhenNoOps(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	pl, err := Build(Input{
		RepoRoot:    repo,
		Contract:    c,
		Detect:      model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:       state.Empty(""),
		FeatureID:   "feat",
		Projections: nil,
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if pl.RequiredPermission != model.PermissionReadOnly {
		t.Fatalf("expected read_only, got %s", pl.RequiredPermission)
	}
}

func TestBuildAdoptsEquivalentUnmanagedProjection(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	target := filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md"))
	if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, target)), 0o755); err != nil {
		t.Fatalf("mkdir target dir failed: %v", err)
	}
	d := projection.Desired{
		Path:        target,
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:fp-adopt",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
	}
	body := projection.ManagedContentForPath(target, d.TemplateID, d.Fingerprint, "# hello\n")
	if err := os.WriteFile(filepath.Join(repo, target), []byte(body), 0o644); err != nil {
		t.Fatalf("write projection failed: %v", err)
	}

	pl, err := Build(Input{
		RepoRoot:    repo,
		Contract:    c,
		Detect:      model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:       state.Empty(""),
		FeatureID:   "feat-adopt",
		Projections: []projection.Desired{d},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pl.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(pl.Ops))
	}
	if pl.Ops[0].Op != "adopt" {
		t.Fatalf("expected adopt op, got %s", pl.Ops[0].Op)
	}
	if pl.Ops[0].RequiresApproval {
		t.Fatalf("adopt should not require approval")
	}
}

func TestBuildCompactsDuplicateProjectionPaths(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	d1 := projection.Desired{
		Path:        filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md")),
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:fp-dup",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
	}
	d2 := d1
	pl, err := Build(Input{
		RepoRoot:  repo,
		Contract:  c,
		Detect:    model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:     state.Empty(""),
		FeatureID: "feat-dup",
		Projections: []projection.Desired{
			d1,
			d2,
		},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pl.Ops) != 1 {
		t.Fatalf("expected duplicate paths to compact to 1 op, got %d", len(pl.Ops))
	}
}

func TestBuildRecreatesManagedProjectionWhenArtifactMissing(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	path := filepath.ToSlash(filepath.Join(".cursor", "AGENTS.md"))
	st := state.Empty("")
	st.ManagedPaths[path] = state.ManagedRecord{
		Interface:   "cursor",
		Kind:        "copy",
		Fingerprint: "sha256:fp-keep",
		TemplateID:  "cursor:agents-md@1",
		Target:      "AGENTS.md",
	}
	d := projection.Desired{
		Path:        path,
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:fp-keep",
		Interface:   "cursor",
		TemplateID:  "cursor:agents-md@1",
	}
	pl, err := Build(Input{
		RepoRoot:    repo,
		Contract:    c,
		Detect:      model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:       st,
		FeatureID:   "feat-repair",
		Projections: []projection.Desired{d},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pl.Ops) != 1 {
		t.Fatalf("expected recreate op for missing managed artifact, got %d ops", len(pl.Ops))
	}
	if pl.Ops[0].Op != "link" {
		t.Fatalf("expected link op for recreate, got %s", pl.Ops[0].Op)
	}
}

func TestBuildUsesCopyWhenDesiredTargetIsEmpty(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# hello\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	d := projection.Desired{
		Path:        ".cursor/rules/00-atrakta.mdc",
		Source:      "AGENTS.md",
		Target:      "",
		Fingerprint: "sha256:cursor-rule",
		Interface:   "cursor",
		TemplateID:  "cursor:cursor-rule@1",
	}
	pl, err := Build(Input{
		RepoRoot:    repo,
		Contract:    c,
		Detect:      model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:       state.Empty(""),
		FeatureID:   "feat-cursor-rule",
		Projections: []projection.Desired{d},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pl.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(pl.Ops))
	}
	if pl.Ops[0].Op != "copy" {
		t.Fatalf("expected copy op for empty target projection, got %s", pl.Ops[0].Op)
	}
}

func TestBuildAdoptsSelfSourceProjectionWithoutHeaderRewrite(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# root\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS failed: %v", err)
	}
	d := projection.Desired{
		Path:        "AGENTS.md",
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: "sha256:codex-agents",
		Interface:   "codex_cli",
		TemplateID:  "codex_cli:agents-md@1",
	}
	pl, err := Build(Input{
		RepoRoot:    repo,
		Contract:    c,
		Detect:      model.DetectResult{TargetSet: []string{"codex_cli"}, PruneAllowed: false, Reason: model.ReasonExplicit},
		State:       state.Empty(""),
		FeatureID:   "feat-codex",
		Projections: []projection.Desired{d},
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(pl.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(pl.Ops))
	}
	if pl.Ops[0].Op != "adopt" {
		t.Fatalf("expected adopt op for self-source projection, got %s", pl.Ops[0].Op)
	}
}
