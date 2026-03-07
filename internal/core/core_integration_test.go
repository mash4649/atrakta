package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"atrakta/internal/adapter"
	"atrakta/internal/checkpoint"
	"atrakta/internal/contract"
	"atrakta/internal/core"
	"atrakta/internal/doctor"
	"atrakta/internal/events"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/state"
	"atrakta/internal/util"
)

type testAdapter struct{}

func (testAdapter) EmitStatus(message string)                  {}
func (testAdapter) PresentDiff(summary string, details string) {}
func (testAdapter) RequestApproval(context any) adapter.ApprovalResponse {
	return adapter.ApprovalResponse{Approved: true}
}
func (testAdapter) RequestInput(prompt string, schema map[string]any) adapter.InputResponse {
	return adapter.InputResponse{Value: nil}
}
func (testAdapter) PresentNextAction(next model.NextAction) {}
func (testAdapter) NotifyBlocked(reason string)             {}

type denyApprovalAdapter struct{}

func (denyApprovalAdapter) EmitStatus(message string)                  {}
func (denyApprovalAdapter) PresentDiff(summary string, details string) {}
func (denyApprovalAdapter) RequestApproval(context any) adapter.ApprovalResponse {
	return adapter.ApprovalResponse{Approved: false}
}
func (denyApprovalAdapter) RequestInput(prompt string, schema map[string]any) adapter.InputResponse {
	return adapter.InputResponse{Value: nil}
}
func (denyApprovalAdapter) PresentNextAction(next model.NextAction) {}
func (denyApprovalAdapter) NotifyBlocked(reason string)             {}

func TestA1StartCreatesOnlyRequiredProjection(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "human constitution\n")

	_, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	mustExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
	mustNotExist(t, filepath.Join(repo, ".windsurf", "AGENTS.md"))
	mustNotExist(t, filepath.Join(repo, ".vscode", "AGENTS.md"))
}

func TestAutoCreatesRootAGENTSWhenMissing(t *testing.T) {
	repo := t.TempDir()
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("start failed without initial AGENTS.md: %v", err)
	}
	agentsPath := filepath.Join(repo, "AGENTS.md")
	mustExist(t, agentsPath)
	b, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read auto-created AGENTS.md: %v", err)
	}
	text := string(b)
	if !strings.Contains(text, "Machine-executable contract is located at `.atrakta/contract.json`.") {
		t.Fatalf("auto-created AGENTS.md missing contract header")
	}
	mustExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
}

func TestA2PruneNeverRunsUnderAmbiguity(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "trae"}); err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	mustExist(t, filepath.Join(repo, ".trae", "AGENTS.md"))

	if err := os.MkdirAll(filepath.Join(repo, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".windsurf"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{})
	if err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	if res.Detect.Reason != model.ReasonConflict && res.Detect.Reason != model.ReasonMixed {
		t.Fatalf("expected ambiguity reason conflict/mixed, got %s", res.Detect.Reason)
	}
	mustExist(t, filepath.Join(repo, ".trae", "AGENTS.md"))
}

func TestRunCheckpointSavedOnDone(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", FeatureID: "feat-checkpoint"}); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	cp, err := checkpoint.LoadLatest(repo)
	if err != nil {
		t.Fatalf("load checkpoint failed: %v", err)
	}
	if cp.Stage != "done" || cp.Outcome != "done" {
		t.Fatalf("unexpected checkpoint stage/outcome: %#v", cp)
	}
	if cp.FeatureID != "feat-checkpoint" || cp.PlanID == "" || cp.TaskGraphID == "" {
		t.Fatalf("missing checkpoint identifiers: %#v", cp)
	}
}

