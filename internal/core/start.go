package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"atrakta/internal/adapter"
	"atrakta/internal/apply"
	"atrakta/internal/bootstrap"
	"atrakta/internal/checkpoint"
	agentsctx "atrakta/internal/context"
	"atrakta/internal/contract"
	"atrakta/internal/detect"
	"atrakta/internal/doctor"
	"atrakta/internal/events"
	"atrakta/internal/gate"
	"atrakta/internal/gitauto"
	"atrakta/internal/ifaceauto"
	"atrakta/internal/manifest"
	"atrakta/internal/model"
	"atrakta/internal/plan"
	"atrakta/internal/policy"
	"atrakta/internal/progress"
	"atrakta/internal/projection"
	"atrakta/internal/proof"
	"atrakta/internal/registry"
	"atrakta/internal/repomap"
	"atrakta/internal/routing"
	"atrakta/internal/runtimeobs"
	"atrakta/internal/startfast"
	"atrakta/internal/state"
	"atrakta/internal/subworker"
	"atrakta/internal/syncpolicy"
	"atrakta/internal/taskgraph"
	"atrakta/internal/util"
)

type StartFlags struct {
	Interfaces string
	FeatureID  string
	SyncLevel  string
	MapTokens  int
	MapRefresh int
}

type StartResult struct {
	Detect model.DetectResult
	Plan   model.PlanResult
	Apply  model.ApplyResult
	Gate   model.GateResult
	Step   model.StepEvent
}

