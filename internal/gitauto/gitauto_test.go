package gitauto

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
)

func TestResolveModeDefaultAuto(t *testing.T) {
	t.Setenv("ATRAKTA_GIT_AUTOMATION", "")
	mode := ResolveMode(contract.Default(t.TempDir()))
	if mode != "auto" {
		t.Fatalf("expected auto mode, got %s", mode)
	}
}

func TestResolveModeEnvOverride(t *testing.T) {
	t.Setenv("ATRAKTA_GIT_AUTOMATION", "off")
	mode := ResolveMode(contract.Default(t.TempDir()))
	if mode != "off" {
		t.Fatalf("expected off mode, got %s", mode)
	}
}

func TestCaptureNonGitRepo(t *testing.T) {
	s := Capture(t.TempDir())
	if s.Available {
		t.Fatalf("expected unavailable snapshot for non-git repo")
	}
	if s.Reason != "not_git_repo" {
		t.Fatalf("unexpected reason: %s", s.Reason)
	}
}

func TestWriteCheckpointSkipsWhenDisabled(t *testing.T) {
	repo := t.TempDir()
	pre := Snapshot{Available: false, Reason: "not_git_repo"}
	post := Snapshot{Available: false, Reason: "not_git_repo"}
	cp, wrote, err := WriteCheckpoint(repo, "feat-x", "off", pre, post, model.PlanResult{}, model.ApplyResult{}, model.GateResult{Safety: model.GatePass, Quick: model.GatePass})
	if err != nil {
		t.Fatalf("write checkpoint returned error: %v", err)
	}
	if wrote {
		t.Fatalf("expected checkpoint skipped in off mode")
	}
	if cp.Reason != "mode_off" {
		t.Fatalf("unexpected reason: %s", cp.Reason)
	}
	if _, err := os.Stat(filepath.Join(repo, ".atrakta", "git", "checkpoint-latest.json")); !os.IsNotExist(err) {
		t.Fatalf("checkpoint file should not exist in off mode")
	}
}

func TestEnsureSetupOffWritesBootstrapGuide(t *testing.T) {
	repo := t.TempDir()
	rep, err := EnsureSetup(repo, "off")
	if err != nil {
		t.Fatalf("ensure setup failed: %v", err)
	}
	if rep.Initialized {
		t.Fatalf("off mode should not initialize git")
	}
	if rep.Reason != "mode_off" {
		t.Fatalf("unexpected reason: %s", rep.Reason)
	}
	if _, err := os.Stat(filepath.Join(repo, ".atrakta", "git", "bootstrap.md")); err != nil {
		t.Fatalf("bootstrap guide should exist: %v", err)
	}
}

func TestEnsureSetupAutoInitializesWhenSafe(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}
	repo := t.TempDir()
	rep, err := EnsureSetup(repo, "auto")
	if err != nil {
		t.Fatalf("ensure setup failed: %v", err)
	}
	if !rep.Initialized {
		t.Fatalf("expected git to be initialized in auto mode")
	}
	if !exists(filepath.Join(repo, ".git")) {
		t.Fatalf(".git should be created")
	}
	if _, err := os.Stat(filepath.Join(repo, ".gitignore")); err != nil {
		t.Fatalf(".gitignore should be ensured: %v", err)
	}
}

func TestEnsureSetupAutoDegradesWhenGitUnavailable(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("PATH", "")

	rep, err := EnsureSetup(repo, "auto")
	if err != nil {
		t.Fatalf("auto mode should degrade when git is unavailable: %v", err)
	}
	if rep.Initialized {
		t.Fatalf("auto mode should not initialize git when unavailable")
	}
	if rep.Reason != "git_unavailable" {
		t.Fatalf("unexpected reason: %s", rep.Reason)
	}
	if _, err := os.Stat(filepath.Join(repo, ".atrakta", "git", "bootstrap.md")); err != nil {
		t.Fatalf("bootstrap guide should exist: %v", err)
	}
}

func TestEnsureSetupOnFailsWhenGitUnavailable(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("PATH", "")

	rep, err := EnsureSetup(repo, "on")
	if err == nil {
		t.Fatalf("on mode should fail when git is unavailable")
	}
	if rep.Reason != "git_unavailable" {
		t.Fatalf("unexpected reason: %s", rep.Reason)
	}
}
