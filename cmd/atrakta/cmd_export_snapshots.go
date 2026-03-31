package main

import (
	"flag"
	"io"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runExportSnapshots(args []string) error {
	fs := flag.NewFlagSet("export-snapshots", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var dir string
	fs.StringVar(&dir, "dir", "fixtures/snapshots", "snapshot output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	onboardingBundle, err := onboarding.BuildOnboardingProposal("")
	if err != nil {
		return err
	}
	if err := validation.ValidateOnboardingProposal(onboardingBundle); err != nil {
		return err
	}
	if err := writeArtifact(dir, "onboarding.proposal.json", onboardingBundle); err != nil {
		return err
	}

	inspectOnboardInput, err := buildDefaultInput("inspect")
	if err != nil {
		return err
	}
	inspectOnboardInput = applyOnboardingFailure(inspectOnboardInput, onboardingBundle)
	inspectOnboardOut, err := executeBundle("inspect", inspectOnboardInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "inspect.onboard.bundle.json", inspectOnboardOut); err != nil {
		return err
	}

	previewOnboardInput, err := buildDefaultInput("preview")
	if err != nil {
		return err
	}
	previewOnboardInput = applyOnboardingFailure(previewOnboardInput, onboardingBundle)
	previewOnboardOut, err := executeBundle("preview", previewOnboardInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "preview.onboard.bundle.json", previewOnboardOut); err != nil {
		return err
	}

	simulateOnboardInput, err := buildDefaultInput("simulate")
	if err != nil {
		return err
	}
	simulateOnboardInput = applyOnboardingFailure(simulateOnboardInput, onboardingBundle)
	simulateOnboardOut, err := executeBundle("simulate", simulateOnboardInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "simulate.onboard.bundle.json", simulateOnboardOut); err != nil {
		return err
	}

	inspectInput, err := buildDefaultInput("inspect")
	if err != nil {
		return err
	}
	inspectOut, err := executeBundle("inspect", inspectInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "inspect.bundle.json", inspectOut); err != nil {
		return err
	}

	previewInput, err := buildDefaultInput("preview")
	if err != nil {
		return err
	}
	previewOut, err := executeBundle("preview", previewInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "preview.bundle.json", previewOut); err != nil {
		return err
	}

	simulateInput, err := buildDefaultInput("simulate")
	if err != nil {
		return err
	}
	simulateOut, err := executeBundle("simulate", simulateInput)
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "simulate.bundle.json", simulateOut); err != nil {
		return err
	}

	report, err := runFixturesReport()
	if err != nil {
		return err
	}
	if err := writeArtifact(dir, "fixtures.report.json", report); err != nil {
		return err
	}

	return nil
}

