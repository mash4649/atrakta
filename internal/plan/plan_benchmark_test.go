package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/state"
)

func BenchmarkBuildNoopManagedScaling(b *testing.B) {
	for _, n := range []int{100, 500, 1000} {
		b.Run(fmt.Sprintf("managed_%d", n), func(b *testing.B) {
			repo := b.TempDir()
			if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# benchmark\n"), 0o644); err != nil {
				b.Fatalf("write AGENTS: %v", err)
			}
			c := contract.Default(repo)
			st := state.Empty("sha256:contract")
			projs := make([]projection.Desired, 0, n)
			for i := 0; i < n; i++ {
				path := fmt.Sprintf(".bench%d/AGENTS.md", i)
				fp := fmt.Sprintf("sha256:fp-%d", i)
				iface := fmt.Sprintf("if%d", i)
				tpl := fmt.Sprintf("%s:agents-md@1", iface)
				d := projection.Desired{
					Path:        path,
					Source:      "AGENTS.md",
					Target:      "AGENTS.md",
					Fingerprint: fp,
					Interface:   iface,
					TemplateID:  tpl,
				}
				abs := filepath.Join(repo, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
					b.Fatalf("mkdir: %v", err)
				}
				body := projection.ManagedContentForPath(path, tpl, fp, "# benchmark\n")
				if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
					b.Fatalf("write projection: %v", err)
				}
				st.ManagedPaths[path] = state.ManagedRecord{
					Interface:   iface,
					Kind:        "copy",
					Fingerprint: fp,
					TemplateID:  tpl,
				}
				projs = append(projs, d)
			}

			in := Input{
				RepoRoot:    repo,
				Contract:    c,
				Detect:      model.DetectResult{TargetSet: []string{"cursor"}, PruneAllowed: false, Reason: model.ReasonExplicit},
				State:       st,
				FeatureID:   "bench-noop",
				Projections: projs,
			}
			b.ResetTimer()
			for b.Loop() {
				pl, err := Build(in)
				if err != nil {
					b.Fatalf("build failed: %v", err)
				}
				if len(pl.Ops) != 0 {
					b.Fatalf("expected no ops, got %d", len(pl.Ops))
				}
			}
		})
	}
}