func TestRunCheckpointSavedOnNeedsApproval(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	c := contract.Default(repo)
	c.Tools.ApprovalRequiredFor = []string{"boundary_expand", "external_side_effect", "destructive_prune"}
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}
	cursorPath := filepath.Join(repo, ".cursor", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o755); err != nil {
		t.Fatalf("mkdir cursor dir failed: %v", err)
	}
	mustWrite(t, cursorPath, "manual\n")
	res, err := core.Start(repo, denyApprovalAdapter{}, core.StartFlags{Interfaces: "cursor", FeatureID: "feat-approval"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "NEEDS_APPROVAL" {
		t.Fatalf("expected NEEDS_APPROVAL, got %s", res.Step.Outcome)
	}
	cp, err := checkpoint.LoadLatest(repo)
	if err != nil {
		t.Fatalf("load checkpoint failed: %v", err)
	}
	if cp.Stage != "needs_approval" || cp.Outcome != "needs_approval" {
		t.Fatalf("unexpected checkpoint: %#v", cp)
	}
}

func TestResumeFromNeedsApprovalIsDeterministic(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", FeatureID: "feat-resume"}); err != nil {
		t.Fatalf("bootstrap start failed: %v", err)
	}

	first, err := core.Start(repo, denyApprovalAdapter{}, core.StartFlags{Interfaces: "trae", FeatureID: "feat-resume"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if first.Step.Outcome != "NEEDS_APPROVAL" {
		t.Fatalf("expected NEEDS_APPROVAL, got %s", first.Step.Outcome)
	}
	cp, err := checkpoint.LoadLatest(repo)
	if err != nil {
		t.Fatalf("load checkpoint failed: %v", err)
	}
	if cp.Stage != "needs_approval" || cp.FeatureID != "feat-resume" {
		t.Fatalf("unexpected checkpoint stage: %#v", cp)
	}

	flags := core.StartFlags{
		Interfaces: cp.Interfaces,
		FeatureID:  cp.FeatureID,
		SyncLevel:  cp.SyncLevel,
	}
	second, err := core.Start(repo, testAdapter{}, flags)
	if err != nil {
		t.Fatalf("resume start failed: %v", err)
	}
	if second.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE after resume, got %s", second.Step.Outcome)
	}
	if err := events.VerifyChain(repo); err != nil {
		t.Fatalf("events chain verification failed: %v", err)
	}
	st1, _, err := state.LoadOrEmpty(repo, "")
	if err != nil {
		t.Fatalf("load state after resume failed: %v", err)
	}

	third, err := core.Start(repo, testAdapter{}, flags)
	if err != nil {
		t.Fatalf("rerun start failed: %v", err)
	}
	if third.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE on rerun, got %s", third.Step.Outcome)
	}
	if err := events.VerifyChain(repo); err != nil {
		t.Fatalf("events chain verification after rerun failed: %v", err)
	}
	st2, _, err := state.LoadOrEmpty(repo, "")
	if err != nil {
		t.Fatalf("load state after rerun failed: %v", err)
	}
	if !reflect.DeepEqual(st1.ManagedPaths, st2.ManagedPaths) {
		t.Fatalf("managed state changed across deterministic rerun")
	}
}

func TestA8StateRebuildFromEvents(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	orig, _, err := state.LoadOrEmpty(repo, "")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if err := os.Remove(filepath.Join(repo, ".atrakta", "state.json")); err != nil {
		t.Fatalf("remove state: %v", err)
	}
	rebuilt, err := doctor.RebuildStateFromEvents(repo)
	if err != nil {
		t.Fatalf("rebuild state: %v", err)
	}

	if len(orig.ManagedPaths) != len(rebuilt.ManagedPaths) {
		t.Fatalf("managed paths count mismatch: orig=%d rebuilt=%d", len(orig.ManagedPaths), len(rebuilt.ManagedPaths))
	}
	for p, rec := range orig.ManagedPaths {
		r2, ok := rebuilt.ManagedPaths[p]
		if !ok {
			t.Fatalf("missing rebuilt path: %s", p)
		}
		if rec.Fingerprint != r2.Fingerprint || rec.TemplateID != r2.TemplateID || rec.Interface != r2.Interface {
			t.Fatalf("rebuilt record mismatch for %s", p)
		}
	}
}

