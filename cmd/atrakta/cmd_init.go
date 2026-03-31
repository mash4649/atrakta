package main

import (
	"flag"
	"io"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/entry"
	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
)

func initCommand(args []string) (int, error) {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var interfaceID string
	var artifactDir string
	var mode string
	var jsonOut bool
	var nonInteractive bool
	var apply bool
	var approve bool
	var noOverwrite bool
	var noWrap bool
	var noHook bool
	var noIDEAutostart bool

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&interfaceID, "interface", "", "runtime interface id")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.StringVar(&mode, "mode", "", "bootstrap mode hint (greenfield|brownfield)")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "disable approval prompt")
	fs.BoolVar(&apply, "apply", false, "request apply route")
	fs.BoolVar(&approve, "approve", false, "explicitly approve write path")
	fs.BoolVar(&noOverwrite, "no-overwrite", false, "do not overwrite existing integration assets")
	fs.BoolVar(&noWrap, "no-wrap", false, "skip wrap integration step")
	fs.BoolVar(&noHook, "no-hook", false, "skip hook integration step")
	fs.BoolVar(&noIDEAutostart, "no-ide-autostart", false, "skip ide-autostart integration step")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	root, err := entry.Resolve(entry.Input{
		ProjectRoot: projectRoot,
		InterfaceID: interfaceID,
	})
	if err != nil {
		return exitRuntimeError, err
	}
	if root.NeedsInput {
		return exitNeedsInput, atraktaerrors.NewExitError(
			exitNeedsInput,
			atraktaerrors.Usage(
				"init requires an explicit interface on this project",
				"Re-run `atrakta init --interface <id>` or set ATRAKTA_TRIGGER_INTERFACE and retry.",
			),
			false,
		)
	}

	resolvedInterface := strings.TrimSpace(root.Interface.InterfaceID)
	if resolvedInterface == "" {
		resolvedInterface = "generic-cli"
	}

	if err := appendOperationalRunEvent(root.ProjectRoot, runEventInitBegin, resolvedInterface, map[string]any{
		"command":           "init",
		"path":              root.Path,
		"canonical_state":   root.CanonicalState,
		"requested_mode":    strings.TrimSpace(mode),
		"requested_iface":   strings.TrimSpace(interfaceID),
		"resolved_iface":    resolvedInterface,
		"resolved_source":   root.Interface.Source,
		"no_overwrite":      noOverwrite,
		"no_wrap":           noWrap,
		"no_hook":           noHook,
		"no_ide_autostart":  noIDEAutostart,
		"non_interactive":   nonInteractive,
		"apply_requested":   apply,
		"explicit_approved": approve,
	}); err != nil {
		return exitRuntimeError, err
	}

	endStatus := "ok"
	startExitCode := exitOK
	startError := ""
	stepsCompleted := 0
	defer func() {
		_ = appendOperationalRunEvent(root.ProjectRoot, runEventInitEnd, resolvedInterface, map[string]any{
			"command":         "init",
			"status":          endStatus,
			"steps_completed": stepsCompleted,
			"start_exit_code": startExitCode,
			"start_error":     startError,
			"resolved_iface":  resolvedInterface,
			"resolved_source": root.Interface.Source,
			"canonical_state": root.CanonicalState,
			"requested_iface": strings.TrimSpace(interfaceID),
			"requested_mode":  strings.TrimSpace(mode),
		})
	}()

	if root.Path == entry.PathOnboarding {
		setupStatus, setupErr := runInitOnboardingSetup(root.ProjectRoot, resolvedInterface, noOverwrite, noWrap, noHook, noIDEAutostart, nonInteractive, approve)
		if setupErr != nil {
			endStatus = "error"
			startExitCode = exitRuntimeError
			startError = setupErr.Error()
			return exitRuntimeError, setupErr
		}
		stepsCompleted = setupStatus.CompletedSteps
		if setupStatus.NeedsApproval {
			endStatus = "needs_approval"
			startExitCode = exitNeedsApproval
			return exitNeedsApproval, nil
		}
	}

	forwarded := []string{"--project-root", root.ProjectRoot}
	if resolvedInterface != "" {
		forwarded = append(forwarded, "--interface", resolvedInterface)
	}
	if strings.TrimSpace(artifactDir) != "" {
		forwarded = append(forwarded, "--artifact-dir", strings.TrimSpace(artifactDir))
	}
	if jsonOut {
		forwarded = append(forwarded, "--json")
	}
	if nonInteractive {
		forwarded = append(forwarded, "--non-interactive")
	}
	if apply {
		forwarded = append(forwarded, "--apply")
	}
	if approve {
		forwarded = append(forwarded, "--approve")
	}

	code, runErr := startCommand(forwarded)
	startExitCode = code
	if runErr != nil {
		endStatus = "error"
		startError = runErr.Error()
		return code, runErr
	}
	if code != exitOK {
		endStatus = "non_ok"
	}
	return code, nil
}
