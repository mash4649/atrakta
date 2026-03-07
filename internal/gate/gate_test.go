package gate

import (
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/progress"
	"atrakta/internal/registry"
)

func TestUnknownQuickCheckFails(t *testing.T) {
	c := contract.Default("/tmp/repo")
	c.Quality = &contract.Quality{QuickChecks: []string{"unknown_check"}}
	in := Input{
		RepoRoot: "/tmp/repo",
		Contract: c,
		Detect:   model.DetectResult{TargetSet: []string{"cursor"}, Reason: model.ReasonExplicit},
		Plan: model.PlanResult{
			ID:  "p1",
			Ops: []model.Operation{{Path: ".cursor/AGENTS.md", Op: "link"}},
		},
		Apply:    model.ApplyResult{PlanID: "p1", Result: "success", Ops: []model.OpResult{{Path: ".cursor/AGENTS.md", Op: "link", Status: "ok"}}},
		Approved: true,
		Registry: registry.Default(),
		Quality:  c.Quality,
		Progress: progress.Empty(),
	}
	res := Run(in)
	if res.Quick != model.GateFail {
		t.Fatalf("expected quick fail, got %#v", res)
	}
}

func TestReadOnlySecurityProfileBlocksMutations(t *testing.T) {
	c := contract.Default("/tmp/repo")
	c.Security.Profile = "read_only"
	in := Input{
		RepoRoot: "/tmp/repo",
		Contract: c,
		Detect:   model.DetectResult{TargetSet: []string{"cursor"}, Reason: model.ReasonExplicit},
		Plan: model.PlanResult{
			ID:                 "p2",
			RequiredPermission: model.PermissionWorkspaceWrite,
			Ops:                []model.Operation{{Path: ".cursor/AGENTS.md", Op: "link"}},
		},
		Apply:    model.ApplyResult{PlanID: "p2", Result: "success", Ops: []model.OpResult{{Path: ".cursor/AGENTS.md", Op: "link", Status: "ok"}}},
		Approved: true,
		Registry: registry.Default(),
		Quality:  c.Quality,
		Progress: progress.Empty(),
	}
	res := Run(in)
	if res.Safety != model.GateFail {
		t.Fatalf("expected safety fail, got %#v", res)
	}
}
