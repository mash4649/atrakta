package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/bootstrap"
	agentsctx "atrakta/internal/context"
	"atrakta/internal/contract"
	"atrakta/internal/events"
	"atrakta/internal/model"
	"atrakta/internal/policy"
	"atrakta/internal/progress"
	"atrakta/internal/proof"
	"atrakta/internal/state"
	"atrakta/internal/taskgraph"
)

type Report struct {
	Outcome    string
	Reason     string
	NextAction model.NextAction
	Rebuilt    bool
	Repairs    []string
}

func Run(repoRoot string, sourceAGENTS string) (Report, state.State, error) {
	repairs := []string{}
	if err := events.VerifyChain(repoRoot); err != nil {
		return Report{
			Outcome: "BLOCKED",
			Reason:  "events integrity failed",
			NextAction: model.NextAction{
				Kind:    "blocked",
				Hint:    "repair_event_log",
				Command: "atrakta doctor",
			},
		}, state.State{}, err
	}
	c, _, err := contract.LoadOrInit(repoRoot)
	if err != nil {
		return Report{
			Outcome: "BLOCKED",
			Reason:  "contract validation failed",
			NextAction: model.NextAction{
				Kind:    "blocked",
				Hint:    "fix .atrakta/contract.json",
				Command: "atrakta doctor",
			},
		}, state.State{}, err
	}
	c = contract.CanonicalizeBoundary(c)
	if c.Policies != nil && c.Policies.PromptMin != nil {
		if _, err := policy.EnsureDefaultPromptMin(repoRoot, c.Policies.PromptMin.Ref); err != nil {
			return Report{
				Outcome: "BLOCKED",
				Reason:  "prompt policy initialization failed",
				NextAction: model.NextAction{
					Kind:    "blocked",
					Hint:    "repair .atrakta/policies/prompt-min.json",
					Command: "atrakta doctor",
				},
			}, state.State{}, err
		}
		_, err := policy.LoadPromptMin(repoRoot, *c.Policies.PromptMin)
		if err != nil {
			if os.IsNotExist(err) && !c.Policies.PromptMin.Required {
				// optional policy not present
			} else {
				return Report{
					Outcome: "BLOCKED",
					Reason:  "prompt policy invalid",
					NextAction: model.NextAction{
						Kind:    "blocked",
						Hint:    "fix prompt policy file",
						Command: "atrakta doctor",
					},
				}, state.State{}, err
			}
		}
	}
	if _, _, err := bootstrap.EnsureRootAGENTS(repoRoot); err != nil {
		return Report{
			Outcome: "BLOCKED",
			Reason:  "AGENTS initialization failed",
			NextAction: model.NextAction{
				Kind:    "blocked",
				Hint:    "repair AGENTS.md",
				Command: "atrakta doctor",
			},
		}, state.State{}, err
	}
	resolvedAGENTS, _, err := agentsctx.Resolve(agentsctx.ResolveInput{
		RepoRoot: repoRoot,
		StartDir: repoRoot,
		Config:   c.Context,
	})
	if err != nil {
		return Report{
			Outcome: "BLOCKED",
			Reason:  "context resolution failed",
			NextAction: model.NextAction{
				Kind:    "blocked",
				Hint:    "fix AGENTS import chain",
				Command: "atrakta doctor",
			},
		}, state.State{}, err
	}
	sourceAGENTS = resolvedAGENTS
	if _, _, err := taskgraph.Load(repoRoot); err != nil {
		return Report{
			Outcome: "BLOCKED",
			Reason:  "task graph invalid",
			NextAction: model.NextAction{
				Kind:    "blocked",
				Hint:    "repair .atrakta/task-graph.json",
				Command: "atrakta doctor",
			},
		}, state.State{}, err
	}

	s, existed, loadErr := state.LoadOrEmpty(repoRoot, "")
	if loadErr != nil || !existed {
		rebuilt, err := RebuildStateFromEvents(repoRoot)
		if err != nil {
			return Report{Outcome: "BLOCKED", Reason: "state rebuild failed", NextAction: model.NextAction{Kind: "blocked", Hint: "repair state from events"}}, state.State{}, err
		}
		if err := state.Save(repoRoot, rebuilt); err != nil {
			return Report{Outcome: "BLOCKED", Reason: "state save failed", NextAction: model.NextAction{Kind: "blocked", Hint: "save rebuilt state"}}, state.State{}, err
		}
		s = rebuilt
		repairs = append(repairs, "state_rebuilt")
	}

	for p, rec := range s.ManagedPaths {
		exp := proof.Expected{Fingerprint: rec.Fingerprint, Target: rec.Target, TemplateID: rec.TemplateID, SourceText: sourceAGENTS}
		if exp.Target == "" {
			exp.Target = "AGENTS.md"
		}
		if err := proof.Revalidate(repoRoot, p, rec, exp); err != nil {
			return Report{
				Outcome: "BLOCKED",
				Reason:  "projection drift: " + p,
				NextAction: model.NextAction{
					Kind:    "blocked",
					Hint:    "run start to generate explicit repair plan",
					Command: "atrakta start",
				},
			}, s, fmt.Errorf("drift detected: %w", err)
		}
	}
	if _, _, err := progress.LoadOrInit(repoRoot); err != nil {
		rebuilt, rebuildErr := RebuildProgressFromEvents(repoRoot)
		if rebuildErr != nil {
			return Report{
				Outcome: "BLOCKED",
				Reason:  "progress integrity failed",
				NextAction: model.NextAction{
					Kind:    "blocked",
					Hint:    "repair progress.json",
					Command: "atrakta doctor",
				},
			}, s, err
		}
		if saveErr := progress.Save(repoRoot, rebuilt); saveErr != nil {
			return Report{
				Outcome: "BLOCKED",
				Reason:  "progress save failed",
				NextAction: model.NextAction{
					Kind:    "blocked",
					Hint:    "save rebuilt progress.json",
					Command: "atrakta doctor",
				},
			}, s, saveErr
		}
		repairs = append(repairs, "progress_rebuilt")
	}

	report := Report{
		Outcome:    "PROGRESSED",
		Reason:     "ok",
		NextAction: model.NextAction{Kind: "run_start_again", Hint: "system healthy"},
		Repairs:    repairs,
	}
	if len(repairs) > 0 {
		report.Rebuilt = true
		report.Reason = "repaired: " + strings.Join(repairs, ",")
	}
	return report, s, nil
}

