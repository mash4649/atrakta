package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/harnessprofile"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runHarnessProfile(args []string) error {
	fs := flag.NewFlagSet("harness-profile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var modelGeneration string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&modelGeneration, "model-generation", "current", "model generation label")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	report, err := harnessprofile.Generate(projectRoot, modelGeneration)
	if err != nil {
		return err
	}
	if err := validation.ValidateHarnessProfile(report); err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "harness-profile.report.json", report); err != nil {
			return err
		}
		if err := maybeWriteSelectiveOrchestrationPolicy(projectRoot, modelGeneration, artifactDir); err != nil {
			return err
		}
	}
	return nil
}
