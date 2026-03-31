package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/wrap"
)

type wrapInstallResponse struct {
	ToolID            string   `json:"tool_id"`
	Status            string   `json:"status"`
	InstalledPath     string   `json:"installed_path"`
	NextAllowedAction string   `json:"next_allowed_action"`
	ProjectRoot       string   `json:"project_root"`
	DryRun            bool     `json:"dry_run"`
	Capabilities      []string `json:"capabilities,omitempty"`
	Message           string   `json:"message"`
}

type wrapUninstallResponse struct {
	ToolID            string `json:"tool_id"`
	Status            string `json:"status"`
	RemovedPath       string `json:"removed_path"`
	NextAllowedAction string `json:"next_allowed_action"`
	ProjectRoot       string `json:"project_root"`
	DryRun            bool   `json:"dry_run"`
	Message           string `json:"message"`
}

func runWrap(args []string) (int, error) {
	if len(args) == 0 {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"wrap requires install, uninstall, or run",
				"Run `atrakta wrap --help` to inspect the supported subcommands.",
			),
			false,
		)
	}
	switch strings.TrimSpace(args[0]) {
	case "install":
		return runWrapInstall(args[1:])
	case "uninstall":
		return runWrapUninstall(args[1:])
	case "run":
		return runWrapRun(args[1:])
	default:
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				fmt.Sprintf("unsupported wrap subcommand %q", args[0]),
				"Run `atrakta wrap --help` to inspect the supported subcommands.",
			),
			false,
		)
	}
}

func runWrapInstall(args []string) (int, error) {
	fs := flag.NewFlagSet("wrap install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var toolIDFlag string
	var projectRoot string
	var jsonOut bool
	var dryRun bool
	var approve bool
	var nonInteractive bool

	fs.StringVar(&toolIDFlag, "tool", "", "tool id")
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&dryRun, "dry-run", false, "preview only")
	fs.BoolVar(&approve, "approve", false, "explicitly approve installation")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "disable approval prompt")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	toolID := strings.TrimSpace(toolIDFlag)
	if toolID == "" && fs.NArg() == 1 {
		toolID = strings.TrimSpace(fs.Arg(0))
	}
	if toolID == "" {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"wrap install requires a tool id",
				"Pass `--tool <id>` or a single positional tool id and retry.",
			),
			false,
		)
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plan, err := wrap.BuildInstallPlan(root, toolID)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := wrapInstallResponse{
		ToolID:            plan.ToolID,
		InstalledPath:     plan.InstallPath,
		NextAllowedAction: "approve",
		ProjectRoot:       plan.ProjectRoot,
		DryRun:            dryRun,
		Capabilities:      append([]string(nil), plan.Capabilities...),
	}

	if dryRun {
		resp.Status = "ok"
		resp.Message = "wrap install dry-run plan generated"
		if err := emitWrapInstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if !approve {
		if nonInteractive {
			resp.Status = "needs_approval"
			resp.Message = "wrap install requires approval before writing files"
			if err := emitWrapInstallResponse(resp, jsonOut); err != nil {
				return exitRuntimeError, err
			}
			return exitNeedsApproval, nil
		}
		ok, promptErr := promptApproval(fmt.Sprintf("Install wrap for %s?", toolID))
		if promptErr != nil {
			return exitRuntimeError, promptErr
		}
		if !ok {
			resp.Status = "needs_approval"
			resp.Message = "wrap install requires approval before writing files"
			if err := emitWrapInstallResponse(resp, jsonOut); err != nil {
				return exitRuntimeError, err
			}
			return exitNeedsApproval, nil
		}
	}

	if err := wrap.WriteInstallPlan(plan); err != nil {
		return exitRuntimeError, err
	}
	if err := appendOperationalRunEvent(root, runEventWrapInstall, "", map[string]any{
		"command":      "wrap.install",
		"tool_id":      plan.ToolID,
		"install_path": plan.InstallPath,
		"capabilities": plan.Capabilities,
		"dry_run":      dryRun,
		"status":       "installed",
	}); err != nil {
		return exitRuntimeError, err
	}

	resp.Status = "installed"
	resp.Message = "wrap install completed"
	resp.NextAllowedAction = "run"
	if err := emitWrapInstallResponse(resp, jsonOut); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func emitWrapInstallResponse(resp wrapInstallResponse, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
	} else {
		fmt.Printf("tool_id: %s\n", resp.ToolID)
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("installed_path: %s\n", resp.InstalledPath)
		fmt.Printf("next_allowed_action: %s\n", resp.NextAllowedAction)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("dry_run: %t\n", resp.DryRun)
		if len(resp.Capabilities) > 0 {
			fmt.Printf("capabilities: %s\n", strings.Join(resp.Capabilities, ","))
		}
		fmt.Printf("message: %s\n", resp.Message)
	}
	return nil
}