func Start(repoRoot string, ad adapter.Adapter, flags StartFlags) (StartResult, error) {
	startedAt := time.Now()
	defer func() {
		snap, err := runtimeobs.Record(repoRoot, "start", time.Since(startedAt))
		if err == nil {
			ad.EmitStatus(fmt.Sprintf("runtime metrics: start last=%dms p95=%dms n=%d", snap.LastMs, snap.P95Ms, snap.Count))
		}
	}()
	c, cb, err := contract.LoadOrInit(repoRoot)
	if err != nil {
		return StartResult{}, err
	}
	c = contract.CanonicalizeBoundary(c)
	contractHash := contract.ContractHash(cb)
	resolvedSyncLevel := strings.TrimSpace(flags.SyncLevel)
	if resolvedSyncLevel == "" {
		resolvedSyncLevel = strings.TrimSpace(os.Getenv("ATRAKTA_SYNC_LEVEL"))
	}
	mapTokens, mapRefresh := resolveRepoMapSettings(c, flags)
	taskCategory := resolveTaskCategory()
	fastTargets, fastReason := resolveFastPathInterfaces(flags)
	fastTargetKey := canonicalInterfaceKey(fastTargets)
	fastFeatureID := strings.TrimSpace(flags.FeatureID)
	if fastFeatureID == "" {
		fastFeatureID = "adhoc"
	}
	fastInput := startfast.Input{
		ContractHash:   contractHash,
		WorkspaceStamp: util.WorkspaceStamp(repoRoot, c.Boundary.Include),
		Interfaces:     fastTargetKey,
		FeatureID:      fastFeatureID,
		ConfigKey:      startFastConfigKey(resolvedSyncLevel, mapTokens, mapRefresh, taskCategory),
	}
	if fastTargetKey != "" {
		decision, ferr := startfast.Check(repoRoot, fastInput, time.Now().UTC())
		if ferr != nil {
			ad.EmitStatus("start fast gate unavailable; fallback strict path")
		} else if decision.Hit {
			if !managedArtifactsIntact(repoRoot) {
				ad.EmitStatus("start fast-path bypassed: managed artifacts drift detected")
			} else {
				if err := events.VerifyChainCached(repoRoot); err != nil {
					ad.NotifyBlocked("events.jsonl corrupted")
					return StartResult{}, err
				}
				defer func() {
					if err := events.Flush(repoRoot); err != nil {
						ad.EmitStatus("events flush failed: " + err.Error())
					}
				}()
				_, _ = events.Append(repoRoot, "start_fast_hit", "orchestrator", map[string]any{
					"reason":      decision.Reason,
					"target_set":  fastTargets,
					"feature_id":  fastFeatureID,
					"config_key":  fastInput.ConfigKey,
					"sync_level":  resolvedSyncLevel,
					"map_tokens":  mapTokens,
					"map_refresh": mapRefresh,
				})
				next := model.NextAction{Kind: "done", Hint: "completed (fast path)", Command: "atrakta start"}
				step := model.StepEvent{
					ActorRole:  "worker",
					TaskID:     fastFeatureID,
					Outcome:    "DONE",
					Gate:       model.GateResult{Safety: model.GatePass, Quick: model.GatePass, Reason: "fast_path_snapshot"},
					NextAction: next,
				}
				_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
				cp := checkpoint.RunCheckpoint{
					Interfaces:   fastTargetKey,
					FeatureID:    fastFeatureID,
					SyncLevel:    resolvedSyncLevel,
					DetectReason: string(fastReason),
					Stage:        "done_fast",
					Outcome:      "done",
				}
				writeCheckpointBestEffort(repoRoot, ad, cp)
				ad.EmitStatus("start fast-path hit: snapshot_match")
				ad.PresentNextAction(next)
				return StartResult{
					Detect: model.DetectResult{TargetSet: fastTargets, PruneAllowed: false, Reason: fastReason},
					Step:   step,
				}, nil
			}
		}
	}
	var gitPre gitauto.Snapshot
	if c.Policies != nil && c.Policies.PromptMin != nil {
		if created, err := policy.EnsureDefaultPromptMin(repoRoot, c.Policies.PromptMin.Ref); err != nil {
			ad.NotifyBlocked("failed to initialize prompt policy")
			return StartResult{}, err
		} else if created {
			ad.EmitStatus("initialized .atrakta/policies/prompt-min.json")
		}
	}
	if err := events.VerifyChainCached(repoRoot); err != nil {
		ad.NotifyBlocked("events.jsonl corrupted")
		return StartResult{}, err
	}
	defer func() {
		if err := events.Flush(repoRoot); err != nil {
			ad.EmitStatus("events flush failed: " + err.Error())
		}
	}()
	gitMode := gitauto.ResolveMode(c)
	gitSetup, err := gitauto.EnsureSetup(repoRoot, gitMode)
	if err != nil {
		ad.NotifyBlocked("git setup failed")
		return StartResult{}, err
	}
	if _, err := events.Append(repoRoot, "git_setup", "orchestrator", structToMap(gitSetup)); err != nil {
		return StartResult{}, err
	}
	if gitSetup.Performed {
		ad.EmitStatus("git setup: initialized repository automatically")
	}
	gitPre = gitauto.Capture(repoRoot)
	if _, err := events.Append(repoRoot, "git_snapshot", "orchestrator", map[string]any{
		"phase":    "pre",
		"snapshot": structToMap(gitPre),
	}); err != nil {
		return StartResult{}, err
	}

	_, createdAgents, err := bootstrap.EnsureRootAGENTS(repoRoot)
	if err != nil {
		ad.NotifyBlocked("failed to initialize AGENTS.md")
		return StartResult{}, err
	}
	if createdAgents {
		ad.EmitStatus("created AGENTS.md at repo root")
	}
	sourceAGENTS, contextReport, err := agentsctx.Resolve(agentsctx.ResolveInput{
		RepoRoot: repoRoot,
		StartDir: repoRoot,
		Config:   c.Context,
	})
	if err != nil {
		ad.NotifyBlocked("failed to resolve context AGENTS")
		return StartResult{}, err
	}
	if _, err := events.Append(repoRoot, "context_resolved", "orchestrator", structToMap(contextReport)); err != nil {
		return StartResult{}, err
	}
	repoMapReport, repoMapErr := repomap.LoadOrRefresh(repoRoot, repomap.Config{
		MaxTokens:      mapTokens,
		RefreshSeconds: mapRefresh,
		Includes:       c.Boundary.Include,
		Excludes:       c.Boundary.Exclude,
	})
	if repoMapErr == nil {
		if _, err := events.Append(repoRoot, "repo_map", "orchestrator", structToMap(repoMapReport)); err != nil {
			return StartResult{}, err
		}
		if repoMapReport.Refreshed {
			ad.EmitStatus(fmt.Sprintf("repo map refreshed: files=%d tokens=%d", repoMapReport.FileCount, repoMapReport.UsedTokens))
		}
	} else {
		ad.EmitStatus("repo map unavailable; continuing without refresh")
	}
	routeDecision := routing.Resolve(c, taskCategory)
	if _, err := events.Append(repoRoot, "routing_decision", "orchestrator", structToMap(routeDecision)); err != nil {
		return StartResult{}, err
	}
	pgr, createdProgress, err := progress.LoadOrInit(repoRoot)
	if err != nil {
		ad.NotifyBlocked("failed to initialize progress.json")
		return StartResult{}, err
	}
	if createdProgress {
		ad.EmitStatus("initialized progress.json")
	}
	resolvedInterfaces := strings.TrimSpace(flags.Interfaces)
	if resolvedInterfaces == "" {
		resolvedInterfaces = strings.TrimSpace(os.Getenv("ATRAKTA_INTERFACES"))
	}
	cp := checkpoint.RunCheckpoint{
		Interfaces: resolvedInterfaces,
		SyncLevel:  resolvedSyncLevel,
		Stage:      "start_initialized",
		Outcome:    "running",
	}
	featureID := strings.TrimSpace(flags.FeatureID)
	if featureID == "" {
		if pgr.ActiveFeature != nil && *pgr.ActiveFeature != "" {
			featureID = *pgr.ActiveFeature
			ad.EmitStatus("resuming active feature: " + featureID)
		} else {
			featureID = "adhoc"
		}
	}
	cp.FeatureID = featureID
	writeCheckpointBestEffort(repoRoot, ad, cp)
	if featureID == "adhoc" && os.Getenv("ATRAKTA_REQUIRE_FEATURE") == "1" {
		next := model.NextAction{Kind: "needs_input", Hint: "feature_id is required", Command: "atrakta start --feature-id <id>"}
		step := model.StepEvent{ActorRole: "initializer", TaskID: "adhoc", Outcome: "NEEDS_INPUT", Gate: model.GateResult{Safety: model.GatePass, Quick: model.GateSkip}, NextAction: next}
		_, _ = events.Append(repoRoot, "step", "initializer", structToMap(step))
		cp.Stage = "needs_input"
		cp.Outcome = "needs_input"
		cp.Reason = "feature_id is required"
		writeCheckpointBestEffort(repoRoot, ad, cp)
		ad.PresentNextAction(next)
		return StartResult{Step: step}, nil
	}
	if pgr.ActiveFeature != nil && *pgr.ActiveFeature != "" && *pgr.ActiveFeature != featureID {
		next := model.NextAction{Kind: "blocked", Hint: "feature switch requires explicit reset", Command: "atrakta doctor"}
		step := model.StepEvent{ActorRole: "worker", TaskID: featureID, Outcome: "BLOCKED", Gate: model.GateResult{Safety: model.GatePass, Quick: model.GateSkip}, NextAction: next}
		_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
		cp.Stage = "blocked"
		cp.Outcome = "blocked"
		cp.Reason = "feature switch blocked"
		writeCheckpointBestEffort(repoRoot, ad, cp)
		ad.NotifyBlocked("feature switch blocked")
		ad.PresentNextAction(next)
		return StartResult{Step: step}, fmt.Errorf("blocked: active feature mismatch")
	}
	if featureID != "adhoc" && (pgr.ActiveFeature == nil || *pgr.ActiveFeature == "") {
		pgr.ActiveFeature = &featureID
		if err := progress.Save(repoRoot, pgr); err != nil {
			return StartResult{}, err
		}
	}

	level := syncpolicy.ParseLevel(flags.SyncLevel)
	if flags.SyncLevel == "" {
		level = syncpolicy.ParseLevel(os.Getenv("ATRAKTA_SYNC_LEVEL"))
	}
	if level == syncpolicy.Level1 {
		sp, _, err := syncpolicy.ProposeFromAGENTS(c, sourceAGENTS)
		if err == nil && sp.Needed {
			_, _ = events.Append(repoRoot, "intent", "doctor", map[string]any{
				"text":  sp.Summary,
				"tag":   "ops",
				"sync":  sp,
				"level": 1,
			})
			ad.EmitStatus("sync proposal available (doctor --sync-proposal)")
		}
	}

	st, _, loadErr := state.LoadOrEmpty(repoRoot, contractHash)
	if loadErr != nil {
		rebuilt, err := doctor.RebuildStateFromEvents(repoRoot)
		if err != nil {
			ad.NotifyBlocked("state.json corrupted and rebuild failed")
			return StartResult{}, err
		}
		rebuilt.ContractHash = contractHash
		if err := state.Save(repoRoot, rebuilt); err != nil {
			ad.NotifyBlocked("failed to save rebuilt state")
			return StartResult{}, err
		}
		st = rebuilt
	}

	reg := registry.ApplyOverrides(registry.Default(), c)
	autoState, autoStateErr := ifaceauto.Load(repoRoot)
	if autoStateErr != nil {
		ad.EmitStatus("auto interface state unavailable; continuing with fresh resolution")
		autoState = ifaceauto.Empty()
	}

	explicit := parseExplicit(flags.Interfaces)
	if len(explicit) == 0 {
		explicit = parseExplicit(os.Getenv("ATRAKTA_INTERFACES"))
	}
	resolutionSource := "explicit"
	effectiveExplicit := explicit
	if len(effectiveExplicit) == 0 {
		resolutionSource = ""
		if trigger := strings.TrimSpace(os.Getenv("ATRAKTA_TRIGGER_INTERFACE")); trigger != "" {
			if _, ok := reg.Entries[trigger]; ok {
				effectiveExplicit = []string{trigger}
				resolutionSource = "trigger"
			}
		}
	}
	runDetect := func(explicitList []string) (model.DetectResult, error) {
		return detect.Run(detect.Input{
			RepoRoot: repoRoot,
			Contract: c,
			Registry: reg,
			State:    st,
			Explicit: explicitList,
			StrongS1Validator: func(path string, rec state.ManagedRecord) bool {
				exp := proof.Expected{Fingerprint: rec.Fingerprint, TemplateID: rec.TemplateID, Target: rec.Target, SourceText: sourceAGENTS}
				if exp.Target == "" {
					exp.Target = "AGENTS.md"
				}
				return proof.Revalidate(repoRoot, path, rec, exp) == nil
			},
		})
	}
	det, err := runDetect(effectiveExplicit)
	if err != nil {
		ad.NotifyBlocked(err.Error())
		cp.Stage = "blocked"
		cp.Outcome = "blocked"
		cp.Reason = err.Error()
		writeCheckpointBestEffort(repoRoot, ad, cp)
		return StartResult{}, err
	}
	if resolutionSource == "trigger" {
		det.PruneAllowed = false
		det.Reason = model.ReasonTriggered
		if det.Signals == nil {
			det.Signals = map[string]any{}
		}
		det.Signals["trigger_source"] = strings.TrimSpace(os.Getenv("ATRAKTA_TRIGGER_SOURCE"))
		det.Signals["trigger_interface"] = append([]string{}, det.TargetSet...)
	} else if resolutionSource == "auto_last" {
		det.PruneAllowed = false
		det.Reason = model.ReasonAutoLast
		if det.Signals == nil {
			det.Signals = map[string]any{}
		}
		det.Signals["auto_last"] = append([]string{}, det.TargetSet...)
	}
	if len(explicit) == 0 && resolutionSource == "" && det.Reason == model.ReasonUnknown {
		if last, ok := ifaceauto.LastSingleTarget(autoState, interfaceSetFromRegistry(reg)); ok {
			det, err = runDetect([]string{last})
			if err != nil {
				ad.NotifyBlocked(err.Error())
				cp.Stage = "blocked"
				cp.Outcome = "blocked"
				cp.Reason = err.Error()
				writeCheckpointBestEffort(repoRoot, ad, cp)
				return StartResult{}, err
			}
			det.PruneAllowed = false
			det.Reason = model.ReasonAutoLast
			if det.Signals == nil {
				det.Signals = map[string]any{}
			}
			det.Signals["auto_last"] = append([]string{}, det.TargetSet...)
			resolutionSource = "auto_last"
		}
	}
	if len(explicit) == 0 && resolutionSource == "" && det.Reason == model.ReasonUnknown {
		prompt, schema := interfacePromptPayload(reg)
		resp := adapter.InputResponse{}
		if shouldPromptForInterface() {
			resp = ad.RequestInput(prompt, schema)
		}
		if resp.Value != nil {
			entered := parseExplicit(*resp.Value)
			filtered := filterKnownInterfaces(entered, reg)
			if len(filtered) > 0 {
				det, err = runDetect(filtered)
				if err != nil {
					ad.NotifyBlocked(err.Error())
					cp.Stage = "blocked"
					cp.Outcome = "blocked"
					cp.Reason = err.Error()
					writeCheckpointBestEffort(repoRoot, ad, cp)
					return StartResult{}, err
				}
				resolutionSource = "prompt"
			}
		}
	}
	if len(explicit) == 0 && resolutionSource == "" && det.Reason == model.ReasonUnknown {
		next := model.NextAction{
			Kind:    "needs_input",
			Hint:    "interface is required for first run",
			Command: "atrakta start --interfaces <id>",
		}
		step := model.StepEvent{
			ActorRole:  "initializer",
			TaskID:     featureID,
			Outcome:    "NEEDS_INPUT",
			Gate:       model.GateResult{Safety: model.GatePass, Quick: model.GateSkip},
			NextAction: next,
		}
		_, _ = events.Append(repoRoot, "step", "initializer", structToMap(step))
		cp.Stage = "needs_input"
		cp.Outcome = "needs_input"
		cp.Reason = "interface is required for first run"
		writeCheckpointBestEffort(repoRoot, ad, cp)
		ad.PresentNextAction(next)
		return StartResult{Detect: det, Step: step}, nil
	}
	cp.Interfaces = strings.Join(det.TargetSet, ",")
	cp.DetectReason = string(det.Reason)
	cp.Stage = "detect_done"
	cp.Outcome = "running"
	cp.Reason = ""
	writeCheckpointBestEffort(repoRoot, ad, cp)
	if _, err := events.Append(repoRoot, "detect", "kernel", structToMap(det)); err != nil {
		return StartResult{}, err
	}

	projections, err := projection.RequiredForTargets(repoRoot, c, reg, det.TargetSet, contractHash, sourceAGENTS)
	if err != nil {
		ad.NotifyBlocked(err.Error())
		return StartResult{}, err
	}
	swPlan := subworker.BuildPhaseA(det, projections, subworker.ResolveConfig(c))
	swPlan, budgetReport := subworker.ApplyBudgetGuard(swPlan, c.TokenBudget)
	initialEvents := []events.AppendInput{
		{Type: "context_budget", Actor: "orchestrator", Payload: structToMap(budgetReport)},
		{Type: "subworker_decision", Actor: "orchestrator", Payload: map[string]any{
			"mode":    swPlan.Decision.Mode,
			"enabled": swPlan.Decision.Enabled,
			"reason":  swPlan.Decision.Reason,
			"signals": swPlan.Decision.Signals,
		}},
	}
	if _, err := events.AppendBatch(repoRoot, initialEvents); err != nil {
		return StartResult{}, err
	}
	if budgetReport.Applied {
		ad.EmitStatus(fmt.Sprintf("context budget guard: %s (%d -> %d)", budgetReport.Reason, budgetReport.BeforeTokenEstimate, budgetReport.AfterTokenEstimate))
	}
	if budgetReport.Disabled {
		ad.EmitStatus("subworker disabled by budget guard; falling back to single-writer baseline")
	}
	if swPlan.Decision.Enabled {
		dispatchEvents := []events.AppendInput{{
			Type:  "subworker_dispatch",
			Actor: "orchestrator",
			Payload: map[string]any{
				"phase":      "A",
				"mode":       "read_only",
				"task_count": len(swPlan.Tasks),
				"config":     swPlan.Config,
				"tasks":      swPlan.Tasks,
			},
		}}
		for _, r := range swPlan.Results {
			dispatchEvents = append(dispatchEvents, events.AppendInput{
				Type:    "subworker_result",
				Actor:   "subworker/" + r.WorkerID,
				Payload: structToMap(r),
			})
		}
		if _, err := events.AppendBatch(repoRoot, dispatchEvents); err != nil {
			return StartResult{}, err
		}
		ad.EmitStatus(fmt.Sprintf("subworker enabled: %d task(s), mode=%s, reason=%s", len(swPlan.Tasks), swPlan.Decision.Mode, swPlan.Decision.Reason))
	}
	mergedProjections, mergeReport, err := subworker.MergePhaseA(swPlan, projections)
	if err != nil {
		return StartResult{}, err
	}
	queue := subworker.BuildSingleWriterQueue(swPlan, mergedProjections, mergeReport.UsedFallback)
	if err := subworker.ValidateSingleWriterQueue(queue); err != nil {
		ad.EmitStatus("single writer queue fallback: validation failed, reverting to baseline projections")
		queue = subworker.BuildSingleWriterQueue(subworker.Plan{}, projections, true)
		if qerr := subworker.ValidateSingleWriterQueue(queue); qerr != nil {
			return StartResult{}, qerr
		}
	}
	postMergeEvents := []events.AppendInput{
		{Type: "subworker_merge", Actor: "orchestrator", Payload: structToMap(mergeReport)},
		{Type: "single_writer_queue", Actor: "orchestrator", Payload: structToMap(queue)},
	}
	if mergeReport.BranchEnabled {
		postMergeEvents = append(postMergeEvents, events.AppendInput{
			Type:  "subworker_branch_plan",
			Actor: "orchestrator",
			Payload: map[string]any{
				"mode":     swPlan.Config.BranchMode,
				"reason":   mergeReport.BranchReason,
				"branches": mergeReport.Branches,
			},
		})
		ad.EmitStatus(fmt.Sprintf("branch lanes planned: %d (mode=%s)", len(mergeReport.Branches), swPlan.Config.BranchMode))
	}
	if _, err := events.AppendBatch(repoRoot, postMergeEvents); err != nil {
		return StartResult{}, err
	}
	if mergeReport.UsedFallback {
		ad.EmitStatus("subworker merge fallback: single-writer projection set retained")
	}
	metrics := subworker.AggregateMetrics(swPlan)

	plannedProjections := subworker.QueueProjections(queue)
	pl, err := plan.Build(plan.Input{RepoRoot: repoRoot, Contract: c, Detect: det, State: st, FeatureID: featureID, Projections: plannedProjections})
	if err != nil {
		cp.Stage = "blocked"
		cp.Outcome = "blocked"
		cp.Reason = err.Error()
		writeCheckpointBestEffort(repoRoot, ad, cp)
		return StartResult{}, err
	}
	cp.PlanID = pl.ID
	cp.TaskGraphID = pl.TaskGraphID
	cp.Stage = "plan_done"
	cp.Outcome = "running"
	cp.Reason = ""
	writeCheckpointBestEffort(repoRoot, ad, cp)
	graph, err := taskgraph.GraphFromOps(pl.ID, pl.Ops)
	if err != nil {
		return StartResult{}, err
	}
	if err := taskgraph.Save(repoRoot, graph); err != nil {
		return StartResult{}, err
	}
	if _, err := events.Append(repoRoot, "task_graph_planned", "orchestrator", map[string]any{
		"graph_id":   graph.GraphID,
		"plan_id":    graph.PlanID,
		"task_count": graph.TaskCount,
		"edge_count": graph.EdgeCount,
		"digest":     graph.Digest,
	}); err != nil {
		return StartResult{}, err
	}
	if _, err := events.Append(repoRoot, "plan", "kernel", map[string]any{
		"id":                  pl.ID,
		"task_graph_id":       pl.TaskGraphID,
		"task_count":          pl.TaskCount,
		"task_edge_count":     pl.TaskEdgeCount,
		"feature_id":          pl.FeatureID,
		"required_permission": pl.RequiredPermission,
		"ops":                 pl.Ops,
	}); err != nil {
		return StartResult{}, err
	}

	presentSummary := pl.Summary
	presentDetails := pl.Details
	if c.Policies != nil && c.Policies.PromptMin != nil {
		pol, err := policy.LoadPromptMin(repoRoot, *c.Policies.PromptMin)
		if err != nil {
			if os.IsNotExist(err) && !c.Policies.PromptMin.Required {
				ad.EmitStatus("prompt policy missing; continuing without prompt-min")
			} else {
				ad.NotifyBlocked("prompt policy load failed")
				return StartResult{}, err
			}
		} else if policy.ShouldApplyPromptMin(routeDecision.TaskCategory, false, pol) {
			s, d, applied := policy.ApplyGoalPrefix(presentSummary, presentDetails, pol)
			if applied {
				presentSummary, presentDetails = s, d
				if _, err := events.Append(repoRoot, "policy_applied", "orchestrator", map[string]any{
					"policy_id":     pol.ID,
					"ref":           c.Policies.PromptMin.Ref,
					"task_category": routeDecision.TaskCategory,
				}); err != nil {
					return StartResult{}, err
				}
			}
		}
	}
	ad.PresentDiff(presentSummary, presentDetails)
	approved := true
	if pl.RequiresApproval {
		resp := ad.RequestApproval(pl.ApprovalContext)
		approved = resp.Approved
		if !approved {
			next := model.NextAction{Kind: "needs_approval", Hint: "approval required to continue", Command: "atrakta start"}
			step := model.StepEvent{ActorRole: "worker", TaskID: featureID, Outcome: "NEEDS_APPROVAL", Gate: model.GateResult{Safety: model.GatePass, Quick: model.GateSkip}, NextAction: next}
			_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
			cp.Stage = "needs_approval"
			cp.Outcome = "needs_approval"
			cp.Reason = "approval required to continue"
			writeCheckpointBestEffort(repoRoot, ad, cp)
			ad.PresentNextAction(next)
			return StartResult{Detect: det, Plan: pl, Step: step}, nil
		}
	}
	if reason := preflightSecurityPolicy(c, pl); reason != "" {
		gt := model.GateResult{Safety: model.GateFail, Quick: model.GateSkip, Reason: reason}
		next := model.NextAction{Kind: "blocked", Hint: reason, Command: "atrakta doctor"}
		step := model.StepEvent{ActorRole: "worker", TaskID: featureID, Outcome: "BLOCKED", Gate: gt, NextAction: next}
		_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
		cp.Stage = "blocked"
		cp.Outcome = "blocked"
		cp.Reason = reason
		writeCheckpointBestEffort(repoRoot, ad, cp)
		ad.NotifyBlocked(reason)
		ad.PresentNextAction(next)
		return StartResult{Detect: det, Plan: pl, Gate: gt, Step: step}, fmt.Errorf("blocked: %s", reason)
	}

	parallelMode, parallelWorkers := resolveApplyParallelSettings(queue, mergeReport)
	ap := apply.Run(apply.Input{
		RepoRoot:           repoRoot,
		Contract:           c,
		ContractHash:       contractHash,
		State:              st,
		Plan:               pl,
		Approved:           approved,
		DetectReason:       det.Reason,
		SourceAGENTS:       sourceAGENTS,
		ParallelMode:       parallelMode,
		ParallelMaxWorkers: parallelWorkers,
	})
	if _, err := events.Append(repoRoot, "apply", "worker", structToMap(ap)); err != nil {
		return StartResult{}, err
	}
	cp.ApplyResult = ap.Result
	cp.Stage = "apply_done"
	cp.Outcome = "running"
	cp.Reason = ""
	writeCheckpointBestEffort(repoRoot, ad, cp)
	if ap.Result == "fail" {
		gt := model.GateResult{Safety: model.GatePass, Quick: model.GateSkip, Reason: "apply failed"}
		next := model.NextAction{Kind: "blocked", Hint: "apply failed", Command: "atrakta doctor"}
		step := model.StepEvent{ActorRole: "worker", TaskID: featureID, Outcome: "BLOCKED", Gate: gt, NextAction: next}
		_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
		cp.Stage = "apply_failed"
		cp.Outcome = "blocked"
		cp.Reason = "apply failed"
		writeCheckpointBestEffort(repoRoot, ad, cp)
		ad.NotifyBlocked("apply failed")
		ad.PresentNextAction(next)
		return StartResult{Detect: det, Plan: pl, Apply: ap, Gate: gt, Step: step}, fmt.Errorf("blocked: apply failed")
	}

	gt := gate.Run(gate.Input{
		RepoRoot:                repoRoot,
		Contract:                c,
		Detect:                  det,
		Plan:                    pl,
		Apply:                   ap,
		Approved:                approved,
		Registry:                reg,
		Quality:                 c.Quality,
		FeatureID:               featureID,
		Progress:                pgr,
		Strict:                  level == syncpolicy.Level2 || os.Getenv("ATRAKTA_STRICT") == "1",
		RouteQuality:            routeDecision.Quality,
		SubworkerEnabled:        metrics.Enabled,
		SubworkerTokenEstimate:  metrics.TotalTokenEstimate,
		SubworkerMaxDigestChars: metrics.MaxDigestCharsUsed,
		SubworkerMaxOutputChars: metrics.MaxSummaryCharsUsed,
	})
	if gt.Safety == model.GateFail || gt.Quick == model.GateFail {
		next := model.NextAction{Kind: "blocked", Hint: gt.Reason, Command: "atrakta doctor"}
		step := model.StepEvent{ActorRole: "worker", TaskID: featureID, Outcome: "BLOCKED", Gate: gt, NextAction: next}
		_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
		cp.Stage = "gate_failed"
		cp.Outcome = "blocked"
		cp.Reason = gt.Reason
		writeCheckpointBestEffort(repoRoot, ad, cp)
		ad.NotifyBlocked(gt.Reason)
		ad.PresentNextAction(next)
		return StartResult{Detect: det, Plan: pl, Apply: ap, Gate: gt, Step: step}, fmt.Errorf("blocked: %s", gt.Reason)
	}

	st2 := state.UpdateFromApply(st, contractHash, toStateApply(ap))
	manifestResult, manifestErr := manifest.UpdateFromApply(repoRoot, ap, contractHash)
	if manifestErr != nil {
		now := util.NowUTC()
		st2.Integration = &state.IntegrationState{
			LastCheckedAt:   now,
			LastResult:      "blocked",
			BlockingReasons: []string{"manifest update failed: " + manifestErr.Error()},
		}
		_ = state.Save(repoRoot, st2)
		_, _ = events.Append(repoRoot, events.EventIntegrationBlocked, "orchestrator", map[string]any{
			"feature_id": featureID,
			"reason":     "manifest_update_failed",
			"error":      manifestErr.Error(),
		})
		return StartResult{}, fmt.Errorf("update manifests: %w", manifestErr)
	}
	now := util.NowUTC()
	st2.Projection = &state.ProjectionState{
		LastRenderedAt: now,
		SourceHash:     manifestResult.SourceHash,
		RenderHash:     manifestResult.RenderHash,
		Status:         "ok",
	}
	st2.Integration = &state.IntegrationState{
		LastCheckedAt: now,
		LastResult:    "ok",
	}
	if err := state.Save(repoRoot, st2); err != nil {
		return StartResult{}, err
	}
	if _, err := events.Append(repoRoot, events.EventProjectionRendered, "orchestrator", map[string]any{
		"feature_id":         featureID,
		"source_hash":        manifestResult.SourceHash,
		"render_hash":        manifestResult.RenderHash,
		"projection_entries": manifestResult.ProjectionEntries,
		"extension_entries":  manifestResult.ExtensionEntries,
	}); err != nil {
		return StartResult{}, err
	}
	if _, err := events.Append(repoRoot, events.EventIntegrationChecked, "orchestrator", map[string]any{
		"feature_id": featureID,
		"result":     "ok",
	}); err != nil {
		return StartResult{}, err
	}
	if featureID != "adhoc" {
		pgr.ActiveFeature = nil
		if !progress.ContainsFeature(pgr.CompletedFeatures, featureID) {
			pgr.CompletedFeatures = append(pgr.CompletedFeatures, featureID)
		}
		if err := progress.Save(repoRoot, pgr); err != nil {
			return StartResult{}, err
		}
	}
	gitPost := gitauto.Capture(repoRoot)
	if _, err := events.Append(repoRoot, "git_snapshot", "orchestrator", map[string]any{
		"phase":    "post",
		"snapshot": structToMap(gitPost),
	}); err != nil {
		return StartResult{}, err
	}
	if cp, wrote, err := gitauto.WriteCheckpoint(repoRoot, featureID, gitMode, gitPre, gitPost, pl, ap, gt); err != nil {
		return StartResult{}, err
	} else if wrote {
		if _, err := events.Append(repoRoot, "git_checkpoint", "orchestrator", structToMap(cp)); err != nil {
			return StartResult{}, err
		}
		ad.EmitStatus(fmt.Sprintf("git checkpoint: mode=%s reason=%s", cp.Mode, cp.Reason))
	}

	next := model.NextAction{Kind: "done", Hint: "completed", Command: "atrakta start"}
	step := model.StepEvent{ActorRole: "worker", TaskID: featureID, Outcome: "DONE", Gate: gt, NextAction: next}
	_, _ = events.Append(repoRoot, "step", "worker", structToMap(step))
	cp.Stage = "done"
	cp.Outcome = "done"
	cp.Reason = ""
	writeCheckpointBestEffort(repoRoot, ad, cp)
	ad.PresentNextAction(next)

	autoState = ifaceauto.Record(autoState, det.TargetSet, autoResolutionSource(resolutionSource, det.Reason))
	if err := ifaceauto.Save(repoRoot, autoState); err != nil {
		ad.EmitStatus("auto interface state save failed: " + err.Error())
	} else if stale := ifaceauto.SuggestStale(autoState, det.TargetSet, staleSuggestionWindow(), time.Now().UTC()); len(stale) > 0 {
		ad.EmitStatus("stale interface candidates: " + strings.Join(stale, ",") + " (proposal only; no auto-delete)")
	}
	if err := startfast.SaveSuccess(repoRoot, startfast.Input{
		ContractHash:   contractHash,
		WorkspaceStamp: util.WorkspaceStamp(repoRoot, c.Boundary.Include),
		Interfaces:     canonicalInterfaceKey(det.TargetSet),
		FeatureID:      featureID,
		ConfigKey:      startFastConfigKey(resolvedSyncLevel, mapTokens, mapRefresh, taskCategory),
	}, string(det.Reason), time.Now().UTC()); err != nil {
		ad.EmitStatus("start fast snapshot save failed: " + err.Error())
	}

	return StartResult{Detect: det, Plan: pl, Apply: ap, Gate: gt, Step: step}, nil
}

