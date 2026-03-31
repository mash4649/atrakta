package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/ideautostart"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

type ideAutostartFile struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Changed bool   `json:"changed"`
}

type ideAutostartResponse struct {
	Status            string             `json:"status"`
	ProjectRoot       string             `json:"project_root"`
	DryRun            bool               `json:"dry_run"`
	NextAllowedAction string             `json:"next_allowed_action"`
	Files             []ideAutostartFile `json:"files"`
	Message           string             `json:"message"`
}

func runIDEAutostart(args []string) (int, error) {
	fs := flag.NewFlagSet("ide-autostart", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var jsonOut bool
	var dryRun bool
	var approve bool

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&dryRun, "dry-run", false, "preview only")
	fs.BoolVar(&approve, "approve", false, "explicitly approve file writes")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}
	if fs.NArg() != 0 {
		return exitRuntimeError, fmt.Errorf("ide-autostart does not accept positional arguments")
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}

	plan, err := ideautostart.BuildPlan(root)
	if err != nil {
		return exitRuntimeError, err
	}

	resp := ideAutostartResponse{
		ProjectRoot:       plan.ProjectRoot,
		DryRun:            dryRun,
		NextAllowedAction: "approve",
		Files:             make([]ideAutostartFile, 0, len(plan.Files)),
	}
	for _, file := range plan.Files {
		resp.Files = append(resp.Files, ideAutostartFile{
			Kind:    file.Kind,
			Path:    file.Path,
			Exists:  file.Exists,
			Changed: file.Changed,
		})
	}

	if dryRun {
		resp.Status = "ok"
		resp.Message = "ide-autostart dry-run plan generated"
		return emitIDEAutostartResponse(resp, jsonOut)
	}

	if !approve {
		ok, promptErr := promptApproval("Generate IDE autostart settings?")
		if promptErr != nil {
			return exitRuntimeError, promptErr
		}
		if !ok {
			resp.Status = "needs_approval"
			resp.Message = "ide-autostart requires approval before writing files"
			return emitIDEAutostartResponse(resp, jsonOut)
		}
	}

	if err := ideautostart.WritePlan(plan); err != nil {
		return exitRuntimeError, err
	}
	if err := appendOperationalRunEvent(root, runEventIDEAutostartInstall, "", map[string]any{
		"command":      "ide-autostart",
		"dry_run":      dryRun,
		"project_root": root,
		"file_count":   len(plan.Files),
		"files":        resp.Files,
		"status":       "installed",
	}); err != nil {
		return exitRuntimeError, err
	}

	resp.Status = "installed"
	resp.NextAllowedAction = "done"
	resp.Message = "ide-autostart completed"
	return emitIDEAutostartResponse(resp, jsonOut)
}

func emitIDEAutostartResponse(resp ideAutostartResponse, jsonOut bool) (int, error) {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return exitRuntimeError, err
		}
	} else {
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("dry_run: %t\n", resp.DryRun)
		fmt.Printf("next_allowed_action: %s\n", resp.NextAllowedAction)
		for _, file := range resp.Files {
			fmt.Printf("file_kind: %s\n", file.Kind)
			fmt.Printf("file_path: %s\n", file.Path)
			fmt.Printf("file_exists: %t\n", file.Exists)
			fmt.Printf("file_changed: %t\n", file.Changed)
		}
		fmt.Printf("message: %s\n", resp.Message)
	}
	if resp.Status == "needs_approval" {
		return exitNeedsApproval, nil
	}
	return exitOK, nil
}
