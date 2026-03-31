package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
	"github.com/mash4649/atrakta/v0/internal/hook"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

type hookInstallResponse struct {
	HookType          string `json:"hook_type"`
	Status            string `json:"status"`
	HookPath          string `json:"hook_path"`
	NextAllowedAction string `json:"next_allowed_action"`
	ProjectRoot       string `json:"project_root"`
	DryRun            bool   `json:"dry_run"`
	Message           string `json:"message"`
}

type hookUninstallResponse struct {
	HookType          string `json:"hook_type"`
	Status            string `json:"status"`
	HookPath          string `json:"hook_path"`
	NextAllowedAction string `json:"next_allowed_action"`
	ProjectRoot       string `json:"project_root"`
	DryRun            bool   `json:"dry_run"`
	Message           string `json:"message"`
}

type hookStatusItem struct {
	HookType string `json:"hook_type"`
	HookPath string `json:"hook_path"`
	Status   string `json:"status"`
	Exists   bool   `json:"exists"`
	Managed  bool   `json:"managed"`
	Drift    bool   `json:"drift"`
	Message  string `json:"message"`
}

type hookStatusResponse struct {
	Status      string           `json:"status"`
	ProjectRoot string           `json:"project_root"`
	Hooks       []hookStatusItem `json:"hooks"`
	Message     string           `json:"message"`
}

type hookRepairItem struct {
	HookType string `json:"hook_type"`
	HookPath string `json:"hook_path"`
	Status   string `json:"status"`
	Repaired bool   `json:"repaired"`
	Message  string `json:"message"`
}

type hookRepairResponse struct {
	Status            string           `json:"status"`
	ProjectRoot       string           `json:"project_root"`
	Hooks             []hookRepairItem `json:"hooks"`
	NextAllowedAction string           `json:"next_allowed_action"`
	Message           string           `json:"message"`
}

func runHook(args []string) (int, error) {
	if len(args) == 0 {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"hook requires install, status, repair, or uninstall",
				"Run `atrakta hook --help` to inspect the supported subcommands.",
			),
			false,
		)
	}
	switch strings.TrimSpace(args[0]) {
	case "install":
		return runHookInstall(args[1:])
	case "status":
		return runHookStatus(args[1:])
	case "repair":
		return runHookRepair(args[1:])
	case "uninstall":
		return runHookUninstall(args[1:])
	default:
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				fmt.Sprintf("unsupported hook subcommand %q", args[0]),
				"Run `atrakta hook --help` to inspect the supported subcommands.",
			),
			false,
		)
	}
}

func runHookStatus(args []string) (int, error) {
	fs := flag.NewFlagSet("hook status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var jsonOut bool

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}
	if fs.NArg() != 0 {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"hook status does not accept positional arguments",
				"Remove the extra arguments and retry `atrakta hook status`.",
			),
			false,
		)
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plans, err := hook.BuildStatusPlans(root)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := hookStatusResponse{
		ProjectRoot: root,
		Hooks:       make([]hookStatusItem, 0, len(plans)),
	}
	managedCount := 0
	driftCount := 0
	missingCount := 0
	for _, plan := range plans {
		item := hookStatusItem{
			HookType: plan.HookType,
			HookPath: plan.HookPath,
			Status:   plan.Status,
			Exists:   plan.Exists,
			Managed:  plan.Managed,
			Drift:    plan.Drift,
		}
		switch plan.Status {
		case "missing":
			item.Message = "hook is not installed"
			missingCount++
		case "up_to_date":
			item.Message = "hook matches the expected managed script"
			managedCount++
		case "drift":
			item.Message = "hook content differs from the expected managed script"
			driftCount++
		default:
			item.Message = "unknown hook status"
		}
		resp.Hooks = append(resp.Hooks, item)
	}

	resp.Status = hookStatusSummaryStatus(resp.Hooks)
	resp.Message = hookStatusSummaryMessage(managedCount, driftCount, missingCount)

	if err := appendOperationalRunEvent(root, runEventHookStatusCheck, "", map[string]any{
		"command":       "hook.status",
		"status":        resp.Status,
		"hook_count":    len(resp.Hooks),
		"managed_count": managedCount,
		"drift_count":   driftCount,
		"missing_count": missingCount,
		"hooks":         hookStatusEventPayload(resp.Hooks),
	}); err != nil {
		return exitRuntimeError, err
	}

	if err := emitHookStatusResponse(resp, jsonOut); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func hookStatusSummaryStatus(hooks []hookStatusItem) string {
	status := "ok"
	for _, hook := range hooks {
		switch hook.Status {
		case "drift":
			return "drift"
		case "missing":
			if status == "ok" {
				status = "missing"
			}
		}
	}
	return status
}

