package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
	"github.com/mash4649/atrakta/v0/internal/projection"
)

type projectionResponse struct {
	Status           string                         `json:"status"`
	Action           string                         `json:"action"`
	ProjectRoot      string                         `json:"project_root"`
	TargetID         string                         `json:"target_id,omitempty"`
	TargetPath       string                         `json:"target_path"`
	ProjectionStatus string                         `json:"projection_status,omitempty"`
	Drift            bool                           `json:"drift"`
	Written          bool                           `json:"written"`
	DryRun           bool                           `json:"dry_run,omitempty"`
	Message          string                         `json:"message"`
	ExpectedHash     string                         `json:"expected_hash,omitempty"`
	ActualHash       string                         `json:"actual_hash,omitempty"`
	DetectedAssets   []string                       `json:"detected_assets,omitempty"`
	Error            *atraktaerrors.StructuredError `json:"error,omitempty"`
}

func runProjection(args []string) error {
	if len(args) == 0 {
		return atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				"projection requires render, status, or repair",
				"Run `atrakta projection --help` to inspect the supported actions.",
			),
			false,
		)
	}
	action := strings.TrimSpace(args[0])
	switch action {
	case "render":
		return runProjectionAction("render", args[1:], true, false)
	case "status":
		return runProjectionAction("status", args[1:], false, false)
	case "repair":
		return runProjectionAction("repair", args[1:], false, true)
	default:
		return atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Usage(
				fmt.Sprintf("unknown projection action %q", action),
				"Run `atrakta projection --help` to inspect the supported actions.",
			),
			false,
		)
	}
}

func runProjectionAction(action string, args []string, writeAlways, repair bool) error {
	fs := flag.NewFlagSet("projection "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var target string
	var interfaceID string
	var jsonOut bool
	var artifactDir string
	var dryRun bool
	var approve bool

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&target, "target", projection.DefaultTarget, "projection target")
	fs.StringVar(&interfaceID, "interface", "", "projection interface")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.BoolVar(&dryRun, "dry-run", false, "preview only")
	fs.BoolVar(&approve, "approve", false, "explicitly approve file writes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var res projection.Result
	var err error
	switch {
	case repair:
		res, err = projection.Repair(projectRoot, target, interfaceID)
	case writeAlways:
		res, err = projection.Render(projectRoot, target, interfaceID)
	default:
		res, err = projection.Status(projectRoot, target, interfaceID)
	}
	if err != nil {
		return err
	}

	resp := projectionResponse{
		Status:           projectionStatus(action, res.Drift, res.Written),
		Action:           action,
		ProjectRoot:      res.ProjectRoot,
		TargetID:         res.TargetID,
		TargetPath:       res.TargetPath,
		ProjectionStatus: res.ProjectionStatus,
		Drift:            res.Drift,
		Written:          res.Written,
		DryRun:           dryRun,
		Message:          res.Message,
		ExpectedHash:     res.ExpectedHash,
		ActualHash:       res.ActualHash,
		DetectedAssets:   append([]string(nil), res.DetectedAssets...),
	}

	if action == "render" {
		if dryRun {
			resp.Status = "ok"
			resp.Message = "projection render dry-run plan generated"
		} else {
			if !approve {
				ok, promptErr := promptApproval(fmt.Sprintf("Render projection for %s?", resp.TargetPath))
				if promptErr != nil {
					return promptErr
				}
				if !ok {
					resp.Status = "needs_approval"
					resp.Message = "projection render requires approval before writing files"
					errDetail := atraktaerrors.NewStructured(
						"ERR_APPROVAL_REQUIRED",
						resp.Message,
						"Re-run with `--approve` or use an interactive terminal to confirm the render.",
					)
					resp.Error = &errDetail
					if err := emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, runEventProjectionRendered); err != nil {
						return err
					}
					return atraktaerrors.NewExitError(exitRuntimeError, errDetail, true)
				}
			}
			res, err = projection.Write(projectRoot, target, interfaceID)
			if err != nil {
				return err
			}
			resp.Status = projectionStatus(action, res.Drift, res.Written)
			resp.Written = res.Written
			resp.ProjectionStatus = res.ProjectionStatus
			resp.Message = res.Message
			resp.ExpectedHash = res.ExpectedHash
			resp.ActualHash = res.ActualHash
		}
		return emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, runEventProjectionRendered)
	}

	if action == "repair" {
		if !approve {
			ok, promptErr := promptApproval(fmt.Sprintf("Repair projection for %s?", resp.TargetPath))
			if promptErr != nil {
				return promptErr
			}
			if !ok {
				resp.Status = "needs_approval"
				resp.Message = "projection repair requires approval before writing files"
				errDetail := atraktaerrors.NewStructured(
					"ERR_APPROVAL_REQUIRED",
					resp.Message,
					"Re-run with `--approve` or use an interactive terminal to confirm the repair.",
				)
				resp.Error = &errDetail
				if err := emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, runEventProjectionRepaired); err != nil {
					return err
				}
				return atraktaerrors.NewExitError(exitRuntimeError, errDetail, true)
			}
		}
		if res.ProjectionStatus == "modified_externally" {
			resp.Status = "blocked"
			resp.Message = "projection repair blocked by externally modified target; review manually"
			errDetail := atraktaerrors.NewStructured(
				"ERR_BLOCKED",
				resp.Message,
				"Inspect the drift with `atrakta projection status`, then resolve the externally modified file before retrying.",
			)
			resp.Error = &errDetail
			if err := emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, runEventProjectionRepaired); err != nil {
				return err
			}
			return atraktaerrors.NewExitError(exitRuntimeError, errDetail, true)
		}
		if !res.Drift {
			resp.Status = "ok"
			resp.Message = "projection already in sync"
			if err := emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, runEventProjectionRepaired); err != nil {
				return err
			}
			return nil
		}
		res, err = projection.Write(projectRoot, target, interfaceID)
		if err != nil {
			return err
		}
		resp.Status = "repaired"
		resp.Written = res.Written
		resp.ProjectionStatus = res.ProjectionStatus
		resp.Message = "projection repaired"
		resp.ExpectedHash = res.ExpectedHash
		resp.ActualHash = res.ActualHash
		eventType := runEventProjectionRepaired
		return emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, eventType)
	}

	eventType := runEventProjectionStatusCheck
	return emitProjectionWithEvent(resp, jsonOut, artifactDir, res.ProjectRoot, res.Interface.InterfaceID, eventType)
}

