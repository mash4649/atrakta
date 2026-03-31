package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/hook"
	"github.com/mash4649/atrakta/v0/internal/ideautostart"
	"github.com/mash4649/atrakta/v0/internal/wrap"
)

const initDefaultHookType = "pre-commit"

type initSetupSummary struct {
	CompletedSteps int
	NeedsApproval  bool
}

func runInitOnboardingSetup(projectRoot, interfaceID string, noOverwrite, noWrap, noHook, noIDEAutostart, nonInteractive, approve bool) (initSetupSummary, error) {
	summary := initSetupSummary{}

	if !noWrap {
		status, err := runInitWrapStep(projectRoot, interfaceID, noOverwrite, nonInteractive, approve)
		if err != nil {
			return summary, err
		}
		if status == "needs_approval" {
			summary.NeedsApproval = true
			if err := appendInitStepEvent(projectRoot, interfaceID, "wrap.install", status, map[string]any{
				"tool_id": interfaceID,
			}); err != nil {
				return summary, err
			}
			return summary, nil
		}
		if err := appendInitStepEvent(projectRoot, interfaceID, "wrap.install", status, map[string]any{
			"tool_id": interfaceID,
		}); err != nil {
			return summary, err
		}
		if status == "installed" {
			summary.CompletedSteps++
		}
	}

	if !noHook {
		status, err := runInitHookStep(projectRoot, noOverwrite, nonInteractive, approve)
		if err != nil {
			return summary, err
		}
		if status == "needs_approval" {
			summary.NeedsApproval = true
			if err := appendInitStepEvent(projectRoot, interfaceID, "hook.install", status, map[string]any{
				"hook_type": initDefaultHookType,
			}); err != nil {
				return summary, err
			}
			return summary, nil
		}
		if err := appendInitStepEvent(projectRoot, interfaceID, "hook.install", status, map[string]any{
			"hook_type": initDefaultHookType,
		}); err != nil {
			return summary, err
		}
		if status == "installed" {
			summary.CompletedSteps++
		}
	}

	if !noIDEAutostart {
		status, err := runInitIDEAutostartStep(projectRoot, noOverwrite, nonInteractive, approve)
		if err != nil {
			return summary, err
		}
		if status == "needs_approval" {
			summary.NeedsApproval = true
			if err := appendInitStepEvent(projectRoot, interfaceID, "ide-autostart.install", status, nil); err != nil {
				return summary, err
			}
			return summary, nil
		}
		if err := appendInitStepEvent(projectRoot, interfaceID, "ide-autostart.install", status, nil); err != nil {
			return summary, err
		}
		if status == "installed" {
			summary.CompletedSteps++
		}
	}

	return summary, nil
}

func appendInitStepEvent(projectRoot, interfaceID, step, status string, extra map[string]any) error {
	payload := map[string]any{
		"command": "init",
		"step":    step,
		"status":  status,
	}
	for k, v := range extra {
		payload[k] = v
	}
	return appendOperationalRunEvent(projectRoot, runEventInitStep, interfaceID, payload)
}

func runInitWrapStep(projectRoot, toolID string, noOverwrite, nonInteractive, approve bool) (string, error) {
	plan, err := wrap.BuildInstallPlan(projectRoot, toolID)
	if err != nil {
		return "", err
	}

	existing, err := os.ReadFile(plan.InstallPath)
	switch {
	case err == nil && strings.TrimSpace(string(existing)) == strings.TrimSpace(plan.Script):
		return "up_to_date", nil
	case err == nil && noOverwrite:
		return "skipped_existing", nil
	case err != nil && !os.IsNotExist(err):
		return "", err
	}

	if !approve {
		if nonInteractive {
			return "needs_approval", nil
		}
		ok, promptErr := promptApproval(fmt.Sprintf("Install wrap for %s?", toolID))
		if promptErr != nil {
			return "", promptErr
		}
		if !ok {
			return "needs_approval", nil
		}
	}

	if err := wrap.WriteInstallPlan(plan); err != nil {
		return "", err
	}
	if err := appendOperationalRunEvent(projectRoot, runEventWrapInstall, toolID, map[string]any{
		"command":      "wrap.install",
		"tool_id":      toolID,
		"install_path": plan.InstallPath,
		"status":       "installed",
		"dry_run":      false,
	}); err != nil {
		return "", err
	}
	return "installed", nil
}

func runInitHookStep(projectRoot string, noOverwrite, nonInteractive, approve bool) (string, error) {
	plan, err := hook.BuildInstallPlan(projectRoot, initDefaultHookType)
	if err != nil {
		return "", err
	}

	existing, err := hook.LoadExistingHook(plan.HookPath)
	switch {
	case err == nil && strings.TrimSpace(string(existing)) == strings.TrimSpace(plan.Script):
		return "up_to_date", nil
	case err == nil && noOverwrite:
		return "skipped_existing", nil
	case err != nil && !os.IsNotExist(err):
		return "", err
	}

	if !approve {
		if nonInteractive {
			return "needs_approval", nil
		}
		ok, promptErr := promptApproval(fmt.Sprintf("Install hook for %s?", initDefaultHookType))
		if promptErr != nil {
			return "", promptErr
		}
		if !ok {
			return "needs_approval", nil
		}
	}

	if err := hook.WriteInstallPlan(plan); err != nil {
		return "", err
	}
	if err := appendOperationalRunEvent(projectRoot, runEventHookInstall, "", map[string]any{
		"command":   "hook.install",
		"hook_type": initDefaultHookType,
		"hook_path": plan.HookPath,
		"status":    "installed",
		"dry_run":   false,
	}); err != nil {
		return "", err
	}
	return "installed", nil
}

func runInitIDEAutostartStep(projectRoot string, noOverwrite, nonInteractive, approve bool) (string, error) {
	plan, err := ideautostart.BuildPlan(projectRoot)
	if err != nil {
		return "", err
	}

	allUpToDate := true
	anyExisting := false
	for _, file := range plan.Files {
		if file.Exists {
			anyExisting = true
		}
		if file.Changed {
			allUpToDate = false
		}
	}
	if allUpToDate {
		return "up_to_date", nil
	}
	if anyExisting && noOverwrite {
		return "skipped_existing", nil
	}
	if !approve {
		if nonInteractive {
			return "needs_approval", nil
		}
		ok, promptErr := promptApproval("Generate IDE autostart settings?")
		if promptErr != nil {
			return "", promptErr
		}
		if !ok {
			return "needs_approval", nil
		}
	}

	if err := ideautostart.WritePlan(plan); err != nil {
		return "", err
	}
	if err := appendOperationalRunEvent(projectRoot, runEventIDEAutostartInstall, "", map[string]any{
		"command":      "ide-autostart",
		"project_root": projectRoot,
		"file_count":   len(plan.Files),
		"status":       "installed",
		"dry_run":      false,
		"files":        ideAutostartEventFiles(plan.Files),
	}); err != nil {
		return "", err
	}
	return "installed", nil
}

func ideAutostartEventFiles(files []ideautostart.FilePlan) []map[string]any {
	out := make([]map[string]any, 0, len(files))
	for _, file := range files {
		out = append(out, map[string]any{
			"kind":    file.Kind,
			"path":    file.Path,
			"exists":  file.Exists,
			"changed": file.Changed,
		})
	}
	return out
}
