package detect

import (
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
	"atrakta/internal/state"
)

func TestExplicitDisabledInterfaceFails(t *testing.T) {
	c := contract.Default("/tmp/repo")
	c.Hints = &contract.Hints{DisableInterfaces: []string{"cursor"}}
	reg := registry.ApplyOverrides(registry.Default(), c)
	_, err := Run(Input{RepoRoot: "/tmp/repo", Contract: c, Registry: reg, State: state.Empty(""), Explicit: []string{"cursor"}})
	if err == nil {
		t.Fatalf("expected explicit disabled interface failure")
	}
}