func projectionStatus(action string, drift, written bool) string {
	switch action {
	case "render":
		return "ok"
	case "repair":
		if written {
			return "repaired"
		}
		return "ok"
	default:
		if drift {
			return "drift"
		}
		return "ok"
	}
}

func emitProjectionResponse(resp projectionResponse, jsonOut bool, artifactDir string) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			return err
		}
	} else {
		fmt.Printf("status: %s\n", resp.Status)
		fmt.Printf("action: %s\n", resp.Action)
		fmt.Printf("project_root: %s\n", resp.ProjectRoot)
		fmt.Printf("target_id: %s\n", resp.TargetID)
		fmt.Printf("target_path: %s\n", resp.TargetPath)
		fmt.Printf("projection_status: %s\n", resp.ProjectionStatus)
		fmt.Printf("drift: %t\n", resp.Drift)
		fmt.Printf("written: %t\n", resp.Written)
		fmt.Printf("dry_run: %t\n", resp.DryRun)
		fmt.Printf("message: %s\n", resp.Message)
		if resp.Error != nil {
			fmt.Printf("error_code: %s\n", resp.Error.Code)
			fmt.Printf("error_message: %s\n", resp.Error.Message)
			if len(resp.Error.RecoverySteps) > 0 {
				fmt.Printf("recovery_steps: %s\n", strings.Join(resp.Error.RecoverySteps, " | "))
			}
		}
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "projection."+resp.Action+".result.json", resp); err != nil {
			return err
		}
	}
	return nil
}

func emitProjectionWithEvent(resp projectionResponse, jsonOut bool, artifactDir, projectRoot, interfaceID, eventType string) error {
	if err := appendOperationalRunEvent(projectRoot, eventType, interfaceID, map[string]any{
		"command":           "projection." + resp.Action,
		"status":            resp.Status,
		"target_id":         resp.TargetID,
		"target_path":       resp.TargetPath,
		"projection_status": resp.ProjectionStatus,
		"drift":             resp.Drift,
		"written":           resp.Written,
		"dry_run":           resp.DryRun,
		"expected_hash":     resp.ExpectedHash,
		"actual_hash":       resp.ActualHash,
	}); err != nil {
		return err
	}
	return emitProjectionResponse(resp, jsonOut, artifactDir)
}