func TestA9ProjectionFingerprintNormalization(t *testing.T) {
	contractHash := "sha256:contract"
	templateID := "cursor:agents-md@1"
	lf := "line1\nline2\n"
	crlf := "line1\r\nline2\r\n"
	h1 := util.SHA256Tagged([]byte(util.NormalizeContentLF(lf)))
	h2 := util.SHA256Tagged([]byte(util.NormalizeContentLF(crlf)))
	if h1 != h2 {
		t.Fatalf("normalized content hash mismatch")
	}
	fp1 := projection.Fingerprint(contractHash, templateID, h1)
	fp2 := projection.Fingerprint(contractHash, templateID, h2)
	if fp1 != fp2 {
		t.Fatalf("fingerprint mismatch across line endings")
	}
}

func TestA6ReplayDeterminismFromFreshWorkspace(t *testing.T) {
	repo1 := t.TempDir()
	mustWrite(t, filepath.Join(repo1, "AGENTS.md"), "constitution\n")

	if _, err := core.Start(repo1, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("start cursor failed: %v", err)
	}
	if _, err := core.Start(repo1, testAdapter{}, core.StartFlags{Interfaces: "trae"}); err != nil {
		t.Fatalf("start trae failed: %v", err)
	}

	orig, _, err := state.LoadOrEmpty(repo1, "")
	if err != nil {
		t.Fatalf("load original state: %v", err)
	}

	repo2 := t.TempDir()
	mustWrite(t, filepath.Join(repo2, "AGENTS.md"), "constitution\n")
	if err := os.MkdirAll(filepath.Join(repo2, ".atrakta"), 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	copyFile(t, filepath.Join(repo1, ".atrakta", "contract.json"), filepath.Join(repo2, ".atrakta", "contract.json"))
	copyFile(t, filepath.Join(repo1, ".atrakta", "events.jsonl"), filepath.Join(repo2, ".atrakta", "events.jsonl"))

	rebuilt1, err := doctor.RebuildStateFromEvents(repo2)
	if err != nil {
		t.Fatalf("rebuild state #1 failed: %v", err)
	}
	rebuilt2, err := doctor.RebuildStateFromEvents(repo2)
	if err != nil {
		t.Fatalf("rebuild state #2 failed: %v", err)
	}

	if !reflect.DeepEqual(rebuilt1.ManagedPaths, rebuilt2.ManagedPaths) {
		t.Fatalf("rebuild is not deterministic")
	}
	if len(orig.ManagedPaths) != len(rebuilt1.ManagedPaths) {
		t.Fatalf("managed paths count mismatch: orig=%d rebuilt=%d", len(orig.ManagedPaths), len(rebuilt1.ManagedPaths))
	}
	for p, rec := range orig.ManagedPaths {
		r2, ok := rebuilt1.ManagedPaths[p]
		if !ok {
			t.Fatalf("missing rebuilt path: %s", p)
		}
		if rec.Interface != r2.Interface || rec.TemplateID != r2.TemplateID || rec.Fingerprint != r2.Fingerprint || rec.Target != r2.Target || rec.Kind != r2.Kind {
			t.Fatalf("replayed record mismatch for %s", p)
		}
	}
}

func TestA5ManagedOnlyTamperBlocksPrune(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("bootstrap start failed: %v", err)
	}

	cPath := filepath.Join(repo, ".atrakta", "contract.json")
	cb, err := os.ReadFile(cPath)
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	var c contract.Contract
	if err := json.Unmarshal(cb, &c); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	c.Tools.ApprovalRequiredFor = []string{"boundary_expand", "external_side_effect"}
	nb, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(cPath, append(nb, '\n'), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	cursorPath := filepath.Join(repo, ".cursor", "AGENTS.md")
	if err := os.Remove(cursorPath); err != nil {
		t.Fatalf("remove projected file: %v", err)
	}
	mustWrite(t, cursorPath, "tampered\n")

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "trae"})
	if err == nil {
		t.Fatalf("expected start to be blocked")
	}
	if !strings.Contains(err.Error(), "apply failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Step.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED outcome, got %s", res.Step.Outcome)
	}
	mustExist(t, cursorPath)
}