func hookStatusSummaryMessage(managedCount, driftCount, missingCount int) string {
	if driftCount > 0 {
		return fmt.Sprintf("%d hook(s) are drifted, %d managed hook(s) are up to date, %d hook(s) are missing", driftCount, managedCount, missingCount)
	}
	if missingCount > 0 {
		return fmt.Sprintf("%d hook(s) are missing, %d managed hook(s) are up to date", missingCount, managedCount)
	}
	return fmt.Sprintf("%d hook(s) are up to date", managedCount)
}

func hookStatusEventPayload(hooks []hookStatusItem) []map[string]any {
	payload := make([]map[string]any, 0, len(hooks))
	for _, hook := range hooks {
		payload = append(payload, map[string]any{
			"hook_type": hook.HookType,
			"hook_path": hook.HookPath,
			"status":    hook.Status,
			"exists":    hook.Exists,
			"managed":   hook.Managed,
			"drift":     hook.Drift,
		})
	}
	return payload
}

func emitHookStatusResponse(resp hookStatusResponse, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
		return nil
	}

	fmt.Printf("status: %s\n", resp.Status)
	fmt.Printf("project_root: %s\n", resp.ProjectRoot)
	fmt.Printf("message: %s\n", resp.Message)
	for _, hookResp := range resp.Hooks {
		fmt.Printf("hook_type: %s\n", hookResp.HookType)
		fmt.Printf("hook_path: %s\n", hookResp.HookPath)
		fmt.Printf("hook_status: %s\n", hookResp.Status)
		fmt.Printf("exists: %t\n", hookResp.Exists)
		fmt.Printf("managed: %t\n", hookResp.Managed)
		fmt.Printf("drift: %t\n", hookResp.Drift)
		fmt.Printf("hook_message: %s\n", hookResp.Message)
	}
	return nil
}

