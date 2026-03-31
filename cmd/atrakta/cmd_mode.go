package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runMode(mode string, args []string) error {
	fs := flag.NewFlagSet(mode, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var inputPath string
	var artifactDir string
	var onboardRoot string
	fs.StringVar(&inputPath, "input", "", "bundle input JSON path")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.StringVar(&onboardRoot, "onboard-root", "", "project root for onboarding-derived failure routing")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var input pipeline.BundleInput
	if inputPath != "" {
		b, err := os.ReadFile(inputPath)
		if err != nil {
			return err
		}
		input, err = validation.DecodeAndValidateBundleInput(b)
		if err != nil {
			return err
		}
	} else {
		var err error
		input, err = buildDefaultInput(mode)
		if err != nil {
			return err
		}
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
		if err := validation.ValidateBundleInput(input); err != nil {
			return err
		}
	}

	out, err := executeBundle(mode, input)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, mode+".bundle.json", out); err != nil {
			return err
		}
	}
	return nil
}

func buildDefaultInput(mode string) (pipeline.BundleInput, error) {
	if err := validation.ValidateMode(mode); err != nil {
		return pipeline.BundleInput{}, err
	}
	input := pipeline.DefaultInput(mode)
	if err := validation.ValidateBundleInput(input); err != nil {
		return pipeline.BundleInput{}, err
	}
	return input, nil
}

func executeBundle(mode string, input pipeline.BundleInput) (pipeline.BundleOutput, error) {
	out, err := pipeline.ExecuteOrdered(mode, input)
	if err != nil {
		return pipeline.BundleOutput{}, err
	}
	if err := validation.ValidateBundleOutput(out); err != nil {
		return pipeline.BundleOutput{}, err
	}
	return out, nil
}

func applyOnboardingFailure(input pipeline.BundleInput, bundle onboarding.ProposalBundle) pipeline.BundleInput {
	input.FailureClass = bundle.InferredFailure.FailureClass
	input.FailureContext.Scope = bundle.InferredFailure.Scope
	input.FailureContext.Triggers = append([]string{}, bundle.InferredFailure.Triggers...)
	input.FailureContext.IsDiagnosticsOnly = len(bundle.Conflicts) == 0
	return input
}

