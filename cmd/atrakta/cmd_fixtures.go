package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mash4649/atrakta/v0/internal/fixtures"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runFixtures(args []string) error {
	fs := flag.NewFlagSet("run-fixtures", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var artifactDir string
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	report, err := runFixturesReport()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "fixtures.report.json", report); err != nil {
			return err
		}
	}
	if report.Failed > 0 {
		return fmt.Errorf("fixture failures: %d", report.Failed)
	}
	return nil
}

func runFixturesReport() (fixtures.Report, error) {
	fixturesDir, err := resolveFixturesDir()
	if err != nil {
		return fixtures.Report{}, err
	}
	report, err := fixtures.RunAll(fixturesDir)
	if err != nil {
		return fixtures.Report{}, err
	}
	if err := validation.ValidateFixtureReport(report); err != nil {
		return fixtures.Report{}, err
	}
	return report, nil
}

func resolveFixturesDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "fixtures")
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("fixtures directory not found from current path")
}