func runWrapUninstall(args []string) (int, error) {
	fs := flag.NewFlagSet("wrap uninstall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var toolIDFlag string
	var projectRoot string
	var jsonOut bool
	var dryRun bool

	fs.StringVar(&toolIDFlag, "tool", "", "tool id")
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&dryRun, "dry-run", false, "preview only")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	toolID := strings.TrimSpace(toolIDFlag)
	if toolID == "" && fs.NArg() == 1 {
		toolID = strings.TrimSpace(fs.Arg(0))
	}
	if toolID == "" {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"wrap uninstall requires a tool id",
				"Pass `--tool <id>` or a single positional tool id and retry.",
			),
			false,
		)
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plan, err := wrap.BuildUninstallPlan(root, toolID)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := wrapUninstallResponse{
		ToolID:            plan.ToolID,
		RemovedPath:       plan.InstallPath,
		NextAllowedAction: "done",
		ProjectRoot:       plan.ProjectRoot,
		DryRun:            dryRun,
	}

	if dryRun {
		resp.Status = "ok"
		resp.Message = "wrap uninstall dry-run plan generated"
		if err := emitWrapUninstallResponse(resp, jsonOut); err != nil {
			return exitRuntimeError, err
		}
		return exitOK, nil
	}

	if err := wrap.RemoveUninstallPlan(plan); err != nil {
		return exitRuntimeError, err
	}
	if err := appendOperationalRunEvent(root, runEventWrapUninstall, "", map[string]any{
		"command":      "wrap.uninstall",
		"tool_id":      plan.ToolID,
		"removed_path": plan.InstallPath,
		"dry_run":      dryRun,
		"status":       "uninstalled",
	}); err != nil {
		return exitRuntimeError, err
	}

	resp.Status = "uninstalled"
	resp.Message = "wrap uninstall completed"
	if err := emitWrapUninstallResponse(resp, jsonOut); err != nil {
		return exitRuntimeError, err
	}
	return exitOK, nil
}

func emitWrapUninstallResponse(resp wrapUninstallResponse, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
	} else {
		fmt.Printf("tool_id: %s\n", resp.ToolID)
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("removed_path: %s\n", resp.RemovedPath)
		fmt.Printf("next_allowed_action: %s\n", resp.NextAllowedAction)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("dry_run: %t\n", resp.DryRun)
		fmt.Printf("message: %s\n", resp.Message)
	}
	return nil
}

func runWrapRun(args []string) (int, error) {
	if len(args) == 0 {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"wrap run requires a tool id",
				"Pass the wrapped tool id as the first argument and retry.",
			),
			false,
		)
	}
	toolID := strings.TrimSpace(args[0])
	forwardArgs := append([]string(nil), args[1:]...)

	root, err := onboarding.DetectProjectRoot("")
	if err != nil {
		return exitRuntimeError, err
	}

	code, runErr := wrap.RunWrapped(toolID, forwardArgs)

	if eventErr := appendOperationalRunEvent(root, runEventWrapRun, "", map[string]any{
		"command":   "wrap.run",
		"tool_id":   toolID,
		"args":      forwardArgs,
		"exit_code": code,
		"status":    wrapRunStatus(code, runErr),
	}); eventErr != nil {
		if runErr != nil {
			return code, fmt.Errorf("%v; audit append failed: %w", runErr, eventErr)
		}
		return exitRuntimeError, eventErr
	}

	return code, runErr
}

func wrapRunStatus(code int, err error) string {
	if err == nil && code == exitOK {
		return "completed"
	}
	return "failed"
}
