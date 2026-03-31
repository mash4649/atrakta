package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/performance"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

func runBenchmark(args []string) error {
	if len(args) == 0 {
		return flag.ErrHelp
	}
	switch args[0] {
	case "start-latency":
		return runStartLatencyBenchmark(args[1:])
	default:
		return flag.ErrHelp
	}
}

func runStartLatencyBenchmark(args []string) error {
	fs := flag.NewFlagSet("benchmark start-latency", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var iterations int
	var interfaceID string
	var workspaceProfile string
	var artifactDir string
	fs.IntVar(&iterations, "iterations", 3, "number of benchmark samples")
	fs.StringVar(&interfaceID, "interface", "generic-cli", "interface id")
	fs.StringVar(&workspaceProfile, "workspace", "small", "workspace profile (small|monorepo)")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	report, err := performance.MeasureStartLatency(performance.Input{
		Iterations:       iterations,
		InterfaceID:      interfaceID,
		WorkspaceProfile: workspaceProfile,
	}, benchmarkStartRunner)
	if err != nil {
		return err
	}
	if err := validation.ValidateStartLatencyBenchmark(report); err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return err
	}
	if strings.TrimSpace(artifactDir) != "" {
		if err := writeArtifact(artifactDir, "start-latency.report.json", report); err != nil {
			return err
		}
	}
	return nil
}

func benchmarkStartRunner(args []string) (performance.Result, error) {
	artifactDir := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--artifact-dir" && i+1 < len(args) {
			artifactDir = args[i+1]
			break
		}
	}
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return performance.Result{}, err
	}
	defer devNull.Close()
	os.Stdout = devNull
	os.Stderr = devNull
	code, runErr := startCommand(args)
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	if runErr != nil {
		return performance.Result{ExitCode: code}, runErr
	}
	resultPath := ""
	if strings.TrimSpace(artifactDir) != "" {
		candidate := filepath.Join(artifactDir, "start.result.json")
		if _, statErr := os.Stat(candidate); statErr == nil {
			resultPath = candidate
		}
	}
	fastPath := false
	if resultPath != "" {
		raw, readErr := os.ReadFile(resultPath)
		if readErr == nil {
			var out map[string]any
			if jsonErr := json.Unmarshal(raw, &out); jsonErr == nil {
				if summary, ok := out["canonical_summary"].(map[string]any); ok {
					fastPath, _ = summary["fast_path"].(bool)
				}
			}
		}
	}
	return performance.Result{
		ExitCode:   code,
		FastPath:   fastPath,
		ResultPath: resultPath,
	}, nil
}
