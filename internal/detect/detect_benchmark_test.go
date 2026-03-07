package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
	"atrakta/internal/state"
)

func BenchmarkDetectScaling(b *testing.B) {
	for _, n := range []int{10, 100, 500} {
		b.Run(fmt.Sprintf("interfaces_%d", n), func(b *testing.B) {
			repo := b.TempDir()
			c := contract.Default(repo)
			c.Interfaces.Supported = make([]string, 0, n)
			c.Interfaces.CoreSet = []string{"if0"}
			reg := registry.Registry{Entries: map[string]registry.Entry{}}
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("if%d", i)
				c.Interfaces.Supported = append(c.Interfaces.Supported, id)
				anchor := "." + id + "/"
				reg.Entries[id] = registry.Entry{InterfaceID: id, Surface: "editor", Anchor: anchor, ProjectionDir: anchor}
			}
			if err := os.MkdirAll(filepath.Join(repo, ".if0"), 0o755); err != nil {
				b.Fatalf("mkdir anchor: %v", err)
			}

			in := Input{RepoRoot: repo, Contract: c, Registry: reg, State: state.Empty(""), Explicit: nil}
			b.ResetTimer()
			for b.Loop() {
				if _, err := Run(in); err != nil {
					b.Fatalf("detect failed: %v", err)
				}
			}
		})
	}
}
