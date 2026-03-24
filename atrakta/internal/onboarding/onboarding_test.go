package onboarding

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildOnboardingProposalBrownfield(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "# test\n")
	mustMkdir(t, filepath.Join(root, ".cursor", "rules"))
	mustMkdir(t, filepath.Join(root, ".github", "workflows"))
	mustWriteFile(t, filepath.Join(root, "package.json"), "{}\n")
	mustMkdir(t, filepath.Join(root, "docs"))
	mustMkdir(t, filepath.Join(root, "src"))

	got, err := BuildOnboardingProposal(root)
	if err != nil {
		t.Fatalf("build onboarding proposal: %v", err)
	}
	if got.InferredMode != ModeBrownfield {
		t.Fatalf("mode=%q", got.InferredMode)
	}
	if len(got.Conflicts) == 0 {
		t.Fatalf("expected conflicts for duplicate guidance")
	}
	if got.InferredFailure.FailureClass != "legacy_conflict_failure" {
		t.Fatalf("failure class=%q", got.InferredFailure.FailureClass)
	}
	if got.InferredFailure.ResolvedTier != "DEGRADE_TO_STRICT" {
		t.Fatalf("resolved tier=%q", got.InferredFailure.ResolvedTier)
	}
	if got.InferredFailure.StrictTransition != "strict" {
		t.Fatalf("strict transition=%q", got.InferredFailure.StrictTransition)
	}
	if len(got.SuggestedNextActions) == 0 || got.SuggestedNextActions[0] != "review conflicts" {
		t.Fatalf("expected first next action to review conflicts")
	}
	if got.DetectedRisks == nil {
		t.Fatalf("detected risks must not be nil")
	}
}

func TestBuildOnboardingProposalNewProject(t *testing.T) {
	root := t.TempDir()
	got, err := BuildOnboardingProposal(root)
	if err != nil {
		t.Fatalf("build onboarding proposal: %v", err)
	}
	if got.InferredMode != ModeNewProject {
		t.Fatalf("mode=%q", got.InferredMode)
	}
	if len(got.Conflicts) != 0 {
		t.Fatalf("expected no conflicts")
	}
	if got.InferredFailure.FailureClass != "projection_failure" {
		t.Fatalf("failure class=%q", got.InferredFailure.FailureClass)
	}
	if got.InferredFailure.ResolvedTier != "WARN_ONLY" {
		t.Fatalf("resolved tier=%q", got.InferredFailure.ResolvedTier)
	}
	if got.InferredFailure.StrictTransition != "none" {
		t.Fatalf("strict transition=%q", got.InferredFailure.StrictTransition)
	}
	if len(got.SuggestedNextActions) == 0 {
		t.Fatalf("expected next actions")
	}
	if len(got.DetectedRisks) != 0 {
		t.Fatalf("expected no detected risks")
	}
}

func TestDetectMode(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "# test\n")
	mustMkdir(t, filepath.Join(root, ".github", "workflows"))

	mode, err := DetectMode(root)
	if err != nil {
		t.Fatalf("detect mode: %v", err)
	}
	if mode != ModeBrownfield {
		t.Fatalf("mode=%q", mode)
	}
}

func TestRiskSignalDetectionAffectsFailureRouting(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "package.json"), `{
  "scripts": {
    "clean": "rm -rf dist",
    "publish": "curl https://api.example.com/upload -X POST"
  }
}`)

	got, err := BuildOnboardingProposal(root)
	if err != nil {
		t.Fatalf("build onboarding proposal: %v", err)
	}
	if len(got.DetectedRisks) == 0 {
		t.Fatalf("expected risk detection")
	}
	if got.InferredFailure.FailureClass != "policy_failure" && got.InferredFailure.FailureClass != "capability_resolution_failure" {
		t.Fatalf("unexpected failure class=%q", got.InferredFailure.FailureClass)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
