package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/persist"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runAccept(args []string) error {
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var proposalPath string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&proposalPath, "proposal", "", "onboarding proposal JSON path")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var bundle onboarding.ProposalBundle
	if proposalPath != "" {
		b, err := os.ReadFile(proposalPath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &bundle); err != nil {
			return err
		}
	} else {
		var err error
		bundle, err = onboarding.BuildOnboardingProposal(projectRoot)
		if err != nil {
			return err
		}
	}
	if err := validation.ValidateOnboardingProposal(bundle); err != nil {
		return err
	}

	result, err := persist.AcceptOnboarding(projectRoot, bundle)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "accept.result.json", result); err != nil {
			return err
		}
		if err := writeAcceptanceArtifacts(artifactDir, bundle); err != nil {
			return err
		}
	}
	return nil
}
