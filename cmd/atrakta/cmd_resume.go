package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	atraktaerrors "github.com/mash4649/atrakta/v0/internal/errors"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

var errResumeRequiresState = errors.New("resume requires existing atrakta state; run `atrakta start` first")
var errResumeBlockedByHandoff = errors.New("resume blocked by handoff next_action=deny")

func resumeCommand(args []string) (int, error) {
	fs := flag.NewFlagSet("resume", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var interfaceID string
	var artifactDir string
	var jsonOut bool
	var nonInteractive bool
	var apply bool
	var approve bool

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&interfaceID, "interface", "", "runtime interface id")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "disable approval prompt")
	fs.BoolVar(&apply, "apply", false, "request apply route")
	fs.BoolVar(&approve, "approve", false, "explicitly approve write path")
	if err := fs.Parse(args); err != nil {
		return exitRuntimeError, err
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return exitRuntimeError, err
	}
	state, err := runpkg.DetectCanonicalState(root)
	if err != nil {
		return exitRuntimeError, err
	}
	if state == runpkg.StateNone {
		return exitRuntimeError, atraktaerrors.NewExitError(
			exitRuntimeError,
			atraktaerrors.Blocked(
				errResumeRequiresState.Error(),
				"Run `atrakta start` first to create the session state, then retry `atrakta resume`.",
			),
			false,
		)
	}

	forwarded := append([]string{}, args...)
	autoState, autoStateErr := runpkg.LoadAutoState(root)
	handoff, handoffErr := runpkg.LoadHandoff(root)
	if handoffErr == nil {
		if shouldResumeBlock(handoff) {
			hint := strings.TrimSpace(handoff.NextAction.Hint)
			if hint != "" {
				return exitRuntimeError, atraktaerrors.NewExitError(
					exitRuntimeError,
					atraktaerrors.Blocked(
						fmt.Sprintf("%s: %s", errResumeBlockedByHandoff.Error(), hint),
						"Inspect the handoff hint, resolve the deny condition, then retry the command.",
					),
					false,
				)
			}
			return exitRuntimeError, atraktaerrors.NewExitError(
				exitRuntimeError,
				atraktaerrors.Blocked(
					errResumeBlockedByHandoff.Error(),
					"Inspect the handoff next_action, resolve the deny condition, then retry the command.",
				),
				false,
			)
		}
		if !flagWasProvided(fs, "interface") && handoff.InterfaceID != "" {
			forwarded = append(forwarded, "--interface", handoff.InterfaceID)
		} else if !flagWasProvided(fs, "interface") && autoStateErr == nil && strings.TrimSpace(autoState.InterfaceID) != "" {
			forwarded = append(forwarded, "--interface", autoState.InterfaceID)
		}
		if !flagWasProvided(fs, "apply") && shouldResumeWithApply(handoff) {
			forwarded = append(forwarded, "--apply")
		}
	} else if !flagWasProvided(fs, "interface") && autoStateErr == nil && strings.TrimSpace(autoState.InterfaceID) != "" {
		forwarded = append(forwarded, "--interface", autoState.InterfaceID)
	}

	return runLikeCommand("resume", forwarded)
}

func handoffNextAction(handoff runpkg.HandoffBundle) string {
	action := strings.TrimSpace(handoff.NextAction.Command)
	if action == "" {
		action = strings.TrimSpace(handoff.NextAllowedAction)
	}
	return action
}

func shouldResumeWithApply(handoff runpkg.HandoffBundle) bool {
	return handoffNextAction(handoff) == "apply"
}

func shouldResumeBlock(handoff runpkg.HandoffBundle) bool {
	return handoffNextAction(handoff) == "deny"
}
