package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/validation"
	"github.com/mash4649/atrakta/v0/resolvers/operations/resolve-operation-capability"
)

func runAlias(alias string, args []string) error {
	if alias == "doctor" {
		return runDoctor(args)
	}

	fs := flag.NewFlagSet(alias, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var failureTier string
	var execute bool
	var onboardRoot string
	var artifactDir string
	fs.StringVar(&failureTier, "failure-tier", "", "failure tier ceiling")
	fs.BoolVar(&execute, "execute", false, "execute mapped pipeline mode")
	fs.StringVar(&onboardRoot, "onboard-root", "", "project root for onboarding-derived failure routing")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	capOut := resolveoperationcapability.ResolveOperationCapability(resolveoperationcapability.Input{
		CommandOrAlias: alias,
		FailureTier:    failureTier,
	})
	response := map[string]any{
		"alias":      alias,
		"capability": capOut,
	}

	if execute {
		decision := capOut.Decision.(resolveoperationcapability.CapabilityDecision)
		mode := "inspect"
		switch decision.EffectiveActionClass {
		case resolveoperationcapability.ActionPropose:
			mode = "preview"
		case resolveoperationcapability.ActionApply:
			mode = "simulate"
		}
		input, err := buildDefaultInput(mode)
		if err != nil {
			return err
		}
		if onboardRoot != "" {
			onboardingBundle, err := onboarding.BuildOnboardingProposal(onboardRoot)
			if err != nil {
				return err
			}
			if err := validation.ValidateOnboardingProposal(onboardingBundle); err != nil {
				return err
			}
			input = applyOnboardingFailure(input, onboardingBundle)
		}
		bundle, err := executeBundle(mode, input)
		if err != nil {
			return err
		}
		response["mode"] = mode
		response["bundle"] = bundle
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(response); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, alias+".alias.json", response); err != nil {
			return err
		}
	}
	return nil
}
