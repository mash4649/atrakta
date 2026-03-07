package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/events"
	"atrakta/internal/progress"
)

func TestRebuildProgressFromEventsUsesStepLifecycle(t *testing.T) {
	repo := t.TempDir()
	appendStep := func(taskID, outcome string) {
		t.Helper()
		if _, err := events.Append(repo, "step", "worker", map[string]any{
			"task_id": taskID,
			"outcome": outcome,
		}); err != nil {
			t.Fatalf("append step event failed: %v", err)
		}
	}
	appendStep("feat-a", "PROGRESSED")
	appendStep("feat-a", "DONE")
	appendStep("feat-b", "NEEDS_APPROVAL")
	appendStep("adhoc", "DONE")

	pgr, err := RebuildProgressFromEvents(repo)
	if err != nil {
		t.Fatalf("rebuild progress failed: %v", err)
	}
	if len(pgr.CompletedFeatures) != 1 || pgr.CompletedFeatures[0] != "feat-a" {
		t.Fatalf("unexpected completed features: %#v", pgr.CompletedFeatures)
	}
	if pgr.ActiveFeature == nil || *pgr.ActiveFeature != "feat-b" {
		t.Fatalf("unexpected active feature: %#v", pgr.ActiveFeature)
	}
}

func TestRebuildStateFromEventsIncludesAdopt(t *testing.T) {
	repo := t.TempDir()
	if _, err := events.Append(repo, "apply", "orchestrator", map[string]any{
		"ops": []any{
			map[string]any{
				"op":          "adopt",
				"path":        ".cursor/AGENTS.md",
				"status":      "skipped",
				"interface":   "cursor",
				"template_id": "cursor:agents-md@1",
				"fingerprint": "sha256:fp-adopt",
				"kind":        "link",
				"target":      "AGENTS.md",
			},
		},
	}); err != nil {
		t.Fatalf("append apply event failed: %v", err)
	}
	st, err := RebuildStateFromEvents(repo)
	if err != nil {
		t.Fatalf("rebuild state failed: %v", err)
	}
	rec, ok := st.ManagedPaths[".cursor/AGENTS.md"]
	if !ok {
		t.Fatalf("expected adopted path to be reconstructed")
	}
	if rec.Kind != "link" || rec.TemplateID != "cursor:agents-md@1" || rec.Fingerprint != "sha256:fp-adopt" {
		t.Fatalf("unexpected record: %#v", rec)
	}
}

func TestRunRepairsCorruptedProgressFile(t *testing.T) {
	repo := t.TempDir()
	progressPath := filepath.Join(repo, ".atrakta", "progress.json")
	if err := os.MkdirAll(filepath.Dir(progressPath), 0o755); err != nil {
		t.Fatalf("mkdir .atrakta failed: %v", err)
	}
	if err := os.WriteFile(progressPath, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("write corrupted progress.json failed: %v", err)
	}

	report, _, err := Run(repo, "# AGENTS")
	if err != nil {
		t.Fatalf("doctor run failed: %v", err)
	}
	if !report.Rebuilt {
		t.Fatalf("expected doctor to repair corrupted files")
	}
	if !contains(report.Repairs, "progress_rebuilt") {
		t.Fatalf("expected progress_rebuilt in repairs, got=%v", report.Repairs)
	}
	if _, _, err := progress.LoadOrInit(repo); err != nil {
		t.Fatalf("expected repaired progress.json to be readable, err=%v", err)
	}
}

func TestRunBlocksWhenRequiredPromptPolicyMissing(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	c.Policies = &contract.Policies{
		PromptMin: &contract.PromptMinRef{
			Ref:      ".atrakta/policies/custom-required.json",
			Required: true,
			Apply:    "conditional",
		},
	}
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}

	report, _, err := Run(repo, "")
	if err == nil {
		t.Fatalf("expected doctor to fail when required policy is missing")
	}
	if report.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED outcome, got %s", report.Outcome)
	}
}

func TestRunBlocksOnInvalidTaskGraph(t *testing.T) {
	repo := t.TempDir()
	taskGraphPath := filepath.Join(repo, ".atrakta", "task-graph.json")
	if err := os.MkdirAll(filepath.Dir(taskGraphPath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(taskGraphPath, []byte("{bad"), 0o644); err != nil {
		t.Fatalf("write invalid task graph failed: %v", err)
	}

	report, _, err := Run(repo, "")
	if err == nil {
		t.Fatalf("expected doctor to fail on invalid task graph")
	}
	if report.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s", report.Outcome)
	}
}

func contains(list []string, v string) bool {
	for _, got := range list {
		if got == v {
			return true
		}
	}
	return false
}
