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

func BenchmarkApplyScaling(b *testing.B) {
	for _, n := range []int{10, 100, 300} {
		b.Run(fmt.Sprintf("ops_%d", n), func(b *testing.B) {
			repo := b.TempDir()
			if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# benchmark\n"), 0o644); err != nil {
				b.Fatalf("write AGENTS: %v", err)
			}
			c := contract.Default(repo)
			st := state.Empty("sha256:contract")
			ops := make([]model.Operation, 0, n)
			for i := 0; i < n; i++ {
				path := fmt.Sprintf(".bench%d/AGENTS.md", i)
				op := model.Operation{
					Op:          "link",
					Path:        path,
					Target:      "AGENTS.md",
					Source:      "AGENTS.md",
					Fingerprint: fmt.Sprintf("sha256:fp-%d", i),
					Interface:   fmt.Sprintf("if%d", i),
					TemplateID:  fmt.Sprintf("if%d:agents-md@1", i),
				}
				ops = append(ops, op)
			}
			pl := model.PlanResult{ID: "plan-1", FeatureID: "bench", Ops: ops}

			// Warmup creates initial artifacts; benchmark captures steady repeated apply behavior.
			_ = Run(Input{RepoRoot: repo, Contract: c, ContractHash: "sha256:contract", State: st, Plan: pl, Approved: true, SourceAGENTS: "# benchmark\n"})

			b.ResetTimer()
			for b.Loop() {
				_ = Run(Input{RepoRoot: repo, Contract: c, ContractHash: "sha256:contract", State: st, Plan: pl, Approved: true, SourceAGENTS: "# benchmark\n"})
			}
		})
	}
}
