package run

import (
	"os"
	"testing"
)

func TestHandoffRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := HandoffBundle{
		Command:           "start",
		CanonicalState:    StateOnboardingComplete,
		Status:            "ok",
		InterfaceID:       "generic-cli",
		InterfaceSource:   "flag",
		ApplyRequested:    true,
		Approved:          true,
		FastPath:          true,
		NextAllowedAction: "inspect",
		NextAction: HandoffNextAction{
			Command: "inspect",
			Hint:    "inspect pipeline executed",
		},
		FeatureSpec: HandoffFeatureSpec{
			Summary:           "managed plan prepared for 3 targets",
			ResolvedTargets:   []string{"agents_md"},
			PortabilityStatus: "supported",
		},
		Acceptance: []string{"requested surface portability is supported"},
		Checkpoint: HandoffCheckpoint{
			AutoStatePath:   "runtime/auto-state.v1.json",
			StartFastPath:   "runtime/start-fast.v1.json",
			RunStatePath:    "state/run-state.json",
			StatePath:       "state.json",
			ProgressPath:    "progress.json",
			TaskGraphPath:   "task-graph.json",
			AuditHeadPath:   "audit/checkpoints/head.json",
			OnboardingState: "state/onboarding-state.json",
		},
	}
	if err := SaveHandoff(root, in); err != nil {
		t.Fatalf("save handoff: %v", err)
	}
	out, err := LoadHandoff(root)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	if out.SchemaVersion == "" {
		t.Fatal("schema version should be set")
	}
	if out.InterfaceID != "generic-cli" {
		t.Fatalf("interface id=%q", out.InterfaceID)
	}
	if !out.FastPath {
		t.Fatal("expected fast path")
	}
	if out.Checkpoint.AutoStatePath == "" {
		t.Fatal("expected checkpoint auto-state path")
	}
	if out.Checkpoint.StartFastPath == "" {
		t.Fatal("expected checkpoint start-fast path")
	}
	if out.Checkpoint.RunStatePath == "" {
		t.Fatal("expected checkpoint run-state path")
	}
	if out.Checkpoint.StatePath == "" {
		t.Fatal("expected checkpoint state path")
	}
	if out.Checkpoint.AuditHeadPath == "" {
		t.Fatal("expected checkpoint audit-head path")
	}
	if out.Checkpoint.OnboardingState == "" {
		t.Fatal("expected checkpoint onboarding-state path")
	}
	if out.UpdatedAt == "" {
		t.Fatal("expected updated_at")
	}
	if out.NextAction.Command != "inspect" {
		t.Fatalf("next action=%q", out.NextAction.Command)
	}
	if out.FeatureSpec.Summary == "" {
		t.Fatal("expected feature spec summary")
	}
	if len(out.Acceptance) == 0 {
		t.Fatal("expected acceptance hints")
	}
}

func TestLoadHandoffMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadHandoff(root); err == nil {
		t.Fatal("expected missing handoff error")
	}
}

func TestHandoffPath(t *testing.T) {
	root := t.TempDir()
	path := HandoffPath(root)
	if path == "" {
		t.Fatal("empty handoff path")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected missing handoff file")
	}
}
