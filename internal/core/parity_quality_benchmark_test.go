package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atrakta/internal/bootstrap"
	"atrakta/internal/brownfield"
	"atrakta/internal/contract"
	"atrakta/internal/core"
	"atrakta/internal/doctor"
	"atrakta/internal/projection"
	"atrakta/internal/registry"
)

func BenchmarkProjectionRender(b *testing.B) {
	repo := b.TempDir()
	mustWriteTB(b, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	c := contract.Default(repo)
	cb, err := contract.Save(repo, c)
	if err != nil {
		b.Fatalf("save contract failed: %v", err)
	}
	reg := registry.ApplyOverrides(registry.Default(), c)
	targets := []string{"cursor", "claude_code", "codex_cli"}
	source := "constitution\n"

	b.ResetTimer()
	for b.Loop() {
		if _, err := projection.RequiredForTargets(repo, c, reg, targets, contract.ContractHash(cb), source); err != nil {
			b.Fatalf("projection render failed: %v", err)
		}
	}
}

func BenchmarkParityDoctor(b *testing.B) {
	repo := b.TempDir()
	mustWriteTB(b, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		b.Fatalf("warmup start failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		rep, err := doctor.RunParity(repo)
		if err != nil {
			b.Fatalf("run parity failed: %v", err)
		}
		if rep.Outcome == "BLOCKED" {
			b.Fatalf("unexpected blocked parity outcome")
		}
	}
}

func BenchmarkProjectionRepair(b *testing.B) {
	repo := b.TempDir()
	mustWriteTB(b, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		b.Fatalf("warmup start failed: %v", err)
	}
	p := filepath.Join(repo, ".cursor", "AGENTS.md")

	b.ResetTimer()
	for b.Loop() {
		if err := os.Remove(p); err != nil {
			b.Fatalf("remove projected file failed: %v", err)
		}
		if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
			b.Fatalf("repair start failed: %v", err)
		}
	}
}

func BenchmarkExtensionRender(b *testing.B) {
	repo := b.TempDir()
	mustWriteTB(b, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	c := contract.Default(repo)
	c.Extensions.Plugins = []contract.ExtensionEntry{{ID: "demo-plugin"}}
	on := true
	if c.Extensions.Hooks == nil {
		c.Extensions.Hooks = &contract.HooksExtension{}
	}
	if c.Extensions.Hooks.Shell == nil {
		c.Extensions.Hooks.Shell = &contract.ShellHooks{}
	}
	c.Extensions.Hooks.Shell.OnCD = &on
	if _, err := contract.Save(repo, c); err != nil {
		b.Fatalf("save contract failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		rep, err := doctor.RunIntegration(repo)
		if err != nil {
			b.Fatalf("run integration failed: %v", err)
		}
		if rep.Outcome == "BLOCKED" {
			b.Fatalf("unexpected blocked integration outcome")
		}
	}
}

func TestParityConsistencyRate(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("initial start failed: %v", err)
	}
	base := mustManifestHashForFile(t, repo, ".cursor/AGENTS.md")

	const runs = 5
	consistent := 0
	for i := 0; i < runs; i++ {
		if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
			t.Fatalf("rerun start failed: %v", err)
		}
		if got := mustManifestHashForFile(t, repo, ".cursor/AGENTS.md"); got == base {
			consistent++
		}
	}
	rate := float64(consistent) / float64(runs)
	if rate < 1.0 {
		t.Fatalf("parity consistency rate below target: %.3f", rate)
	}
}

func TestParityDriftFalsePositiveRate(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("initial start failed: %v", err)
	}

	const runs = 5
	falsePositives := 0
	for i := 0; i < runs; i++ {
		rep, err := doctor.RunParity(repo)
		if err != nil {
			t.Fatalf("run parity failed: %v", err)
		}
		if len(rep.BlockingIssues) > 0 {
			falsePositives++
		}
	}
	rate := float64(falsePositives) / float64(runs)
	if rate > 0 {
		t.Fatalf("parity drift false positive rate too high: %.3f", rate)
	}
}

func TestParityDriftFalseNegativeRate(t *testing.T) {
	const runs = 5
	falseNegatives := 0
	for i := 0; i < runs; i++ {
		repo := t.TempDir()
		mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
		if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
			t.Fatalf("initial start failed: %v", err)
		}
		p := filepath.Join(repo, ".cursor", "AGENTS.md")
		if err := os.Remove(p); err != nil {
			t.Fatalf("remove projected file failed: %v", err)
		}
		if err := os.WriteFile(p, []byte("manual drift\n"), 0o644); err != nil {
			t.Fatalf("write drift file failed: %v", err)
		}
		rep, err := doctor.RunParity(repo)
		if err != nil {
			t.Fatalf("run parity failed: %v", err)
		}
		if len(rep.BlockingIssues) == 0 {
			falseNegatives++
		}
	}
	rate := float64(falseNegatives) / float64(runs)
	if rate > 0 {
		t.Fatalf("parity drift false negative rate too high: %.3f", rate)
	}
}

func TestBrownfieldAppendIdempotent(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "# Existing\n")
	if _, _, err := bootstrap.EnsureRootAGENTSWithMode(repo, "append", ""); err != nil {
		t.Fatalf("first append failed: %v", err)
	}
	if _, _, err := bootstrap.EnsureRootAGENTSWithMode(repo, "append", ""); err != nil {
		t.Fatalf("second append failed: %v", err)
	}
	if _, _, err := bootstrap.EnsureRootAGENTSWithMode(repo, "append", ""); err != nil {
		t.Fatalf("third append failed: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS failed: %v", err)
	}
	if strings.Count(string(b), "<!-- ATRAKTA_MANAGED:START -->") != 1 {
		t.Fatalf("managed append block duplicated")
	}
}

func TestBrownfieldNoOverwrite(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "# Existing\n")
	mustWrite(t, filepath.Join(repo, "CLAUDE.md"), "# User\n")
	desired := []projection.Desired{{Path: "CLAUDE.md", Interface: "claude_code"}}
	conflicts, err := brownfield.FindConflicts(repo, desired, true)
	if err != nil {
		t.Fatalf("find conflicts failed: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0].Path != "CLAUDE.md" {
		t.Fatalf("expected CLAUDE overwrite risk conflict, got %#v", conflicts)
	}
}

func mustWriteTB(tb testing.TB, p, text string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		tb.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		tb.Fatalf("write failed: %v", err)
	}
}