func TestFeatureIDProgressLifecycle(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	feature := "feat-phase2"
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", FeatureID: feature})
	if err != nil {
		t.Fatalf("start with feature failed: %v", err)
	}
	if res.Plan.FeatureID != feature || res.Apply.FeatureID != feature {
		t.Fatalf("feature_id not propagated to plan/apply")
	}

	pb, err := os.ReadFile(filepath.Join(repo, ".atrakta", "progress.json"))
	if err != nil {
		t.Fatalf("read progress.json: %v", err)
	}
	var p map[string]any
	if err := json.Unmarshal(pb, &p); err != nil {
		t.Fatalf("parse progress.json: %v", err)
	}
	if p["active_feature"] != nil {
		t.Fatalf("expected active_feature nil after successful completion")
	}
	completed, ok := p["completed_features"].([]any)
	if !ok || len(completed) == 0 {
		t.Fatalf("expected completed_features to include feature")
	}
}

func TestSubworkerAutoDecisionDefaultOffForSmallWorkload(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	t.Setenv("ATRAKTA_SUBWORKER", "")
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE outcome, got %s", res.Step.Outcome)
	}
	ev, err := events.ReadAll(repo)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	dispatches := 0
	for _, e := range ev {
		if tp, _ := e.Raw["type"].(string); tp == "subworker_dispatch" {
			dispatches++
		}
	}
	if dispatches != 0 {
		t.Fatalf("expected no subworker dispatch for small workload, got %d", dispatches)
	}
}

func TestSubworkerAutoDecisionCanTurnOn(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	t.Setenv("ATRAKTA_SUBWORKER", "auto")
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor,vscode,windsurf,trae,antigravity"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE outcome, got %s", res.Step.Outcome)
	}
	ev, err := events.ReadAll(repo)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	dispatches := 0
	results := 0
	for _, e := range ev {
		tp, _ := e.Raw["type"].(string)
		if tp == "subworker_dispatch" {
			dispatches++
		}
		if tp == "subworker_result" {
			results++
		}
	}
	if dispatches == 0 {
		t.Fatalf("expected subworker dispatch event")
	}
	if results == 0 {
		t.Fatalf("expected subworker result events")
	}
}

func TestSubworkerBranchPlanCanBeEnabledInLimitedMode(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	t.Setenv("ATRAKTA_SUBWORKER", "on")
	t.Setenv("ATRAKTA_SUBWORKER_BRANCH_PARALLEL", "on")
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor,vscode,windsurf,trae"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE outcome, got %s", res.Step.Outcome)
	}
	ev, err := events.ReadAll(repo)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	branchPlan := 0
	queueCount := 0
	for _, e := range ev {
		if tp, _ := e.Raw["type"].(string); tp == "subworker_branch_plan" {
			branchPlan++
		}
		if tp, _ := e.Raw["type"].(string); tp == "single_writer_queue" {
			queueCount++
		}
	}
	if branchPlan == 0 {
		t.Fatalf("expected branch plan event when branch mode is on")
	}
	if queueCount == 0 {
		t.Fatalf("expected single_writer_queue event when subworker flow runs")
	}
}

func TestPlan3EventsContextRoutingPolicyRecorded(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "root constitution\nimport: docs/shared.md\n")
	mustWrite(t, filepath.Join(repo, "docs", "shared.md"), "shared rules\n")

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE outcome, got %s", res.Step.Outcome)
	}
	ev, err := events.ReadAll(repo)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	hasContext := false
	hasRouting := false
	hasPolicy := false
	hasTaskGraph := false
	for _, e := range ev {
		tp, _ := e.Raw["type"].(string)
		switch tp {
		case "context_resolved":
			hasContext = true
		case "routing_decision":
			hasRouting = true
		case "policy_applied":
			hasPolicy = true
		case "task_graph_planned":
			hasTaskGraph = true
		}
	}
	if !hasContext || !hasRouting || !hasPolicy || !hasTaskGraph {
		t.Fatalf("expected context/routing/policy/task_graph events, got context=%v routing=%v policy=%v task_graph=%v", hasContext, hasRouting, hasPolicy, hasTaskGraph)
	}
	mustExist(t, filepath.Join(repo, ".atrakta", "task-graph.json"))
}

