package entry

import (
	"os"
	"path/filepath"
	"testing"

	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

func TestExecuteNeedsInputExitCodeContract(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, ".atrakta", "canonical", "policies", "registry")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}

	code, out, err := Execute(ExecuteInput{ProjectRoot: root})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if code != 2 {
		t.Fatalf("exit code=%d", code)
	}
	if out.NeedsInput == nil {
		t.Fatal("expected needs_input payload")
	}
	if out.NeedsInput.Status != "needs_input" {
		t.Fatalf("status=%q", out.NeedsInput.Status)
	}
}

func TestExecuteOnboardingSuccessExitCodeContract(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	code, out, err := Execute(ExecuteInput{ProjectRoot: root})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if code != 0 {
		t.Fatalf("exit code=%d", code)
	}
	if out.Decision.Path != PathOnboarding {
		t.Fatalf("path=%q", out.Decision.Path)
	}
}

func TestExecuteBlockedOnPartialStateExitCodeContract(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".atrakta", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	code, out, err := Execute(ExecuteInput{ProjectRoot: root})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if code != 1 {
		t.Fatalf("exit code=%d", code)
	}
	if out.HardError == nil {
		t.Fatal("expected hard error payload")
	}
	if out.HardError.CanonicalState != runpkg.StatePartialState {
		t.Fatalf("canonical state=%q", out.HardError.CanonicalState)
	}
	if out.HardError.NextAllowedAction == "" {
		t.Fatal("expected next allowed action")
	}
}

func TestExecuteBlockedOnCorruptStateExitCodeContract(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".atrakta", "canonical"), 0o755); err != nil {
		t.Fatalf("mkdir canonical dir: %v", err)
	}

	code, out, err := Execute(ExecuteInput{ProjectRoot: root})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if code != 1 {
		t.Fatalf("exit code=%d", code)
	}
	if out.HardError == nil {
		t.Fatal("expected hard error payload")
	}
	if out.HardError.CanonicalState != runpkg.StateCorruptState {
		t.Fatalf("canonical state=%q", out.HardError.CanonicalState)
	}
	if out.HardError.RequiredInputs == nil || len(out.HardError.RequiredInputs) == 0 {
		t.Fatal("expected required inputs")
	}
}
