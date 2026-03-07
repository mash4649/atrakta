package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/core"
)

func BenchmarkStartSteadyState(b *testing.B) {
	repo := b.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("constitution\n"), 0o644); err != nil {
		b.Fatalf("write AGENTS: %v", err)
	}
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", MapTokens: 256, MapRefresh: 600}); err != nil {
		b.Fatalf("warmup start failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", MapTokens: 256, MapRefresh: 600})
		if err != nil {
			b.Fatalf("start failed: %v", err)
		}
		if res.Step.Outcome != "DONE" {
			b.Fatalf("unexpected outcome: %s", res.Step.Outcome)
		}
	}
}