func TestReadOnlySecurityBlocksBeforeApplyMutation(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	c := contract.Default(repo)
	c.Security = &contract.Security{Profile: "read_only"}
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract: %v", err)
	}

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"})
	if err == nil {
		t.Fatalf("expected read_only security profile to block start")
	}
	if res.Step.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED outcome, got %s", res.Step.Outcome)
	}
	mustNotExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
}

func TestMissingManagedArtifactGetsRepaired(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("initial start failed: %v", err)
	}
	path := filepath.Join(repo, ".cursor", "AGENTS.md")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove managed artifact failed: %v", err)
	}

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"})
	if err != nil {
		t.Fatalf("repair start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE outcome, got %s", res.Step.Outcome)
	}
	mustExist(t, path)
}

func TestFirstRunWithoutInterfaceReturnsNeedsInput(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "NEEDS_INPUT" {
		t.Fatalf("expected NEEDS_INPUT, got %s", res.Step.Outcome)
	}
	mustNotExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
}

func TestUnknownFallsBackToAutoLastWithoutPrune(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("bootstrap start failed: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".cursor")); err != nil {
		t.Fatalf("remove cursor projection: %v", err)
	}

	res, err := core.Start(repo, testAdapter{}, core.StartFlags{})
	if err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	if res.Detect.Reason != model.ReasonAutoLast {
		t.Fatalf("expected auto_last detect reason, got %s", res.Detect.Reason)
	}
	mustExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
}

func TestTriggerInterfaceDoesNotPrunePreviousManagedSet(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor"}); err != nil {
		t.Fatalf("cursor start failed: %v", err)
	}
	t.Setenv("ATRAKTA_TRIGGER_INTERFACE", "trae")
	t.Setenv("ATRAKTA_TRIGGER_SOURCE", "wrapper")
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{})
	if err != nil {
		t.Fatalf("triggered start failed: %v", err)
	}
	if res.Detect.Reason != model.ReasonTriggered {
		t.Fatalf("expected triggered reason, got %s", res.Detect.Reason)
	}
	mustExist(t, filepath.Join(repo, ".cursor", "AGENTS.md"))
	mustExist(t, filepath.Join(repo, ".trae", "AGENTS.md"))
}

func TestSLORepoMapTokenBudgetRespected(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")
	mustWrite(t, filepath.Join(repo, "CONVENTIONS.md"), strings.Repeat("rule\n", 400))

	const budget = 80
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", MapTokens: budget, MapRefresh: 3600})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE, got %s", res.Step.Outcome)
	}
	ev, err := events.ReadAll(repo)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	used := -1
	for _, e := range ev {
		tp, _ := e.Raw["type"].(string)
		if tp != "repo_map" {
			continue
		}
		switch v := e.Raw["used_tokens"].(type) {
		case float64:
			used = int(v)
		case int:
			used = v
		}
	}
	if used < 0 {
		t.Fatalf("repo_map event with used_tokens not found")
	}
	if used > budget {
		t.Fatalf("repo_map used_tokens exceeded budget: used=%d budget=%d", used, budget)
	}
}

func TestStartFastPathSnapshotHit(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "AGENTS.md"), "constitution\n")

	if _, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", FeatureID: "feat-fast"}); err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	res, err := core.Start(repo, testAdapter{}, core.StartFlags{Interfaces: "cursor", FeatureID: "feat-fast"})
	if err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	if res.Step.Outcome != "DONE" {
		t.Fatalf("expected DONE, got %s", res.Step.Outcome)
	}
	if res.Step.Gate.Quick != model.GatePass || res.Step.Gate.Reason != "fast_path_snapshot" {
		t.Fatalf("expected fast path step gate, got %#v", res.Step.Gate)
	}
	ev, err := events.ReadAll(repo)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	found := false
	for _, e := range ev {
		tp, _ := e.Raw["type"].(string)
		if tp == "start_fast_hit" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("start_fast_hit event not found")
	}
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected path to exist: %s (%v)", p, err)
	}
}

func mustNotExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("expected path not to exist: %s", p)
	}
}

func mustWrite(t *testing.T, p, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func dump(t *testing.T, v any) string {
	t.Helper()
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}
