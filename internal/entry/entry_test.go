package entry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

func TestResolveOnboardingDefaultInterface(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	got, err := Resolve(Input{ProjectRoot: root})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Path != PathOnboarding {
		t.Fatalf("path=%q", got.Path)
	}
	if got.CanonicalState != runpkg.StateNone {
		t.Fatalf("state=%q", got.CanonicalState)
	}
	if got.Interface.InterfaceID == "" {
		t.Fatal("interface id is empty")
	}
}

func TestResolveNeedsInputOnNormalPathWhenInterfaceUnknown(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, ".atrakta", "canonical", "policies", "registry")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}

	got, err := Resolve(Input{ProjectRoot: root})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !got.NeedsInput {
		t.Fatal("expected needs input")
	}
	if got.Path != PathNormal {
		t.Fatalf("path=%q", got.Path)
	}
	if got.Interface.InterfaceID != "unresolved" {
		t.Fatalf("interface=%q", got.Interface.InterfaceID)
	}
}

func TestResolveBlockedOnCorruptState(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".atrakta", "canonical"), 0o755); err != nil {
		t.Fatalf("mkdir canonical: %v", err)
	}
	_, err := Resolve(Input{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error")
	}
	var blocked *BlockedStateError
	if !errors.As(err, &blocked) {
		t.Fatalf("expected blocked state error, got %T", err)
	}
	if blocked.CanonicalState != runpkg.StateCorruptState {
		t.Fatalf("canonical state=%q", blocked.CanonicalState)
	}
	if !strings.Contains(blocked.Error(), "corrupt_state") {
		t.Fatalf("diagnostic=%q", blocked.Error())
	}
	if !strings.Contains(blocked.NextAllowedAction, ".atrakta/canonical/") {
		t.Fatalf("next action=%q", blocked.NextAllowedAction)
	}
}

func TestResolveBlockedOnPartialState(t *testing.T) {
	root := t.TempDir()
	statePath := filepath.Join(root, ".atrakta", "state")
	if err := os.MkdirAll(statePath, 0o755); err != nil {
		t.Fatalf("mkdir state path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(statePath, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	_, err := Resolve(Input{ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error")
	}
	var blocked *BlockedStateError
	if !errors.As(err, &blocked) {
		t.Fatalf("expected blocked state error, got %T", err)
	}
	if blocked.CanonicalState != runpkg.StatePartialState {
		t.Fatalf("canonical state=%q", blocked.CanonicalState)
	}
	if !strings.Contains(blocked.Error(), ".atrakta/state/onboarding-state.json") {
		t.Fatalf("diagnostic=%q", blocked.Error())
	}
	if !strings.Contains(blocked.NextAllowedAction, ".atrakta/canonical/policies/registry/index.json") {
		t.Fatalf("next action=%q", blocked.NextAllowedAction)
	}
}

func TestResolveUsesAutoStateOnNormalPath(t *testing.T) {
	root := t.TempDir()
	policyDir := filepath.Join(root, ".atrakta", "canonical", "policies", "registry")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := runpkg.SaveAutoState(root, runpkg.AutoState{InterfaceID: "cursor", InterfaceSource: "flag"}); err != nil {
		t.Fatalf("save auto state: %v", err)
	}

	got, err := Resolve(Input{ProjectRoot: root})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Interface.InterfaceID != "cursor" {
		t.Fatalf("interface id=%q", got.Interface.InterfaceID)
	}
	if got.Interface.Source != "auto" {
		t.Fatalf("interface source=%q", got.Interface.Source)
	}
}
