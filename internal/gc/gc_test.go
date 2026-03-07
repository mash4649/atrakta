package gc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTmpAutoGCApplyWhenThresholdExceeded(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".tmp", "go-build"), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := strings.Repeat("x", 2048)
	if err := os.WriteFile(filepath.Join(repo, ".tmp", "go-build", "a.bin"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.TmpMaxBytes = 256
	cfg.TmpTargetRatioPercent = 50
	cfg.TmpRetentionDays = 1
	cfg.AutoMinIntervalMinutes = 0

	rep, err := Run(Request{
		RepoRoot: repo,
		Scopes:   map[string]bool{"tmp": true},
		Apply:    true,
		Auto:     true,
	}, cfg)
	if err != nil {
		t.Fatalf("gc run failed: %v", err)
	}
	if !rep.Tmp.Triggered {
		t.Fatalf("expected tmp threshold trigger")
	}
	if rep.Tmp.AppliedDeleteBytes == 0 {
		t.Fatalf("expected deletion bytes > 0")
	}
	if _, err := os.Stat(filepath.Join(repo, ".tmp", "go-build", "a.bin")); !os.IsNotExist(err) {
		t.Fatalf("expected tmp file to be deleted")
	}
}

func TestTmpDryRunLeavesFiles(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".tmp", "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".tmp", "dist", "artifact"), []byte(strings.Repeat("x", 1024)), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.TmpMaxBytes = 256
	cfg.AutoMinIntervalMinutes = 0

	rep, err := Run(Request{
		RepoRoot: repo,
		Scopes:   map[string]bool{"tmp": true},
		Apply:    false,
		Auto:     false,
	}, cfg)
	if err != nil {
		t.Fatalf("gc dry run failed: %v", err)
	}
	if len(rep.Tmp.DryRunDelete) == 0 {
		t.Fatalf("expected dry-run delete candidates")
	}
	if len(rep.Tmp.AppliedDelete) != 0 {
		t.Fatalf("expected no applied deletions on dry-run")
	}
	if _, err := os.Stat(filepath.Join(repo, ".tmp", "dist", "artifact")); err != nil {
		t.Fatalf("expected file to remain on dry-run")
	}
}

func TestEventsProposalOnly(t *testing.T) {
	repo := t.TempDir()
	eventsPath := filepath.Join(repo, ".atrakta", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(eventsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventsPath, []byte(strings.Repeat("x", 4096)), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.EventsProposalBytes = 512
	cfg.AutoMinIntervalMinutes = 0

	rep, err := Run(Request{
		RepoRoot: repo,
		Scopes:   map[string]bool{"events": true},
		Apply:    true,
		Auto:     false,
	}, cfg)
	if err != nil {
		t.Fatalf("events proposal run failed: %v", err)
	}
	if !rep.Events.ProposalOnly {
		t.Fatalf("events scope must remain proposal-only")
	}
	if len(rep.Events.Proposals) == 0 {
		t.Fatalf("expected proposal message for oversized events")
	}
}

func TestShouldRunAutoHonorsInterval(t *testing.T) {
	repo := t.TempDir()
	now := time.Now().UTC()
	if err := saveRuntimeState(repo, runtimeState{
		V:          stateVersion,
		UpdatedAt:  now.Format(time.RFC3339Nano),
		LastAutoAt: now.Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	cfg := DefaultConfig()
	cfg.AutoMinIntervalMinutes = 60
	if ShouldRunAuto(repo, cfg) {
		t.Fatalf("expected auto run to be skipped by interval")
	}
	cfg.AutoMinIntervalMinutes = 0
	if !ShouldRunAuto(repo, cfg) {
		t.Fatalf("expected auto run to be allowed when interval disabled")
	}
}
