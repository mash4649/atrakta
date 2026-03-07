package gate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/events"
	"atrakta/internal/model"
	"atrakta/internal/progress"
	"atrakta/internal/registry"
)

type Input struct {
	RepoRoot     string
	Contract     contract.Contract
	Detect       model.DetectResult
	Plan         model.PlanResult
	Apply        model.ApplyResult
	Approved     bool
	Registry     registry.Registry
	Quality      *contract.Quality
	FeatureID    string
	Progress     progress.Progress
	Strict       bool
	RouteQuality string

	SubworkerEnabled        bool
	SubworkerTokenEstimate  int
	SubworkerMaxDigestChars int
	SubworkerMaxOutputChars int
}

func Run(in Input) model.GateResult {
	if fail := safetyChecks(in); fail != "" {
		return model.GateResult{Safety: model.GateFail, Quick: model.GateSkip, Reason: fail}
	}
	quick, reason := quickChecks(in)
	if quick == model.GateFail {
		return model.GateResult{Safety: model.GatePass, Quick: model.GateFail, Reason: reason}
	}
	return model.GateResult{Safety: model.GatePass, Quick: quick, Reason: reason}
}

func safetyChecks(in Input) string {
	if reason := validateSecurityProfile(in); reason != "" {
		return reason
	}
	if in.Detect.Reason == model.ReasonUnknown || in.Detect.Reason == model.ReasonConflict || in.Detect.Reason == model.ReasonMixed {
		for _, op := range in.Plan.Ops {
			if (op.Op == "delete" || op.Op == "unlink") && op.Reason == "prune_unused" {
				return "prune forbidden under ambiguity"
			}
		}
	}
	for _, op := range in.Plan.Ops {
		if op.RequiresApproval && !in.Approved {
			return "approval required but not granted"
		}
	}
	if reason := validatePlanApplyAlignment(in.Plan.Ops, in.Apply.Ops); reason != "" {
		return reason
	}
	if in.Apply.PlanID != in.Plan.ID {
		return "apply.plan_id mismatch"
	}
	if in.FeatureID != "" && in.Plan.FeatureID != "" && in.FeatureID != in.Plan.FeatureID {
		return "feature_id mismatch between start and plan"
	}
	if in.Progress.ActiveFeature != nil && *in.Progress.ActiveFeature != "" && in.Plan.FeatureID != "" && *in.Progress.ActiveFeature != in.Plan.FeatureID {
		return "active_feature mismatch with plan.feature_id"
	}
	return ""
}