func runHookRepair(args []string) (int, error) {
	fs := flag.NewFlagSet("hook repair", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var jsonOut bool
	var approve bool

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&approve, "approve", false, "explicitly approve repair")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}
	if fs.NArg() != 0 {
		return exitRuntimeError, fmt.Errorf("hook repair does not accept positional arguments")
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plans, err := hook.BuildRepairPlans(root)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := hookRepairResponse{
		ProjectRoot:       root,
		Hooks:             make([]hookRepairItem, 0, len(plans)),
		NextAllowedAction: "done",
	}
	for _, plan := range plans {
		resp.Hooks = append(resp.Hooks, hookRepairItem{
			HookType: plan.HookType,
			HookPath: plan.HookPath,
			Status:   plan.Status,
			Message:  "hook drift detected; repair available",
		})
	}

	if len(plans) == 0 {
		resp.Status = "ok"
		resp.Message = "no drifted hooks to repair"
		if err := appendOperationalRunEvent(root, runEventHookRepair, "", map[string]any{
			"command":        "hook.repair",
			"status":         resp.Status,
			"repaired_count": 0,
			"hook_count":     0,
		}); err != nil {
			return exitRuntimeError, err
		}
		if err := emitHookRepairResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if !approve {
		ok, promptErr := promptApproval("Repair drifted hooks?")
		if promptErr != nil {
			return exitRuntimeError, promptErr
		}
		if !ok {
			resp.Status = "needs_approval"
			resp.NextAllowedAction = "approve"
			resp.Message = "hook repair requires approval before writing files"
			if err := emitHookRepairResponse(resp, jsonOut); err != nil {
				return exitRuntimeError, err
			}
			return exitNeedsApproval, nil
		}
	}

	for i := range plans {
		if err := hook.RepairStatusPlan(plans[i]); err != nil {
			return exitRuntimeError, err
		}
		resp.Hooks[i].Repaired = true
		resp.Hooks[i].Status = "repaired"
		resp.Hooks[i].Message = "hook repaired from expected managed script"
	}

	resp.Status = "repaired"
	resp.Message = fmt.Sprintf("%d drifted hook(s) repaired", len(plans))
	if err := appendOperationalRunEvent(root, runEventHookRepair, "", map[string]any{
		"command":        "hook.repair",
		"status":         resp.Status,
		"repaired_count": len(plans),
		"hook_count":     len(plans),
		"hooks":          hookRepairEventPayload(resp.Hooks),
	}); err != nil {
		return exitRuntimeError, err
	}
	if err := emitHookRepairResponse(resp, jsonOut); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func hookRepairEventPayload(items []hookRepairItem) []map[string]any {
	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, map[string]any{
			"hook_type": item.HookType,
			"hook_path": item.HookPath,
			"status":    item.Status,
			"repaired":  item.Repaired,
		})
	}
	return payload
}

func emitHookRepairResponse(resp hookRepairResponse, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
		return nil
	}

	fmt.Printf("status: %s\n", resp.Status)
	fmt.Printf("project_root: %s\n", resp.ProjectRoot)
	fmt.Printf("next_allowed_action: %s\n", resp.NextAllowedAction)
	fmt.Printf("message: %s\n", resp.Message)
	for _, hookResp := range resp.Hooks {
		fmt.Printf("hook_type: %s\n", hookResp.HookType)
		fmt.Printf("hook_path: %s\n", hookResp.HookPath)
		fmt.Printf("hook_status: %s\n", hookResp.Status)
		fmt.Printf("repaired: %t\n", hookResp.Repaired)
		fmt.Printf("hook_message: %s\n", hookResp.Message)
	}
	return nil
}

func runHookInstall(args []string) (int, error) {
	fs := flag.NewFlagSet("hook install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var hookTypeFlag string
	var projectRoot string
	var jsonOut bool
	var dryRun bool
	var approve bool

	fs.StringVar(&hookTypeFlag, "hook-type", "", "hook type")
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&dryRun, "dry-run", false, "preview only")
	fs.BoolVar(&approve, "approve", false, "explicitly approve installation")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	hookType := strings.TrimSpace(hookTypeFlag)
	if hookType == "" && fs.NArg() == 1 {
		hookType = strings.TrimSpace(fs.Arg(0))
	}
	if hookType == "" {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"hook install requires a hook type",
				"Pass `--hook-type <name>` or a single positional hook type and retry.",
			),
			false,
		)
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plan, err := hook.BuildInstallPlan(root, hookType)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := hookInstallResponse{
		HookType:          plan.HookType,
		HookPath:          plan.HookPath,
		NextAllowedAction: "approve",
		ProjectRoot:       plan.ProjectRoot,
		DryRun:            dryRun,
	}

	if existing, err := hook.LoadExistingHook(plan.HookPath); err == nil {
		resp.Status = "needs_approval"
		resp.NextAllowedAction = string(existing)
		resp.Message = "hook already exists; review before installing"
		if err := emitHookInstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitNeedsApproval, nil
	} else if !os.IsNotExist(err) {
		return exitRuntimeError, err
	}

	if dryRun {
		resp.Status = "ok"
		resp.Message = "hook install dry-run plan generated"
		if err := emitHookInstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if !approve {
		ok, promptErr := promptApproval(fmt.Sprintf("Install hook for %s?", hookType))
		if promptErr != nil {
			return exitRuntimeError, promptErr
		}
		if !ok {
			resp.Status = "needs_approval"
			resp.Message = "hook install requires approval before writing files"
			if err := emitHookInstallResponse(resp, jsonOut); err != nil {
				return exitRuntimeError, err
			}
			return exitNeedsApproval, nil
		}
	}

	if err := hook.WriteInstallPlan(plan); err != nil {
		return exitRuntimeError, err
	}
	if err := appendOperationalRunEvent(root, runEventHookInstall, "", map[string]any{
		"command":   "hook.install",
		"hook_type": plan.HookType,
		"hook_path": plan.HookPath,
		"dry_run":   dryRun,
		"status":    "installed",
	}); err != nil {
		return exitRuntimeError, err
	}

	resp.Status = "installed"
	resp.Message = "hook install completed"
	resp.NextAllowedAction = "done"
	if err := emitHookInstallResponse(resp, jsonOut); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func emitHookInstallResponse(resp hookInstallResponse, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
	} else {
		fmt.Printf("hook_type: %s\n", resp.HookType)
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("hook_path: %s\n", resp.HookPath)
		fmt.Printf("next_allowed_action: %s\n", resp.NextAllowedAction)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("dry_run: %t\n", resp.DryRun)
		fmt.Printf("message: %s\n", resp.Message)
	}
	return nil
}

