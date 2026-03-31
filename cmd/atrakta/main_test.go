package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func TestWriteArtifact(t *testing.T) {
	dir := t.TempDir()
	payload := map[string]string{"mode": "inspect"}
	if err := writeArtifact(dir, "inspect.bundle.json", payload); err != nil {
		t.Fatalf("writeArtifact: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "inspect.bundle.json"))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}
	if got["mode"] != "inspect" {
		t.Fatalf("artifact mode=%q", got["mode"])
	}
}

func TestRunExportSnapshots(t *testing.T) {
	dir := t.TempDir()
	if err := runExportSnapshots([]string{"--dir", dir}); err != nil {
		t.Fatalf("runExportSnapshots: %v", err)
	}

	required := []string{
		"onboarding.proposal.json",
		"inspect.onboard.bundle.json",
		"preview.onboard.bundle.json",
		"simulate.onboard.bundle.json",
		"inspect.bundle.json",
		"preview.bundle.json",
		"simulate.bundle.json",
		"fixtures.report.json",
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing snapshot %s: %v", name, err)
		}
	}
}

func TestRunVerifyCoverage(t *testing.T) {
	if err := runVerifyCoverage(nil); err != nil {
		t.Fatalf("runVerifyCoverage: %v", err)
	}
}

func TestRunOnboard(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	artifactDir := t.TempDir()
	if err := runOnboard([]string{"--project-root", projectRoot, "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runOnboard: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "onboarding.proposal.json")); err != nil {
		t.Fatalf("missing onboarding artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "acceptance-spec.generated.json")); err != nil {
		t.Fatalf("missing acceptance spec artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "acceptance-rubric.generated.json")); err != nil {
		t.Fatalf("missing acceptance rubric artifact: %v", err)
	}
}

func TestApplyOnboardingFailure(t *testing.T) {
	input := pipeline.DefaultInput("inspect")
	bundle := onboarding.ProposalBundle{
		Conflicts: []string{"possible duplicate guidance"},
		InferredFailure: onboarding.FailurePreview{
			FailureClass: "legacy_conflict_failure",
			Scope:        "workspace",
			Triggers:     []string{"instruction_conflict", "policy_ambiguity"},
		},
	}
	got := applyOnboardingFailure(input, bundle)
	if got.FailureClass != "legacy_conflict_failure" {
		t.Fatalf("failure class=%q", got.FailureClass)
	}
	if got.FailureContext.Scope != "workspace" {
		t.Fatalf("scope=%q", got.FailureContext.Scope)
	}
	if len(got.FailureContext.Triggers) != 2 {
		t.Fatalf("trigger length=%d", len(got.FailureContext.Triggers))
	}
	if got.FailureContext.IsDiagnosticsOnly {
		t.Fatalf("expected diagnostics false when conflicts exist")
	}
}

func TestRunModeWithOnboardRoot(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}

	artifactDir := t.TempDir()
	if err := runMode("inspect", []string{"--onboard-root", projectRoot, "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMode with onboard root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "inspect.bundle.json")); err != nil {
		t.Fatalf("missing inspect artifact: %v", err)
	}
}

func TestRunAccept(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	artifactDir := t.TempDir()
	if err := runAccept([]string{"--project-root", projectRoot, "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runAccept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta/state/onboarding-state.json")); err != nil {
		t.Fatalf("missing onboarding state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "accept.result.json")); err != nil {
		t.Fatalf("missing accept artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "acceptance-spec.generated.json")); err != nil {
		t.Fatalf("missing acceptance spec artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "acceptance-rubric.generated.json")); err != nil {
		t.Fatalf("missing acceptance rubric artifact: %v", err)
	}
}

func TestRunHarnessProfile(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	artifactDir := t.TempDir()
	if err := runHarnessProfile([]string{"--project-root", projectRoot, "--model-generation", "gpt-5.4", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runHarnessProfile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "harness-profile.report.json")); err != nil {
		t.Fatalf("missing harness profile artifact: %v", err)
	}
}

func TestRunHarnessProfileSelectiveOrchestrationPolicy(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta", "generated"), 0o755); err != nil {
		t.Fatalf("mkdir generated: %v", err)
	}

	bundle := onboarding.ProposalBundle{
		DetectedAssets: []string{"AGENTS.md", "docs"},
		DetectedRisks:  []string{},
		InferredMode:   onboarding.ModeBrownfield,
		InferredManagedScope: map[string]any{
			".atrakta/generated/**": "managed_block",
		},
		InferredCapabilities: []string{"inspect_repo", "propose_repair"},
		InferredGuidance:     map[string]any{"canonical_policy": "authoritative_constraint"},
		InferredDefaultPolicy: map[string]any{
			"read_only": "allow",
		},
		InferredFailure: onboarding.FailurePreview{
			FailureClass:      "legacy_conflict_failure",
			Scope:             "workspace",
			Triggers:          []string{"instruction_conflict"},
			DefaultTier:       "DEGRADE_TO_STRICT",
			ResolvedTier:      "DEGRADE_TO_STRICT",
			StrictTransition:  "strict",
			ExecutionAllowed:  false,
			ProjectionAllowed: true,
			NextAllowedAction: "inspect",
		},
		Conflicts:            []string{"possible duplicate guidance"},
		SuggestedNextActions: []string{"review conflicts", "inspect details"},
	}
	spec, rubric := onboarding.BuildAcceptanceArtifacts(bundle)
	if err := writeArtifact(filepath.Join(projectRoot, ".atrakta", "generated"), "acceptance-spec.generated.json", spec); err != nil {
		t.Fatalf("write acceptance spec: %v", err)
	}
	if err := writeArtifact(filepath.Join(projectRoot, ".atrakta", "generated"), "acceptance-rubric.generated.json", rubric); err != nil {
		t.Fatalf("write acceptance rubric: %v", err)
	}

	artifactDir := t.TempDir()
	if err := runHarnessProfile([]string{"--project-root", projectRoot, "--model-generation", "gpt-5.4", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runHarnessProfile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "harness-profile.report.json")); err != nil {
		t.Fatalf("missing harness profile artifact: %v", err)
	}
	policyPath := filepath.Join(artifactDir, selectiveOrchestrationPolicyArtifact)
	data, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("missing selective orchestration policy artifact: %v", err)
	}
	var report struct {
		PlannerEnabled    bool `json:"planner_enabled"`
		EvaluatorEnabled  bool `json:"evaluator_enabled"`
		CheckpointEnabled bool `json:"checkpoint_enabled"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode selective orchestration policy: %v", err)
	}
	if !report.PlannerEnabled || !report.EvaluatorEnabled {
		t.Fatalf("expected planner and evaluator enabled: %+v", report)
	}
	if report.CheckpointEnabled {
		t.Fatalf("expected checkpoint to be retired for next generation: %+v", report)
	}
}

func TestRunBenchmarkStartLatency(t *testing.T) {
	artifactDir := t.TempDir()
	if err := runBenchmark([]string{"start-latency", "--iterations", "1", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runBenchmark: %v", err)
	}
	path := filepath.Join(artifactDir, "start-latency.report.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read benchmark artifact: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode benchmark artifact: %v", err)
	}
	if got := int(report["iterations"].(float64)); got != 1 {
		t.Fatalf("iterations=%d", got)
	}
	if got := int(report["fast_path_hits"].(float64)); got != 1 {
		t.Fatalf("fast_path_hits=%d", got)
	}
}

func TestRunBenchmarkStartLatencyMonorepo(t *testing.T) {
	artifactDir := t.TempDir()
	if err := runBenchmark([]string{"start-latency", "--iterations", "1", "--workspace", "monorepo", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runBenchmark monorepo: %v", err)
	}
	path := filepath.Join(artifactDir, "start-latency.report.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read benchmark artifact: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode benchmark artifact: %v", err)
	}
	if got := report["scenario"].(string); got != "start_large_repo" {
		t.Fatalf("scenario=%q", got)
	}
	if got := int(report["iterations"].(float64)); got != 1 {
		t.Fatalf("iterations=%d", got)
	}
}

func TestRunMutatePhases(t *testing.T) {
	projectRoot := t.TempDir()
	artifactDir := t.TempDir()

	if err := runMutate([]string{"inspect", "--target", ".atrakta/generated/x.json", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMutate inspect: %v", err)
	}
	if err := runMutate([]string{"propose", "--target", ".atrakta/generated/x.json", "--content", "{\"x\":1}", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMutate propose: %v", err)
	}
	if err := runMutate([]string{"apply", "--project-root", projectRoot, "--target", ".atrakta/generated/x.json", "--content", "{\"x\":1}\n", "--allow", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMutate apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta/generated/x.json")); err != nil {
		t.Fatalf("missing applied mutation target: %v", err)
	}
}

func TestRunAuditAppendAndVerify(t *testing.T) {
	projectRoot := t.TempDir()
	if err := runAudit([]string{"append", "--project-root", projectRoot, "--level", "A2", "--action", "test_append"}); err != nil {
		t.Fatalf("runAudit append: %v", err)
	}
	if err := runAudit([]string{"verify", "--project-root", projectRoot, "--level", "A2"}); err != nil {
		t.Fatalf("runAudit verify: %v", err)
	}
}

func TestRunGCDryRunTmp(t *testing.T) {
	projectRoot := t.TempDir()
	tmpDir := filepath.Join(projectRoot, ".atrakta", "runtime")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "cache"), 0o755); err != nil {
		t.Fatalf("mkdir runtime cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "cache", "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write runtime file: %v", err)
	}

	if err := runGC([]string{"--project-root", projectRoot, "--scope", "tmp", "--json"}); err != nil {
		t.Fatalf("runGC dry-run: %v", err)
	}
	if _, err := os.Stat(tmpDir); err != nil {
		t.Fatalf("runtime dir should remain in dry-run: %v", err)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventGCRun) {
		t.Fatalf("missing %s in %v", runEventGCRun, types)
	}
}

func TestRunGCApplyTmp(t *testing.T) {
	projectRoot := t.TempDir()
	tmpDir := filepath.Join(projectRoot, ".atrakta", "runtime")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write runtime file: %v", err)
	}

	if err := runGC([]string{"--project-root", projectRoot, "--scope", "tmp", "--apply", "--json"}); err != nil {
		t.Fatalf("runGC apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "x.txt")); !os.IsNotExist(err) {
		t.Fatalf("runtime file should be removed when apply=true")
	}
}

func TestRunGCDryRunEvents(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta", "audit"), 0o755); err != nil {
		t.Fatalf("mkdir audit root: %v", err)
	}

	if _, err := appendRunEventForTest(projectRoot, "old", time.Now().UTC().AddDate(0, 0, -40)); err != nil {
		t.Fatalf("append old event: %v", err)
	}
	if _, err := appendRunEventForTest(projectRoot, "new", time.Now().UTC()); err != nil {
		t.Fatalf("append new event: %v", err)
	}

	if err := runGC([]string{"--project-root", projectRoot, "--scope", "events", "--retention-days", "30", "--json"}); err != nil {
		t.Fatalf("runGC events dry-run: %v", err)
	}
	eventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, eventsPath)
	if len(types) != 3 {
		t.Fatalf("dry-run should preserve stream and append gc event, got %v", types)
	}
	if !containsString(types, "old") || !containsString(types, "new") || !containsString(types, runEventGCRun) {
		t.Fatalf("unexpected event types: %v", types)
	}
}

func TestRunGCApplyEvents(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta", "audit"), 0o755); err != nil {
		t.Fatalf("mkdir audit root: %v", err)
	}

	if _, err := appendRunEventForTest(projectRoot, "old", time.Now().UTC().AddDate(0, 0, -40)); err != nil {
		t.Fatalf("append old event: %v", err)
	}
	if _, err := appendRunEventForTest(projectRoot, "new", time.Now().UTC()); err != nil {
		t.Fatalf("append new event: %v", err)
	}

	if err := runGC([]string{"--project-root", projectRoot, "--scope", "events", "--retention-days", "30", "--apply", "--json"}); err != nil {
		t.Fatalf("runGC events apply: %v", err)
	}
	eventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, eventsPath)
	if len(types) != 2 {
		t.Fatalf("expected pruned stream plus gc event, got %v", types)
	}
	if !containsString(types, "new") || !containsString(types, runEventGCRun) {
		t.Fatalf("unexpected event types after apply: %v", types)
	}
	if err := audit.VerifyRunEventsIntegrity(filepath.Join(projectRoot, ".atrakta", "audit"), audit.LevelA2); err != nil {
		t.Fatal(err)
	}
}

func TestRunMigrateCheckCompatible(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta"), 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta", "audit", "events"), 0o755); err != nil {
		t.Fatalf("mkdir events dir: %v", err)
	}
	writeSessionVersionFile(t, filepath.Join(projectRoot, ".atrakta", "state.json"), "session-state.v1")
	writeSessionVersionFile(t, filepath.Join(projectRoot, ".atrakta", "progress.json"), "session-progress.v1")
	writeSessionVersionFile(t, filepath.Join(projectRoot, ".atrakta", "task-graph.json"), "session-task-graph.v1")
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	runEventsBefore := []byte("{\"schema_version\":1,\"seq\":1,\"timestamp\":\"2026-03-28T00:00:00Z\",\"event_type\":\"migrate.check\",\"integrity_level\":\"A0\",\"payload\":{}}\n")
	if err := os.WriteFile(runEventsPath, runEventsBefore, 0o644); err != nil {
		t.Fatalf("write run-events: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runMigrate([]string{"check", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runMigrate check: %v", err)
		}
	})
	var out migrateCheckResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal migrate output: %v", err)
	}
	if !out.OK {
		t.Fatalf("expected compatible migrate check, got ok=false: %#v", out)
	}
	for _, check := range out.Checks {
		if check.Status != "compatible" {
			t.Fatalf("expected compatible check, got %#v", check)
		}
	}
	after, err := os.ReadFile(runEventsPath)
	if err != nil {
		t.Fatalf("read run-events after check: %v", err)
	}
	if string(after) != string(runEventsBefore) {
		t.Fatalf("migrate check should be read-only, got diff:\nbefore=%s\nafter=%s", string(runEventsBefore), string(after))
	}
}

func TestRunMigrateCheckNeedsMigration(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta", "audit", "events"), 0o755); err != nil {
		t.Fatalf("mkdir events dir: %v", err)
	}
	writeSessionVersionFile(t, filepath.Join(projectRoot, ".atrakta", "state.json"), "session-state.v0")
	writeSessionVersionFile(t, filepath.Join(projectRoot, ".atrakta", "progress.json"), "session-progress.v0")
	writeSessionVersionFile(t, filepath.Join(projectRoot, ".atrakta", "task-graph.json"), "session-task-graph.v0")
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	runEventsBefore := []byte("{\"schema_version\":2,\"seq\":1,\"timestamp\":\"2026-03-28T00:00:00Z\",\"event_type\":\"legacy.event\",\"integrity_level\":\"A0\",\"payload\":{}}\n")
	if err := os.WriteFile(runEventsPath, runEventsBefore, 0o644); err != nil {
		t.Fatalf("write run-events: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runMigrate([]string{"check", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runMigrate check: %v", err)
		}
	})
	var out migrateCheckResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal migrate output: %v", err)
	}
	if out.OK {
		t.Fatalf("expected migration needed, got ok=true: %#v", out)
	}
	needs := 0
	for _, check := range out.Checks {
		if check.Status == "needs_migration" {
			needs++
			if len(check.Guidance) == 0 {
				t.Fatalf("needs_migration guidance missing: %#v", check)
			}
		}
	}
	if needs == 0 {
		t.Fatalf("expected at least one needs_migration check: %#v", out)
	}
	if len(out.Guidance) == 0 {
		t.Fatalf("expected top-level guidance: %#v", out)
	}
	after, err := os.ReadFile(runEventsPath)
	if err != nil {
		t.Fatalf("read run-events after check: %v", err)
	}
	if string(after) != string(runEventsBefore) {
		t.Fatalf("migrate check should be read-only, got diff:\nbefore=%s\nafter=%s", string(runEventsBefore), string(after))
	}
}

func TestRunMigrateCheckUnknown(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta"), 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".atrakta", "state.json"), []byte("{\"schema_version\":\"unexpected\"}\n"), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runMigrate([]string{"check", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runMigrate check: %v", err)
		}
	})
	var out migrateCheckResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal migrate output: %v", err)
	}
	unknown := 0
	for _, check := range out.Checks {
		if check.Status == "unknown" {
			unknown++
		}
	}
	if unknown == 0 {
		t.Fatalf("expected at least one unknown check: %#v", out)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	if _, err := os.Stat(runEventsPath); !os.IsNotExist(err) {
		t.Fatalf("migrate check should be read-only, got run-events log: %v", err)
	}
}

func TestRunProjectionRenderDryRun(t *testing.T) {
	projectRoot := t.TempDir()
	raw := captureStdout(t, func() {
		if err := runProjection([]string{"render", "--project-root", projectRoot, "--dry-run", "--json"}); err != nil {
			t.Fatalf("runProjection render dry-run: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal projection dry-run output: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["written"] != false {
		t.Fatalf("written=%v", out["written"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md should not be written during dry-run")
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventProjectionRendered) {
		t.Fatalf("missing %s in %v", runEventProjectionRendered, types)
	}
}

func TestRunProjectionRenderApply(t *testing.T) {
	projectRoot := t.TempDir()
	raw := captureStdout(t, func() {
		if err := runProjection([]string{"render", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
			t.Fatalf("runProjection render apply: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal projection apply output: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["written"] != true {
		t.Fatalf("written=%v", out["written"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "AGENTS.md")); err != nil {
		t.Fatalf("missing AGENTS.md: %v", err)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventProjectionRendered) {
		t.Fatalf("missing %s in %v", runEventProjectionRendered, types)
	}
}

func writeSessionVersionFile(t *testing.T, path, schemaVersion string) {
	t.Helper()
	payload := map[string]any{
		"schema_version": schemaVersion,
		"command":        "migrate.check",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func appendRunEventForTest(projectRoot, eventType string, ts time.Time) (audit.RunEvent, error) {
	ev, err := audit.AppendRunEventAndVerify(filepath.Join(projectRoot, ".atrakta", "audit"), audit.LevelA2, eventType, map[string]any{
		"event_type": eventType,
	}, audit.RunEventOptions{Actor: "kernel"})
	if err != nil {
		return audit.RunEvent{}, err
	}
	if err := overwriteRunEventTimestamp(filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl"), ev.Seq, ts.UTC().Format(time.RFC3339)); err != nil {
		return audit.RunEvent{}, err
	}
	return ev, nil
}

func overwriteRunEventTimestamp(path string, seq int, timestamp string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return err
		}
		if seqValue, ok := ev["seq"].(float64); !ok || int(seqValue) != seq {
			continue
		}
		ev["timestamp"] = timestamp
		raw, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		lines[i] = string(raw)
		break
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func TestRunAliasAndExtensions(t *testing.T) {
	if err := runAlias("doctor", []string{"--execute"}); err != nil {
		t.Fatalf("runAlias doctor: %v", err)
	}

	projectRoot := t.TempDir()
	manifestDir := filepath.Join(projectRoot, "extensions", "manifests")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifests: %v", err)
	}
	manifest := `{"name":"default","items":[{"id":"policy-1","kind":"policy","enabled":true}]}`
	if err := os.WriteFile(filepath.Join(manifestDir, "default.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := runExtensions([]string{"--project-root", projectRoot}); err != nil {
		t.Fatalf("runExtensions: %v", err)
	}
}

func TestRunCommandOnboardingNeedsApprovalNonInteractive(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--non-interactive", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")); err == nil {
		t.Fatalf("onboarding state should not be written without approval")
	}
}

func TestRunCommandOnboardingApproveFlag(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--non-interactive", "--approve", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")); err != nil {
		t.Fatalf("expected onboarding state written: %v", err)
	}
}

func TestRunCommandNeedsInputWhenCanonicalPresentAndInterfaceUnknown(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitNeedsInput {
		t.Fatalf("exit code=%d", code)
	}
}

func TestRunCommandNormalPathWithExplicitInterface(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "audit", "events", "install-events.jsonl")); err != nil {
		t.Fatalf("missing run audit event log: %v", err)
	}
}

func TestRunCommandNeedsInputWhenContractMissing(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("runCommand: %v", err)
		}
		if code != exitNeedsInput {
			t.Fatalf("exit code=%d", code)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out["status"] != "needs_input" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["portability_reason"] != "machine contract missing or invalid" {
		t.Fatalf("portability_reason=%v", out["portability_reason"])
	}
}

func TestRunCommandNormalPathFailsWhenAuditIntegrityInvalid(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	auditDir := filepath.Join(projectRoot, ".atrakta", "audit", "events")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.MkdirAll(auditDir, 0o755); err != nil {
		t.Fatalf("mkdir audit dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "install-events.jsonl"), []byte("not-json\n"), 0o644); err != nil {
		t.Fatalf("write invalid audit log: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
	if err == nil {
		t.Fatal("expected audit preflight verify error")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
}

func TestStartCommandNormalPathFailsWhenRunEventsIntegrityInvalid(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	// Seed one valid start run so run-events exists.
	captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand seed: %v", err)
		}
		if code != exitOK {
			t.Fatalf("seed exit code=%d", code)
		}
	})

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	if err := os.WriteFile(runEventsPath, []byte("not-json\n"), 0o644); err != nil {
		t.Fatalf("tamper run-events: %v", err)
	}

	code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
	if err == nil {
		t.Fatal("expected run-events preflight verify error")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
}

func TestStartCommandOnboardingNeedsApprovalNonInteractive(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := startCommand([]string{"--project-root", projectRoot, "--non-interactive", "--json"})
	if err != nil {
		t.Fatalf("startCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")); err == nil {
		t.Fatalf("onboarding state should not be written without approval")
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, "gate.result") {
		t.Fatalf("missing gate.result in %v", types)
	}
}

func TestStartCommandJsonOutputValidatesAgainstRunSchema(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	raw := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})
	if err := validation.ValidateStartOutputRaw(raw); err != nil {
		t.Fatalf("validate start output: %v", err)
	}
}

func TestStartCommandPartialStateReturnsDiagnosticEnvelope(t *testing.T) {
	projectRoot := t.TempDir()
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--json"})
		if code != exitRuntimeError {
			t.Fatalf("exit code=%d", code)
		}
		if err == nil {
			t.Fatal("expected diagnostic error")
		}
		if !strings.Contains(err.Error(), "partial_state") {
			t.Fatalf("error=%v", err)
		}
	})
	if err := validation.ValidateStartOutputRaw(raw); err != nil {
		t.Fatalf("validate start output: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out["canonical_state"] != runpkg.StatePartialState {
		t.Fatalf("canonical_state=%v", out["canonical_state"])
	}
	if msg, _ := out["message"].(string); !strings.Contains(msg, ".atrakta/state/onboarding-state.json") {
		t.Fatalf("message=%q", msg)
	}
	if next, _ := out["next_allowed_action"].(string); !strings.Contains(next, ".atrakta/canonical/policies/registry/index.json") {
		t.Fatalf("next_allowed_action=%q", next)
	}
	if reqs, ok := out["required_inputs"].([]any); !ok || len(reqs) == 0 {
		t.Fatalf("required_inputs=%v", out["required_inputs"])
	}
	if iface, ok := out["interface"].(map[string]any); !ok || iface["source"] == "" {
		t.Fatalf("interface=%v", out["interface"])
	}
}

func TestStartCommandCorruptStateReturnsDiagnosticEnvelope(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".atrakta", "canonical"), 0o755); err != nil {
		t.Fatalf("mkdir canonical dir: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--json"})
		if code != exitRuntimeError {
			t.Fatalf("exit code=%d", code)
		}
		if err == nil {
			t.Fatal("expected diagnostic error")
		}
		if !strings.Contains(err.Error(), "corrupt_state") {
			t.Fatalf("error=%v", err)
		}
	})
	if err := validation.ValidateStartOutputRaw(raw); err != nil {
		t.Fatalf("validate start output: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out["canonical_state"] != runpkg.StateCorruptState {
		t.Fatalf("canonical_state=%v", out["canonical_state"])
	}
	if msg, _ := out["message"].(string); !strings.Contains(msg, ".atrakta/canonical/") {
		t.Fatalf("message=%q", msg)
	}
	if next, _ := out["next_allowed_action"].(string); !strings.Contains(next, ".atrakta/canonical/") {
		t.Fatalf("next_allowed_action=%q", next)
	}
	if reqs, ok := out["required_inputs"].([]any); !ok || len(reqs) == 0 {
		t.Fatalf("required_inputs=%v", out["required_inputs"])
	}
}

func TestStartCommandApplyNeedsApprovalNonInteractive(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	raw := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--apply", "--non-interactive", "--json"})
		if err != nil {
			t.Fatalf("startCommand: %v", err)
		}
		if code != exitNeedsApproval {
			t.Fatalf("exit code=%d", code)
		}
	})
	if err := validation.ValidateStartOutputRaw(raw); err != nil {
		t.Fatalf("validate start output: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out["approval_scope"] != "managed_apply" {
		t.Fatalf("approval_scope=%v", out["approval_scope"])
	}
	if next, _ := out["next_allowed_action"].(string); next == "" {
		t.Fatal("next_allowed_action missing")
	}
}

func TestInitCommandDelegatesToStartAndEmitsLifecycleEvents(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := initCommand([]string{"--project-root", projectRoot, "--approve", "--json"})
	if err != nil {
		t.Fatalf("initCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	for _, required := range []string{runEventInitBegin, runEventInitStep, runEventInitEnd, runEventWrapInstall, runEventHookInstall, runEventIDEAutostartInstall, "start.begin"} {
		if !containsString(types, required) {
			t.Fatalf("missing run-event type %q in %v", required, types)
		}
	}
	if stepCount := countString(types, runEventInitStep); stepCount != 3 {
		t.Fatalf("expected 3 init.step events, got %d in %v", stepCount, types)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")); err != nil {
		t.Fatalf("missing wrap script: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("missing hook script: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".vscode", "tasks.json")); err != nil {
		t.Fatalf("missing vscode tasks: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".cursor", "autostart.json")); err != nil {
		t.Fatalf("missing cursor autostart: %v", err)
	}
}

func TestInitCommandSkipsIntegrationStepsWithNoFlags(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := initCommand([]string{
		"--project-root", projectRoot,
		"--non-interactive",
		"--json",
		"--no-wrap",
		"--no-hook",
		"--no-ide-autostart",
	})
	if err != nil {
		t.Fatalf("initCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventInitBegin) || !containsString(types, runEventInitEnd) {
		t.Fatalf("missing init begin/end in %v", types)
	}
	stepCount := 0
	for _, typ := range types {
		if typ == runEventInitStep {
			stepCount++
		}
	}
	if stepCount != 0 {
		t.Fatalf("expected no init.step events, got %d in %v", stepCount, types)
	}
}

func TestStartCommandNeedsInputWhenCanonicalPresentAndInterfaceUnknown(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("startCommand: %v", err)
		}
		if code != exitNeedsInput {
			t.Fatalf("exit code=%d", code)
		}
	})
	if err := validation.ValidateStartOutputRaw(raw); err != nil {
		t.Fatalf("validate start output: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out["required_inputs"] == nil {
		t.Fatal("required_inputs missing")
	}
	if next, _ := out["next_allowed_action"].(string); next == "" {
		t.Fatal("next_allowed_action missing")
	}
}

func TestStartCommandNormalPathWithExplicitInterface(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	var code int
	captureStdout(t, func() {
		var err error
		code, err = startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand: %v", err)
		}
	})
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "audit", "events", "install-events.jsonl")); err != nil {
		t.Fatalf("missing run audit event log: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")); err != nil {
		t.Fatalf("missing run-events log: %v", err)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	for _, required := range []string{"start.begin", "detect.performed", "plan.created", "start.end"} {
		if !containsString(types, required) {
			t.Fatalf("missing run-event type %q in %v", required, types)
		}
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "runtime", "auto-state.v1.json")); err != nil {
		t.Fatalf("missing auto-state snapshot: %v", err)
	}
	handoff, err := runpkg.LoadHandoff(projectRoot)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	if handoff.Command != "start" {
		t.Fatalf("handoff command=%q", handoff.Command)
	}
	if handoff.InterfaceID != "generic-cli" {
		t.Fatalf("handoff interface=%q", handoff.InterfaceID)
	}
	if handoff.NextAllowedAction != "inspect" {
		t.Fatalf("handoff next action=%q", handoff.NextAllowedAction)
	}
	if handoff.NextAction.Command != "inspect" {
		t.Fatalf("handoff next action command=%q", handoff.NextAction.Command)
	}
	if handoff.FeatureSpec.Summary == "" {
		t.Fatal("handoff missing feature summary")
	}
	if len(handoff.FeatureSpec.ResolvedTargets) == 0 {
		t.Fatal("handoff missing resolved targets")
	}
	if len(handoff.Acceptance) == 0 {
		t.Fatal("handoff missing acceptance hints")
	}
	if handoff.NextAction.Hint == "" {
		t.Fatal("handoff missing next action hint")
	}
	if handoff.Checkpoint.AutoStatePath == "" {
		t.Fatal("handoff missing auto-state checkpoint")
	}
	if handoff.Checkpoint.StartFastPath == "" {
		t.Fatal("handoff missing start-fast checkpoint")
	}
	if handoff.Checkpoint.StatePath == "" {
		t.Fatal("handoff missing session state checkpoint")
	}
	if handoff.Checkpoint.ProgressPath == "" {
		t.Fatal("handoff missing session progress checkpoint")
	}
	if handoff.Checkpoint.TaskGraphPath == "" {
		t.Fatal("handoff missing session task-graph checkpoint")
	}
	if handoff.Checkpoint.OnboardingState == "" {
		t.Fatal("handoff missing onboarding-state checkpoint")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state.json")); err != nil {
		t.Fatalf("missing session state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "progress.json")); err != nil {
		t.Fatalf("missing session progress: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "task-graph.json")); err != nil {
		t.Fatalf("missing session task graph: %v", err)
	}
}

func TestStartCommandNormalPathUsesAutoStateWhenInterfaceOmitted(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	var seedCode int
	captureStdout(t, func() {
		var err error
		seedCode, err = startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand seed: %v", err)
		}
	})
	if seedCode != exitOK {
		t.Fatalf("seed exit code=%d", seedCode)
	}

	if err := os.MkdirAll(filepath.Join(projectRoot, "tests"), 0o755); err != nil {
		t.Fatalf("mkdir tests dir: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("startCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	iface, ok := out["interface"].(map[string]any)
	if !ok {
		t.Fatalf("interface payload missing: %#v", out["interface"])
	}
	if iface["interface_id"] != "generic-cli" {
		t.Fatalf("interface_id=%v", iface["interface_id"])
	}
	source, _ := iface["source"].(string)
	if !strings.HasPrefix(source, "auto") {
		t.Fatalf("interface.source=%q", source)
	}
}

func TestStartCommandWritesStartArtifact(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	artifactDir := t.TempDir()

	code, err := startCommand([]string{"--project-root", projectRoot, "--non-interactive", "--json", "--artifact-dir", artifactDir})
	if err != nil {
		t.Fatalf("startCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "start.result.json")); err != nil {
		t.Fatalf("missing start artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "run.result.json")); err == nil {
		t.Fatalf("unexpected run artifact for start command")
	}
}

func TestStartCommandUsesFastPathOnSecondRun(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	rawFirst := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand first: %v", err)
		}
		if code != exitOK {
			t.Fatalf("first exit code=%d", code)
		}
	})
	var first map[string]any
	if err := json.Unmarshal(rawFirst, &first); err != nil {
		t.Fatalf("unmarshal first output: %v", err)
	}
	if first["status"] != "ok" {
		t.Fatalf("first status=%v", first["status"])
	}

	snapshotPath := filepath.Join(projectRoot, ".atrakta", "runtime", "start-fast.v1.json")
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("missing snapshot: %v", err)
	}
	policyBefore, err := os.Stat(filepath.Join(policyDir, "index.json"))
	if err != nil {
		t.Fatalf("stat policy index: %v", err)
	}
	stateBefore, err := os.Stat(filepath.Join(stateDir, "onboarding-state.json"))
	if err != nil {
		t.Fatalf("stat onboarding state: %v", err)
	}
	auditPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "install-events.jsonl")
	auditBefore := countLines(t, auditPath)
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	runEventsBefore := countLines(t, runEventsPath)
	sessionStatePath := filepath.Join(projectRoot, ".atrakta", "state.json")
	progressPath := filepath.Join(projectRoot, ".atrakta", "progress.json")
	taskGraphPath := filepath.Join(projectRoot, ".atrakta", "task-graph.json")
	sessionStateBefore, err := os.Stat(sessionStatePath)
	if err != nil {
		t.Fatalf("stat session state: %v", err)
	}
	progressBefore, err := os.Stat(progressPath)
	if err != nil {
		t.Fatalf("stat progress: %v", err)
	}
	taskGraphBefore, err := os.Stat(taskGraphPath)
	if err != nil {
		t.Fatalf("stat task graph: %v", err)
	}

	rawSecond := captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand second: %v", err)
		}
		if code != exitOK {
			t.Fatalf("second exit code=%d", code)
		}
	})
	var second map[string]any
	if err := json.Unmarshal(rawSecond, &second); err != nil {
		t.Fatalf("unmarshal second output: %v", err)
	}
	msg, _ := second["message"].(string)
	if !strings.Contains(msg, "fast-path") {
		t.Fatalf("expected fast-path message, got: %q", msg)
	}
	summary, _ := second["canonical_summary"].(map[string]any)
	if summary["fast_path"] != true {
		t.Fatalf("canonical_summary.fast_path=%v", summary["fast_path"])
	}
	auditAfter := countLines(t, auditPath)
	if auditAfter != auditBefore+1 {
		t.Fatalf("audit lines before=%d after=%d", auditBefore, auditAfter)
	}
	runEventsAfter := countLines(t, runEventsPath)
	if runEventsAfter != runEventsBefore+2 {
		t.Fatalf("run-events lines before=%d after=%d", runEventsBefore, runEventsAfter)
	}
	policyAfter, err := os.Stat(filepath.Join(policyDir, "index.json"))
	if err != nil {
		t.Fatalf("stat policy index after: %v", err)
	}
	stateAfter, err := os.Stat(filepath.Join(stateDir, "onboarding-state.json"))
	if err != nil {
		t.Fatalf("stat onboarding state after: %v", err)
	}
	if !policyAfter.ModTime().Equal(policyBefore.ModTime()) {
		t.Fatalf("canonical index was modified on fast path")
	}
	if !stateAfter.ModTime().Equal(stateBefore.ModTime()) {
		t.Fatalf("state file was modified on fast path")
	}
	sessionStateAfter, err := os.Stat(sessionStatePath)
	if err != nil {
		t.Fatalf("stat session state after: %v", err)
	}
	progressAfter, err := os.Stat(progressPath)
	if err != nil {
		t.Fatalf("stat progress after: %v", err)
	}
	taskGraphAfter, err := os.Stat(taskGraphPath)
	if err != nil {
		t.Fatalf("stat task graph after: %v", err)
	}
	if !sessionStateAfter.ModTime().Equal(sessionStateBefore.ModTime()) {
		t.Fatalf("session state was modified on fast path")
	}
	if !progressAfter.ModTime().Equal(progressBefore.ModTime()) {
		t.Fatalf("session progress was modified on fast path")
	}
	if !taskGraphAfter.ModTime().Equal(taskGraphBefore.ModTime()) {
		t.Fatalf("session task graph was modified on fast path")
	}
	handoff, err := runpkg.LoadHandoff(projectRoot)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	if !handoff.FastPath {
		t.Fatal("expected fast-path handoff")
	}
	if handoff.NextAllowedAction != "inspect" {
		t.Fatalf("handoff next action=%q", handoff.NextAllowedAction)
	}
	if handoff.FeatureSpec.Summary == "" {
		t.Fatal("handoff missing feature summary")
	}
	if handoff.Checkpoint.StatePath == "" {
		t.Fatal("handoff missing session state checkpoint")
	}
	if handoff.Checkpoint.ProgressPath == "" {
		t.Fatal("handoff missing session progress checkpoint")
	}
	if handoff.Checkpoint.TaskGraphPath == "" {
		t.Fatal("handoff missing session task-graph checkpoint")
	}
}

func TestResumeCommandUsesHandoffWhenAutoStateMissing(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)
	artifactDir := t.TempDir()

	captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand seed: %v", err)
		}
		if code != exitOK {
			t.Fatalf("seed exit code=%d", code)
		}
	})

	if err := os.Remove(filepath.Join(projectRoot, ".atrakta", "runtime", "auto-state.v1.json")); err != nil {
		t.Fatalf("remove auto-state: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := resumeCommand([]string{"--project-root", projectRoot, "--json", "--artifact-dir", artifactDir})
		if err != nil {
			t.Fatalf("resumeCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal resume output: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	iface, ok := out["interface"].(map[string]any)
	if !ok {
		t.Fatalf("interface payload missing: %#v", out["interface"])
	}
	if iface["interface_id"] != "generic-cli" {
		t.Fatalf("interface_id=%v", iface["interface_id"])
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "resume.result.json")); err != nil {
		t.Fatalf("missing resume artifact: %v", err)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventResumeBegin) || !containsString(types, runEventResumeEnd) {
		t.Fatalf("missing resume events in %v", types)
	}
	handoff, err := runpkg.LoadHandoff(projectRoot)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	if handoff.Command != "resume" {
		t.Fatalf("handoff command=%q", handoff.Command)
	}
	if handoff.FeatureSpec.Summary == "" {
		t.Fatal("resume handoff missing feature summary")
	}
	if len(handoff.FeatureSpec.ResolvedTargets) == 0 {
		t.Fatal("resume handoff missing resolved targets")
	}
	if len(handoff.Acceptance) == 0 {
		t.Fatal("resume handoff missing acceptance hints")
	}
	if handoff.NextAction.Hint == "" {
		t.Fatal("resume handoff missing next action hint")
	}
	if handoff.Checkpoint.AutoStatePath == "" {
		t.Fatal("resume handoff missing auto-state checkpoint")
	}
	if handoff.Checkpoint.StartFastPath == "" {
		t.Fatal("resume handoff missing start-fast checkpoint")
	}
	if handoff.Checkpoint.OnboardingState == "" {
		t.Fatal("resume handoff missing onboarding-state checkpoint")
	}
}

func TestResumeCommandUsesAutoStateWhenHandoffMissing(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)
	artifactDir := t.TempDir()

	captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand seed: %v", err)
		}
		if code != exitOK {
			t.Fatalf("seed exit code=%d", code)
		}
	})

	if err := os.Remove(runpkg.HandoffPath(projectRoot)); err != nil {
		t.Fatalf("remove handoff: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := resumeCommand([]string{"--project-root", projectRoot, "--json", "--artifact-dir", artifactDir})
		if err != nil {
			t.Fatalf("resumeCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal resume output: %v", err)
	}
	iface, ok := out["interface"].(map[string]any)
	if !ok {
		t.Fatalf("interface payload missing: %#v", out["interface"])
	}
	if iface["interface_id"] != "generic-cli" {
		t.Fatalf("interface_id=%v", iface["interface_id"])
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "resume.result.json")); err != nil {
		t.Fatalf("missing resume artifact: %v", err)
	}
}

func TestResumeCommandRequiresExistingState(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := resumeCommand([]string{"--project-root", projectRoot, "--json"})
	if err == nil {
		t.Fatal("expected resume error")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
	if err.Error() != errResumeRequiresState.Error() {
		t.Fatalf("error=%v", err)
	}
}

func TestResumeCommandAutoApplyFromHandoffNextAction(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand seed: %v", err)
		}
		if code != exitOK {
			t.Fatalf("seed exit code=%d", code)
		}
	})

	handoff, err := runpkg.LoadHandoff(projectRoot)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	handoff.NextAction.Command = "apply"
	handoff.NextAllowedAction = "apply"
	if err := runpkg.SaveHandoff(projectRoot, handoff); err != nil {
		t.Fatalf("save handoff: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := resumeCommand([]string{"--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("resumeCommand: %v", err)
		}
		if code != exitNeedsApproval {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal resume output: %v", err)
	}
	if out["status"] != "needs_approval" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["approval_scope"] != "managed_apply" {
		t.Fatalf("approval_scope=%v", out["approval_scope"])
	}
}

func TestResumeCommandBlockedByHandoffDeny(t *testing.T) {
	projectRoot := t.TempDir()
	prepareStartReadyProject(t, projectRoot)

	captureStdout(t, func() {
		code, err := startCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
		if err != nil {
			t.Fatalf("startCommand seed: %v", err)
		}
		if code != exitOK {
			t.Fatalf("seed exit code=%d", code)
		}
	})

	handoff, err := runpkg.LoadHandoff(projectRoot)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	handoff.NextAction.Command = "deny"
	handoff.NextAction.Hint = "resolve conflict before resume"
	handoff.NextAllowedAction = "deny"
	if err := runpkg.SaveHandoff(projectRoot, handoff); err != nil {
		t.Fatalf("save handoff: %v", err)
	}

	code, err := resumeCommand([]string{"--project-root", projectRoot, "--json"})
	if err == nil {
		t.Fatal("expected resume error")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
	if !strings.Contains(err.Error(), errResumeBlockedByHandoff.Error()) {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(err.Error(), "resolve conflict before resume") {
		t.Fatalf("error=%v", err)
	}
}

func TestRunCommandApplyNeedsApproval(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--apply", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "audit", "events", "install-events.jsonl")); err != nil {
		t.Fatalf("missing run audit event log: %v", err)
	}
}

func TestRunCommandApplyWithApprove(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--apply", "--approve", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "generated", "repo-map.generated.json")); err != nil {
		t.Fatalf("expected generated projection written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "generated", "capabilities.generated.json")); err != nil {
		t.Fatalf("expected capabilities projection written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "generated", "guidance.generated.json")); err != nil {
		t.Fatalf("expected guidance projection written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")); err != nil {
		t.Fatalf("expected run state written: %v", err)
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventApplyBegin) || !containsString(types, runEventApplyPerformed) {
		t.Fatalf("missing apply events in %v", types)
	}
}

func TestRunCommandApplyBlockedByDegradedPortability(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	raw := captureStdout(t, func() {
		code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "cursor", "--apply", "--approve", "--json"})
		if err != nil {
			t.Fatalf("runCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal run output: %v", err)
	}
	if out["portability_status"] != "degraded" {
		t.Fatalf("portability_status=%v", out["portability_status"])
	}
	if out["next_allowed_action"] != "propose" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")); err == nil {
		t.Fatalf("run-state should not be written when portability is degraded")
	}
}

func TestRunCommandApplyBlockedByUnsupportedPortability(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	raw := captureStdout(t, func() {
		code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "mcp", "--apply", "--approve", "--json"})
		if err != nil {
			t.Fatalf("runCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal run output: %v", err)
	}
	if out["portability_status"] != "unsupported" {
		t.Fatalf("portability_status=%v", out["portability_status"])
	}
	if out["next_allowed_action"] != "propose" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")); err == nil {
		t.Fatalf("run-state should not be written when portability is unsupported")
	}
}

func TestRunCommandInvalidCanonicalIndex(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{not-json}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
	if err == nil {
		t.Fatal("expected canonical parse error")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
}

func TestBuildRunInspectInputFromDetectedAssets(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}

	input, err := buildRunInspectInput(projectRoot, runpkg.InterfaceResolution{InterfaceID: "generic-cli", Source: "flag"}, true, true)
	if err != nil {
		t.Fatalf("buildRunInspectInput: %v", err)
	}
	if input.FailureClass != "approval_failure" {
		t.Fatalf("failure class=%q", input.FailureClass)
	}
	if input.MutationTarget.Path != ".atrakta/generated/repo-map.generated.json" {
		t.Fatalf("mutation target path=%q", input.MutationTarget.Path)
	}
	if len(input.GuidanceItems) < 4 {
		t.Fatalf("guidance item count too small: %d", len(input.GuidanceItems))
	}
	if input.PortabilityInput.InterfaceID != "generic-cli" {
		t.Fatalf("portability interface=%q", input.PortabilityInput.InterfaceID)
	}
}

func TestBuildRunApplyPlans(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	plans, err := buildRunApplyPlans(projectRoot, runpkg.InterfaceResolution{InterfaceID: "generic-cli", Source: "flag"}, map[string]any{"policy_entry_count": 1})
	if err != nil {
		t.Fatalf("buildRunApplyPlans: %v", err)
	}
	if len(plans) != 3 {
		t.Fatalf("plan count=%d", len(plans))
	}
	if plans[0].Target.Path != ".atrakta/generated/repo-map.generated.json" {
		t.Fatalf("first plan target=%q", plans[0].Target.Path)
	}
}

func writeTestMachineContract(t *testing.T, projectRoot string) {
	t.Helper()
	contractDir := filepath.Join(projectRoot, ".atrakta")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatalf("mkdir contract dir: %v", err)
	}
	contract := map[string]any{
		"v":          1,
		"project_id": "test",
		"interfaces": map[string]any{
			"supported": []string{"generic-cli", "cursor", "vscode", "mcp", "github-actions"},
			"fallback":  "generic-cli",
		},
		"boundary": map[string]any{
			"managed_root": ".atrakta/",
		},
		"tools": map[string]any{
			"allow": []string{"create", "edit", "run"},
		},
		"security": map[string]any{
			"destructive":      "deny",
			"external_send":    "deny",
			"approval":         "explicit",
			"permission_model": "proposal_only",
		},
		"routing": map[string]any{
			"default": map[string]any{"worker": "general"},
		},
	}
	b, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractDir, "contract.json"), append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
}

func prepareStartReadyProject(t *testing.T, projectRoot string) {
	t.Helper()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeTestMachineContract(t, projectRoot)
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := strings.TrimSpace(string(raw))
	if content == "" {
		return 0
	}
	return len(strings.Split(content, "\n"))
}

func readRunEventTypes(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	types := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("parse run-event line: %v", err)
		}
		if typ, _ := row["event_type"].(string); typ != "" {
			types = append(types, typ)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan run-events: %v", err)
	}
	return types
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return out
}