func quickChecks(in Input) (model.GateState, string) {
	if len(in.Apply.Ops) == 0 {
		return model.GateSkip, "no delta"
	}
	if in.Apply.Result != "success" && in.Apply.Result != "partial" {
		return model.GateSkip, "apply failed"
	}
	if reason := validateSubworkerBudget(in); reason != "" {
		return model.GateFail, reason
	}

	checks := []string{"projection_integrity"}
	if in.Quality != nil && len(in.Quality.QuickChecks) > 0 {
		checks = in.Quality.QuickChecks
	}
	for _, chk := range checks {
		switch chk {
		case "projection_integrity":
			for _, id := range in.Detect.TargetSet {
				e, ok := in.Registry.Entries[id]
				if !ok || e.ProjectionDir == "" {
					continue
				}
				p := filepath.Join(in.RepoRoot, filepath.FromSlash(filepath.ToSlash(filepath.Join(e.ProjectionDir, "AGENTS.md"))))
				if _, err := os.Stat(p); err != nil {
					return model.GateFail, "required projection missing: " + id
				}
			}
		case "events_chain":
			if err := events.VerifyChain(in.RepoRoot); err != nil {
				return model.GateFail, "events_chain check failed"
			}
		case "progress_integrity":
			if in.Progress.CompletedFeatures == nil {
				return model.GateFail, "progress_integrity check failed"
			}
		default:
			return model.GateFail, "unknown quick check: " + chk
		}
	}

	heavyByRoute := strings.TrimSpace(strings.ToLower(in.RouteQuality)) == "heavy"
	enableHeavy := in.Strict || heavyByRoute
	heavyChecks := []string{"go_test_compile"}
	if in.Quality != nil {
		enableHeavy = enableHeavy || in.Quality.EnableHeavy
		if len(in.Quality.HeavyChecks) > 0 {
			heavyChecks = in.Quality.HeavyChecks
		}
	}
	if enableHeavy {
		if len(heavyChecks) == 0 {
			heavyChecks = []string{"go_test_compile"}
		}
		for _, chk := range heavyChecks {
			switch chk {
			case "go_test_compile":
				cmd := exec.Command("go", "test", "./...", "-run", "^$")
				cmd.Dir = in.RepoRoot
				env := os.Environ()
				hasCache := false
				for _, kv := range env {
					if strings.HasPrefix(kv, "GOCACHE=") {
						hasCache = true
						break
					}
				}
				if !hasCache {
					env = append(env, "GOCACHE="+filepath.Join(in.RepoRoot, ".tmp", "go-build"))
				}
				cmd.Env = env
				if out, err := cmd.CombinedOutput(); err != nil {
					if errors.Is(err, exec.ErrNotFound) {
						return model.GateFail, "go command not found for heavy check"
					}
					return model.GateFail, "heavy check failed: " + strings.TrimSpace(string(out))
				}
			default:
				return model.GateFail, "unknown heavy check: " + chk
			}
		}
	}
	return model.GatePass, "ok"
}

func validateSubworkerBudget(in Input) string {
	if !in.SubworkerEnabled {
		return ""
	}
	if in.SubworkerTokenEstimate > in.Contract.TokenBudget.Hard {
		return "subworker token budget hard limit exceeded"
	}
	if in.Strict && in.SubworkerTokenEstimate > in.Contract.TokenBudget.Soft {
		return "subworker token budget soft limit exceeded in strict mode"
	}
	if in.SubworkerMaxDigestChars > 512 {
		return "subworker digest chars exceeded policy cap"
	}
	if in.SubworkerMaxOutputChars > 512 {
		return "subworker output chars exceeded policy cap"
	}
	return ""
}

func validateSecurityProfile(in Input) string {
	profile := contract.ResolveSecurityProfile(in.Contract)
	required := in.Plan.RequiredPermission
	if required == "" {
		required = requiredPermissionFromOps(in.Plan.Ops)
	}
	if profile == string(model.PermissionReadOnly) && required != model.PermissionReadOnly {
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

func validatePlanApplyAlignment(planOps []model.Operation, applyOps []model.OpResult) string {
	if len(applyOps) > len(planOps) {
		return "apply has extra ops"
	}
	useTaskIDs := true
	planByTask := map[string]model.Operation{}
	for _, p := range planOps {
		if strings.TrimSpace(p.TaskID) == "" {
			useTaskIDs = false
			break
		}
		planByTask[p.TaskID] = p
	}
	if useTaskIDs {
		for i, r := range applyOps {
			if strings.TrimSpace(r.TaskID) == "" {
				return fmt.Sprintf("apply op %d missing task_id", i)
			}
			p, ok := planByTask[r.TaskID]
			if !ok {
				return fmt.Sprintf("unknown apply task_id at op %d", i)
			}
			if p.Path != r.Path || p.Op != r.Op {
				return fmt.Sprintf("plan/apply mismatch for task %s", r.TaskID)
			}
		}
		return ""
	}
	for i, r := range applyOps {
		if i >= len(planOps) {
			return "apply has extra ops"
		}
		p := planOps[i]
		if p.Path != r.Path || p.Op != r.Op {
			return fmt.Sprintf("plan/apply mismatch at op %d", i)
		}
	}
	return ""
}
