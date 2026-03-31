package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runOnboard(args []string) error {
	fs := flag.NewFlagSet("onboard", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root to inspect")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	bundle, err := onboarding.BuildOnboardingProposal(projectRoot)
	if err != nil {
		return err
	}
	if err := validation.ValidateOnboardingProposal(bundle); err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(bundle); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "onboarding.proposal.json", bundle); err != nil {
			return err
		}
		if err := writeAcceptanceArtifacts(artifactDir, bundle); err != nil {
			return err
		}
	}
	return nil
}
