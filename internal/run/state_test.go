package run

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCanonicalState(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		root := t.TempDir()
		got, err := DetectCanonicalState(root)
		if err != nil {
			t.Fatalf("detect canonical state: %v", err)
		}
		if got != StateNone {
			t.Fatalf("state=%q", got)
		}
	})

	t.Run("canonical present", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ".atrakta", "canonical", "policies", "registry")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(path, "index.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write index: %v", err)
		}
		got, err := DetectCanonicalState(root)
		if err != nil {
			t.Fatalf("detect canonical state: %v", err)
		}
		if got != StateCanonicalPresent {
			t.Fatalf("state=%q", got)
		}
	})

	t.Run("onboarding complete", func(t *testing.T) {
		root := t.TempDir()
		policyPath := filepath.Join(root, ".atrakta", "canonical", "policies", "registry")
		if err := os.MkdirAll(policyPath, 0o755); err != nil {
			t.Fatalf("mkdir policy path: %v", err)
		}
		if err := os.WriteFile(filepath.Join(policyPath, "index.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write index: %v", err)
		}
		statePath := filepath.Join(root, ".atrakta", "state")
		if err := os.MkdirAll(statePath, 0o755); err != nil {
			t.Fatalf("mkdir state path: %v", err)
		}
		if err := os.WriteFile(filepath.Join(statePath, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write onboarding state: %v", err)
		}
		got, err := DetectCanonicalState(root)
		if err != nil {
			t.Fatalf("detect canonical state: %v", err)
		}
		if got != StateOnboardingComplete {
			t.Fatalf("state=%q", got)
		}
	})

	t.Run("partial state", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ".atrakta", "state")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir state path: %v", err)
		}
		if err := os.WriteFile(filepath.Join(path, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write onboarding state: %v", err)
		}
		got, err := DetectCanonicalState(root)
		if err != nil {
			t.Fatalf("detect canonical state: %v", err)
		}
		if got != StatePartialState {
			t.Fatalf("state=%q", got)
		}
	})

	t.Run("corrupt state", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, ".atrakta", "canonical")
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir canonical path: %v", err)
		}
		got, err := DetectCanonicalState(root)
		if err != nil {
			t.Fatalf("detect canonical state: %v", err)
		}
		if got != StateCorruptState {
			t.Fatalf("state=%q", got)
		}
	})
}
