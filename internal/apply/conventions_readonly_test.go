package apply

import (
	"strings"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/state"
)

func TestApplyBlocksConventionMutationWhenReadOnly(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	c = contract.CanonicalizeBoundary(c)
	pl := model.PlanResult{
		ID: "plan-conventions-ro",
		Ops: []model.Operation{{
			Op:          "write",
			Path:        "CONVENTIONS.md",
			Source:      "AGENTS.md",
			TemplateID:  "custom:conventions@1",
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
		SourceAGENTS: "rules\n",
	})
	if res.Result != "fail" {
		t.Fatalf("expected fail, got %s", res.Result)
	}
	if len(res.Ops) != 1 {
		t.Fatalf("expected one op result")
	}
	if res.Ops[0].Status != "failed" || !strings.Contains(res.Ops[0].Error, "read-only") {
		t.Fatalf("unexpected op result: %#v", res.Ops[0])
	}
}
