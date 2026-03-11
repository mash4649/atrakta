package core_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"atrakta/internal/core"
	"atrakta/internal/doctor"
)

func TestWindowsNativeParity(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-native parity gate test")
	}

	repo := t.TempDir()
	// Strict gate runs go_test_compile, so provide a minimal Go module fixture.
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/windowsnativeparity\n\ngo 1.22\n")
	mustWrite(t, filepath.Join(repo, "internal", "dummy", "dummy.go"), "package dummy\n\nfunc Nop() {}\n")
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{
		Interfaces: "cursor,claude_code,codex_cli",
		SyncLevel:  "2",
	})
	if err != nil {
		t.Fatalf("strict start failed on windows: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE outcome, got %s", res.Step.Outcome)
	}

	mustExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
	mustExist(t, filepath.Join(repo, ".cursor", "rules", "00-atrakta.mdc"))
	mustExist(t, filepath.Join(repo, "CLAUDE.md"))
	mustExist(t, filepath.Join(repo, ".codex", "config.toml"))
	mustExist(t, filepath.Join(repo, ".atrakta", "projections", "manifest.json"))
	mustExist(t, filepath.Join(repo, ".atrakta", "extensions", "manifest.json"))

	parity, err := doctor.RunParity(repo)
	if err != nil {
		t.Fatalf("doctor parity failed: %v", err)
	}
	if len(parity.BlockingIssues) != 0 {
		t.Fatalf("expected no parity blocking issues, got %#v", parity.BlockingIssues)
	}

	integration, err := doctor.RunIntegration(repo)
	if err != nil {
		t.Fatalf("doctor integration failed: %v", err)
	}
	if integration.Outcome == "BLOCKED" {
		t.Fatalf("expected integration to avoid BLOCKED outcome, got %#v", integration.BlockingIssues)
	}
}
