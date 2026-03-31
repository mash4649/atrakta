package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/extensions"
)

func runExtensions(args []string) error {
	fs := flag.NewFlagSet("extensions", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var projectRoot string
	var artifactDir string
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	out, err := extensions.Resolve(projectRoot)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "extensions.resolve.json", out); err != nil {
			return err
		}
	}
	return nil
}

