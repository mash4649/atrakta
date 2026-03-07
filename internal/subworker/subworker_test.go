package subworker

import (
	"reflect"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/projection"
)

func TestResolveConfigDefaultsToAuto(t *testing.T) {
	t.Setenv("ATRAKTA_SUBWORKER", "")
	t.Setenv("ATRAKTA_SUBWORKER_MAX_WORKERS", "")
	cfg := ResolveConfig(contract.Default(t.TempDir()))
	if cfg.Mode != "auto" {
		t.Fatalf("expected mode auto, got %s", cfg.Mode)
	}
	if cfg.MaxWorkers <= 0 {
		t.Fatalf("expected positive max_workers, got %d", cfg.MaxWorkers)
	}
}

func TestResolveConfigEnvCanForceOn(t *testing.T) {
	t.Setenv("ATRAKTA_SUBWORKER", "on")
	cfg := ResolveConfig(contract.Default(t.TempDir()))
	if cfg.Mode != "on" {
		t.Fatalf("expected mode on, got %s", cfg.Mode)
	}
}

func TestBuildPhaseAAutoDecision(t *testing.T) {
	cfg := Config{
		Mode:               "auto",
		MaxWorkers:         4,
		TimeoutMs:          12000,
		RetryLimit:         1,
		MaxDigestChars:     200,
		MaxOutputChars:     180,
		AutoMinProjections: 3,
		AutoMinScopes:      2,
	}
	det := model.DetectResult{
		TargetSet: []string{"cursor", "windsurf", "trae"},
		Reason:    model.ReasonConflict,
	}
	projections := []projection.Desired{
		{Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1"},
		{Path: ".windsurf/AGENTS.md", TemplateID: "windsurf:agents-md@1"},
		{Path: ".trae/AGENTS.md", TemplateID: "trae:agents-md@1"},
	}
	plan := BuildPhaseA(det, projections, cfg)
	if !plan.Decision.Enabled {
		t.Fatalf("expected auto mode to enable subworkers, reason=%s", plan.Decision.Reason)
	}
	if len(plan.Tasks) != 3 || len(plan.Results) != 3 {
		t.Fatalf("unexpected task/result counts: tasks=%d results=%d", len(plan.Tasks), len(plan.Results))
	}
}

func TestBuildPhaseADeterministicOrder(t *testing.T) {
	cfg := Config{
		Mode:               "on",
		MaxWorkers:         2,
		TimeoutMs:          12000,
		RetryLimit:         1,
		MaxDigestChars:     200,
		MaxOutputChars:     180,
		AutoMinProjections: 1,
		AutoMinScopes:      1,
	}
	det := model.DetectResult{TargetSet: []string{"cursor", "vscode"}, Reason: model.ReasonExplicit}
	projections := []projection.Desired{
		{Path: ".vscode/AGENTS.md", TemplateID: "vscode:agents-md@1"},
		{Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1"},
	}
	p1 := BuildPhaseA(det, projections, cfg)
	p2 := BuildPhaseA(det, projections, cfg)
	if len(p1.Tasks) != len(p2.Tasks) {
		t.Fatalf("task count mismatch")
	}
	if !reflect.DeepEqual(p1.Tasks, p2.Tasks) {
		t.Fatalf("task order is not deterministic")
	}
}

func TestMergePhaseAUsesProposals(t *testing.T) {
	cfg := Config{
		Mode:               "on",
		MaxWorkers:         2,
		TimeoutMs:          12000,
		RetryLimit:         1,
		MaxDigestChars:     200,
		MaxOutputChars:     180,
		AutoMinProjections: 1,
		AutoMinScopes:      1,
		BranchMode:         "auto",
		BranchAutoMinTasks: 2,
	}
	det := model.DetectResult{TargetSet: []string{"cursor", "vscode"}, Reason: model.ReasonExplicit}
	input := []projection.Desired{
		{Interface: "vscode", Path: ".vscode/AGENTS.md", TemplateID: "vscode:agents-md@1", Fingerprint: "sha256:vscode", Source: "AGENTS.md", Target: "AGENTS.md"},
		{Interface: "cursor", Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1", Fingerprint: "sha256:cursor", Source: "AGENTS.md", Target: "AGENTS.md"},
	}
	p := BuildPhaseA(det, input, cfg)
	merged, report, err := MergePhaseA(p, input)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if report.UsedFallback {
		t.Fatalf("expected merged proposals without fallback, reason=%s", report.Reason)
	}
	if len(merged) != len(input) {
		t.Fatalf("merged proposal count mismatch: got=%d want=%d", len(merged), len(input))
	}
}

func TestRollbackToSingleWriterOnProposalConflict(t *testing.T) {
	p := Plan{
		Config:   Config{Mode: "on", MaxDigestChars: 240, MaxOutputChars: 220},
		Decision: Decision{Mode: "on", Enabled: true, Reason: "mode_on"},
		Tasks: []Task{
			{
				WorkerID: "w01",
				Order:    1,
				Digest:   "ok",
				Proposals: []ProposedProjection{
					{Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1", Fingerprint: "sha256:a", Interface: "cursor"},
				},
			},
			{
				WorkerID: "w02",
				Order:    2,
				Digest:   "ok",
				Proposals: []ProposedProjection{
					{Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1", Fingerprint: "sha256:b", Interface: "cursor"},
				},
			},
		},
		Results: []Result{
			{Summary: "ok"},
			{Summary: "ok"},
		},
	}
	fallback := []projection.Desired{{Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1", Fingerprint: "sha256:fallback", Interface: "cursor"}}
	merged, report, err := MergePhaseA(p, fallback)
	if err != nil {
		t.Fatalf("merge error: %v", err)
	}
	if !report.UsedFallback {
		t.Fatalf("expected fallback on conflicting proposals")
	}
	if report.Reason != "proposal_conflict_fallback_single_writer" {
		t.Fatalf("unexpected reason: %s", report.Reason)
	}
	if !reflect.DeepEqual(merged, fallback) {
		t.Fatalf("expected fallback output")
	}
}

func TestBuildSingleWriterQueueOrdersByPathThenWorker(t *testing.T) {
	p := Plan{
		Config:   Config{Mode: "on", MaxDigestChars: 240, MaxOutputChars: 220},
		Decision: Decision{Mode: "on", Enabled: true, Reason: "mode_on"},
		Tasks: []Task{
			{
				WorkerID: "w02",
				Order:    2,
				Proposals: []ProposedProjection{
					{Path: ".b/AGENTS.md", TemplateID: "b:agents@1", Fingerprint: "sha256:b1", Interface: "b"},
					{Path: ".b/SHARED.md", TemplateID: "b:shared2@1", Fingerprint: "sha256:b3", Interface: "b"},
				},
			},
			{
				WorkerID: "w01",
				Order:    1,
				Proposals: []ProposedProjection{
					{Path: ".a/AGENTS.md", TemplateID: "a:agents@1", Fingerprint: "sha256:a1", Interface: "a"},
					{Path: ".b/EXTRA.md", TemplateID: "b:extra@1", Fingerprint: "sha256:b2", Interface: "b"},
					{Path: ".b/SHARED.md", TemplateID: "b:shared1@1", Fingerprint: "sha256:b4", Interface: "b"},
				},
			},
		},
	}
	selected := []projection.Desired{
		{Path: ".b/EXTRA.md", TemplateID: "b:extra@1", Fingerprint: "sha256:b2", Interface: "b"},
		{Path: ".b/SHARED.md", TemplateID: "b:shared2@1", Fingerprint: "sha256:b3", Interface: "b"},
		{Path: ".b/AGENTS.md", TemplateID: "b:agents@1", Fingerprint: "sha256:b1", Interface: "b"},
		{Path: ".a/AGENTS.md", TemplateID: "a:agents@1", Fingerprint: "sha256:a1", Interface: "a"},
		{Path: ".b/SHARED.md", TemplateID: "b:shared1@1", Fingerprint: "sha256:b4", Interface: "b"},
	}
	q := BuildSingleWriterQueue(p, selected, false)
	if err := ValidateSingleWriterQueue(q); err != nil {
		t.Fatalf("queue validation failed: %v", err)
	}
	if q.ItemCount != 5 || len(q.Items) != 5 {
		t.Fatalf("unexpected queue item count: %d", q.ItemCount)
	}
	if q.Items[0].Path != ".a/AGENTS.md" || q.Items[0].WorkerID != "w01" {
		t.Fatalf("unexpected first queue item: %+v", q.Items[0])
	}
	if q.Items[1].Path != ".b/AGENTS.md" || q.Items[1].WorkerID != "w02" {
		t.Fatalf("unexpected second queue item: %+v", q.Items[1])
	}
	if q.Items[2].Path != ".b/EXTRA.md" || q.Items[2].WorkerID != "w01" {
		t.Fatalf("unexpected third queue item: %+v", q.Items[2])
	}
	sharedSeen := []string{}
	for _, it := range q.Items {
		if it.Path == ".b/SHARED.md" {
			sharedSeen = append(sharedSeen, it.WorkerID)
		}
	}
	if !reflect.DeepEqual(sharedSeen, []string{"w01", "w02"}) {
		t.Fatalf("expected worker order for same path to be deterministic, got=%v", sharedSeen)
	}
	out := QueueProjections(q)
	if len(out) != 5 || out[0].Path != ".a/AGENTS.md" || out[4].Path != ".b/SHARED.md" {
		t.Fatalf("queue projections order mismatch")
	}
}

func TestValidateSingleWriterQueueDetectsNonDeterministicOrder(t *testing.T) {
	q := IntegrationQueue{
		Mode:      "single_writer",
		ItemCount: 2,
		Items: []QueueItem{
			{Index: 1, Path: ".z/AGENTS.md", WorkerID: "w02", TemplateID: "z:agents@1", Fingerprint: "sha256:z"},
			{Index: 2, Path: ".a/AGENTS.md", WorkerID: "w01", TemplateID: "a:agents@1", Fingerprint: "sha256:a"},
		},
	}
	if err := ValidateSingleWriterQueue(q); err == nil {
		t.Fatalf("expected queue order validation error")
	}
}

func TestApplyBudgetGuardCompressesDigestAndSummary(t *testing.T) {
	p := Plan{
		Config:   Config{Mode: "on", MaxDigestChars: 240, MaxOutputChars: 220},
		Decision: Decision{Mode: "on", Enabled: true, Reason: "mode_on"},
		Tasks: []Task{
			{
				WorkerID:  "w01",
				Digest:    "scope=.cursor/ ops=4 sample_paths=.cursor/AGENTS.md,.cursor/.atrakta-link,.cursor/CONTRACT.json,.cursor/NOTES.md",
				Proposals: []ProposedProjection{{Path: ".cursor/AGENTS.md", TemplateID: "cursor:agents-md@1", Fingerprint: "sha256:1"}},
			},
		},
		Results: []Result{
			{
				Summary:       "scope .cursor/ proposed 4 operation(s) with extended context summary for verification and planning purposes",
				TokenEstimate: 900,
			},
		},
	}
	out, report := ApplyBudgetGuard(p, contract.TokenBudget{Soft: 80, Hard: 200})
	if !report.Applied {
		t.Fatalf("expected budget guard to be applied")
	}
	if !out.Decision.Enabled {
		t.Fatalf("expected plan to stay enabled after successful compression")
	}
	if len(out.Tasks[0].Digest) >= len(p.Tasks[0].Digest) {
		t.Fatalf("expected digest to be trimmed")
	}
	if len(out.Results[0].Summary) >= len(p.Results[0].Summary) {
		t.Fatalf("expected summary to be trimmed")
	}
}

func TestApplyBudgetGuardDisablesWhenHardLimitStillExceeded(t *testing.T) {
	p := Plan{
		Config:   Config{Mode: "on", MaxDigestChars: 240, MaxOutputChars: 220},
		Decision: Decision{Mode: "on", Enabled: true, Reason: "mode_on"},
		Tasks: []Task{
			{
				WorkerID: "w01",
				Digest:   "scope=.a/ ops=10 sample_paths=.a/1,.a/2,.a/3",
				Proposals: []ProposedProjection{
					{Path: ".a/1", TemplateID: "a:t@1", Fingerprint: "sha256:1"},
					{Path: ".a/2", TemplateID: "a:t@2", Fingerprint: "sha256:2"},
					{Path: ".a/3", TemplateID: "a:t@3", Fingerprint: "sha256:3"},
					{Path: ".a/4", TemplateID: "a:t@4", Fingerprint: "sha256:4"},
					{Path: ".a/5", TemplateID: "a:t@5", Fingerprint: "sha256:5"},
					{Path: ".a/6", TemplateID: "a:t@6", Fingerprint: "sha256:6"},
					{Path: ".a/7", TemplateID: "a:t@7", Fingerprint: "sha256:7"},
					{Path: ".a/8", TemplateID: "a:t@8", Fingerprint: "sha256:8"},
					{Path: ".a/9", TemplateID: "a:t@9", Fingerprint: "sha256:9"},
					{Path: ".a/10", TemplateID: "a:t@10", Fingerprint: "sha256:10"},
				},
			},
		},
		Results: []Result{{Summary: "long summary", TokenEstimate: 9999}},
	}
	out, report := ApplyBudgetGuard(p, contract.TokenBudget{Soft: 50, Hard: 10})
	if !report.Disabled {
		t.Fatalf("expected guard to disable subworker plan when hard limit is exceeded")
	}
	if out.Decision.Enabled {
		t.Fatalf("expected subworker decision to be disabled")
	}
	if len(out.Tasks) != 0 || len(out.Results) != 0 {
		t.Fatalf("expected tasks/results to be cleared on hard limit fallback")
	}
}