func RebuildStateFromEvents(repoRoot string) (state.State, error) {
	ev, err := events.ReadAll(repoRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state.Empty(""), nil
		}
		return state.State{}, err
	}
	out := state.Empty("")
	for _, e := range ev {
		t, _ := e.Raw["type"].(string)
		if t != "apply" {
			continue
		}
		opsAny, ok := e.Raw["ops"].([]any)
		if !ok {
			continue
		}
		for _, opRaw := range opsAny {
			m, ok := opRaw.(map[string]any)
			if !ok {
				continue
			}
			op, _ := m["op"].(string)
			path, _ := m["path"].(string)
			status, _ := m["status"].(string)
			if status != "ok" && status != "skipped" {
				continue
			}
			switch op {
			case "adopt", "link", "copy", "write":
				iface, _ := m["interface"].(string)
				tid, _ := m["template_id"].(string)
				fp, _ := m["fingerprint"].(string)
				kind, _ := m["kind"].(string)
				target, _ := m["target"].(string)
				if path == "" || iface == "" || tid == "" || fp == "" {
					continue
				}
				if kind == "" {
					kind = "copy"
				}
				out.ManagedPaths[path] = state.ManagedRecord{Interface: iface, Kind: kind, Fingerprint: fp, TemplateID: tid, Target: target}
			case "delete", "unlink":
				if path != "" {
					delete(out.ManagedPaths, path)
				}
			}
		}
	}

	if out.ContractHash == "" {
		cp := filepath.Join(repoRoot, ".atrakta", "contract.json")
		if b, err := os.ReadFile(cp); err == nil {
			out.ContractHash = contract.ContractHash(b)
		}
	}
	return out, nil
}

func RebuildProgressFromEvents(repoRoot string) (progress.Progress, error) {
	ev, err := events.ReadAll(repoRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return progress.Empty(), nil
		}
		return progress.Progress{}, err
	}
	out := progress.Empty()
	completed := map[string]struct{}{}
	active := ""
	for _, e := range ev {
		t, _ := e.Raw["type"].(string)
		if t != "step" {
			continue
		}
		taskID, _ := e.Raw["task_id"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" || taskID == "adhoc" {
			continue
		}
		outcome, _ := e.Raw["outcome"].(string)
		switch strings.ToUpper(strings.TrimSpace(outcome)) {
		case "DONE":
			completed[taskID] = struct{}{}
			if active == taskID {
				active = ""
			}
		case "PROGRESSED", "NEEDS_APPROVAL", "NEEDS_INPUT":
			active = taskID
		}
	}
	if len(completed) > 0 {
		out.CompletedFeatures = make([]string, 0, len(completed))
		for f := range completed {
			out.CompletedFeatures = append(out.CompletedFeatures, f)
		}
		sort.Strings(out.CompletedFeatures)
	}
	if active != "" {
		if _, ok := completed[active]; !ok {
			out.ActiveFeature = &active
		}
	}
	return out, nil
}