func runHookUninstall(args []string) (int, error) {
	fs := flag.NewFlagSet("hook uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var hookTypeFlag string
	var projectRoot string
	var jsonOut bool
	var dryRun bool

	fs.StringVar(&hookTypeFlag, "hook-type", "", "hook type")
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&dryRun, "dry-run", false, "preview only")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	hookType := strings.TrimSpace(hookTypeFlag)
	if hookType == "" && fs.NArg() == 1 {
		hookType = strings.TrimSpace(fs.Arg(0))
	}
	if hookType == "" {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"hook uninstall requires a hook type",
				"Pass `--hook-type <name>` or a single positional hook type and retry.",
			),
			false,
		)
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plan, err := hook.BuildUninstallPlan(root, hookType)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := hookUninstallResponse{
		HookType:    plan.HookType,
		HookPath:    plan.HookPath,
		ProjectRoot: plan.ProjectRoot,
		DryRun:      dryRun,
	}

	if !plan.Exists {
		resp.Status = "skipped"
		resp.NextAllowedAction = "done"
		resp.Message = "hook uninstall target not found"
		if err := emitHookUninstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if !plan.Managed {
		resp.Status = "skipped"
		resp.NextAllowedAction = plan.CurrentScript
		resp.Message = "hook uninstall target is not managed by atrakta"
		if err := emitHookUninstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if dryRun {
		resp.Status = "ok"
		resp.NextAllowedAction = "approve"
		resp.Message = "hook uninstall dry-run plan generated"
		if err := emitHookUninstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if err := hook.RemoveUninstallPlan(plan); err != nil {
		return exitRuntimeError, err
	}
	if err := appendOperationalRunEvent(root, runEventHookUninstall, "", map[string]any{
		"command":   "hook.uninstall",
		"hook_type": plan.HookType,
		"hook_path": plan.HookPath,
		"dry_run":   dryRun,
		"status":    "uninstalled",
	}); err != nil {
		return exitRuntimeError, err
	}

	resp.Status = "uninstalled"
	resp.NextAllowedAction = "done"
	resp.Message = "hook uninstall completed"
	if err := emitHookUninstallResponse(resp, jsonOut); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func emitHookUninstallResponse(resp hookUninstallResponse, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
	} else {
		fmt.Printf("hook_type: %s\n", resp.HookType)
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("hook_path: %s\n", resp.HookPath)
		fmt.Printf("next_allowed_action: %s\n", resp.NextAllowedAction)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("dry_run: %t\n", resp.DryRun)
		fmt.Printf("message: %s\n", resp.Message)
	}
	return nil
}