func parseExplicit(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func resolveRepoMapSettings(c contract.Contract, flags StartFlags) (int, int) {
	mapTokens := 0
	mapRefresh := 0
	if c.Context != nil {
		mapTokens = c.Context.RepoMapTokens
		mapRefresh = c.Context.RepoMapRefreshSec
	}
	if flags.MapTokens > 0 {
		mapTokens = flags.MapTokens
	}
	if flags.MapRefresh > 0 {
		mapRefresh = flags.MapRefresh
	}
	return mapTokens, mapRefresh
}

func resolveFastPathInterfaces(flags StartFlags) ([]string, model.DetectReason) {
	explicit := parseExplicit(flags.Interfaces)
	if len(explicit) == 0 {
		explicit = parseExplicit(os.Getenv("ATRAKTA_INTERFACES"))
	}
	if len(explicit) > 0 {
		return explicit, model.ReasonExplicit
	}
	if trigger := strings.TrimSpace(os.Getenv("ATRAKTA_TRIGGER_INTERFACE")); trigger != "" {
		return []string{trigger}, model.ReasonTriggered
	}
	return nil, model.ReasonUnknown
}

func canonicalInterfaceKey(in []string) string {
	if len(in) == 0 {
		return ""
	}
	cp := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, v := range in {
		n := strings.TrimSpace(v)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		cp = append(cp, n)
	}
	sort.Strings(cp)
	return strings.Join(cp, ",")
}

func startFastConfigKey(syncLevel string, mapTokens, mapRefresh int, taskCategory string) string {
	return strings.TrimSpace(syncLevel) + "|" +
		strconv.Itoa(mapTokens) + "|" +
		strconv.Itoa(mapRefresh) + "|" +
		strings.TrimSpace(taskCategory)
}

func managedArtifactsIntact(repoRoot string) bool {
	st, _, err := state.LoadOrEmpty(repoRoot, "")
	if err != nil {
		return false
	}
	for p := range st.ManagedPaths {
		abs := filepath.Join(repoRoot, filepath.FromSlash(p))
		if _, err := os.Stat(abs); err != nil {
			return false
		}
	}
	return true
}

func shouldPromptForInterface() bool {
	if strings.TrimSpace(os.Getenv("ATRAKTA_NONINTERACTIVE")) == "1" {
		return false
	}
	return strings.TrimSpace(os.Getenv("ATRAKTA_TRIGGER_SOURCE")) == ""
}

func interfaceSetFromRegistry(reg registry.Registry) map[string]struct{} {
	out := make(map[string]struct{}, len(reg.Entries))
	for id := range reg.Entries {
		out[id] = struct{}{}
	}
	return out
}

func filterKnownInterfaces(in []string, reg registry.Registry) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, id := range in {
		if _, ok := reg.Entries[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func interfacePromptPayload(reg registry.Registry) (string, map[string]any) {
	ids := reg.InterfaceIDs()
	hint := "interface id required (choose from: " + strings.Join(ids, ",") + ")"
	return hint, map[string]any{
		"field":     "interfaces",
		"supported": ids,
		"example":   "cursor",
	}
}

func staleSuggestionWindow() time.Duration {
	days := parseIntOrZero(os.Getenv("ATRAKTA_STALE_INTERFACE_DAYS"))
	if days == 0 {
		days = 30
	}
	if days < 0 {
		return 0
	}
	return time.Duration(days) * 24 * time.Hour
}

func autoResolutionSource(source string, reason model.DetectReason) string {
	s := strings.TrimSpace(source)
	if s != "" {
		return s
	}
	return string(reason)
}

func resolveApplyParallelSettings(queue subworker.IntegrationQueue, merge subworker.MergeReport) (mode string, maxWorkers int) {
	rawMode := strings.TrimSpace(strings.ToLower(os.Getenv("ATRAKTA_PARALLEL_APPLY")))
	switch rawMode {
	case "on", "auto", "off":
		mode = rawMode
	default:
		// Keep default off; promote to auto only when the single-writer queue proves a safe workload.
		mode = "off"
		if queue.NonConflicting && queue.ItemCount >= 4 {
			mode = "auto"
		} else if merge.BranchEnabled {
			mode = "auto"
		}
	}
	maxWorkers = 4
	if n := parseIntOrZero(os.Getenv("ATRAKTA_PARALLEL_APPLY_MAX_WORKERS")); n > 0 {
		maxWorkers = n
	}
	return mode, maxWorkers
}

func parseIntOrZero(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	return n
}

func resolveTaskCategory() string {
	c := strings.TrimSpace(strings.ToLower(os.Getenv("ATRAKTA_TASK_CATEGORY")))
	if c == "" {
		return "sync"
	}
	return c
}

func preflightSecurityPolicy(c contract.Contract, pl model.PlanResult) string {
	required := pl.RequiredPermission
	if required == "" {
		required = requiredPermissionFromOps(pl.Ops)
	}
	if contract.ResolveSecurityProfile(c) == string(model.PermissionReadOnly) && required != model.PermissionReadOnly {
		return "security profile read_only blocks filesystem mutations"
	}
	return ""
}

func requiredPermissionFromOps(ops []model.Operation) model.Permission {
	for _, op := range ops {
		switch op.Op {
		case "adopt", "link", "copy", "write", "delete", "unlink":
			return model.PermissionWorkspaceWrite
		}
	}
	return model.PermissionReadOnly
}

func structToMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(b, &out)
	return out
}

func writeCheckpointBestEffort(repoRoot string, ad adapter.Adapter, cp checkpoint.RunCheckpoint) {
	if err := checkpoint.SaveLatest(repoRoot, cp); err != nil {
		ad.EmitStatus("run checkpoint save failed: " + err.Error())
	}
}

func toStateApply(ap model.ApplyResult) state.ApplyResult {
	out := state.ApplyResult{Ops: make([]state.ApplyOpResult, 0, len(ap.Ops))}
	for _, r := range ap.Ops {
		out.Ops = append(out.Ops, state.ApplyOpResult{
			TaskID:      r.TaskID,
			Path:        r.Path,
			Op:          r.Op,
			Status:      r.Status,
			Error:       r.Error,
			Interface:   r.Interface,
			TemplateID:  r.TemplateID,
			Kind:        r.Kind,
			Target:      r.Target,
			Fingerprint: r.Fingerprint,
		})
	}
	return out
}
