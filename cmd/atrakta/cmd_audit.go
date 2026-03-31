package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

func runAudit(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: atrakta audit <append|verify> [flags]")
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "append":
		fs := flag.NewFlagSet("audit append", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var projectRoot string
		var level string
		var action string
		var payloadPath string
		var artifactDir string
		fs.StringVar(&projectRoot, "project-root", "", "project root")
		fs.StringVar(&level, "level", audit.LevelA2, "audit integrity level")
		fs.StringVar(&action, "action", "", "audit action")
		fs.StringVar(&payloadPath, "payload-file", "", "payload json file")
		fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		if action == "" {
			return fmt.Errorf("--action is required")
		}

		root, err := onboarding.DetectProjectRoot(projectRoot)
		if err != nil {
			return err
		}
		payload := map[string]any{}
		if payloadPath != "" {
			b, err := os.ReadFile(payloadPath)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &payload); err != nil {
				return err
			}
		}

		event, err := audit.AppendAndVerify(filepath.Join(root, ".atrakta", "audit"), level, action, payload)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(event); err != nil {
			return err
		}
		if artifactDir != "" {
			if err := writeArtifact(artifactDir, "audit.append.json", event); err != nil {
				return err
			}
		}
		return nil

	case "verify":
		fs := flag.NewFlagSet("audit verify", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var projectRoot string
		var level string
		var artifactDir string
		fs.StringVar(&projectRoot, "project-root", "", "project root")
		fs.StringVar(&level, "level", audit.LevelA2, "audit integrity level")
		fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
		if err := fs.Parse(subArgs); err != nil {
			return err
		}
		root, err := onboarding.DetectProjectRoot(projectRoot)
		if err != nil {
			return err
		}
		err = audit.VerifyIntegrity(filepath.Join(root, ".atrakta", "audit"), level)
		out := map[string]any{
			"project_root": root,
			"level":        level,
			"ok":           err == nil,
		}
		if err != nil {
			out["error"] = err.Error()
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return err
		}
		if artifactDir != "" {
			if err := writeArtifact(artifactDir, "audit.verify.json", out); err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported audit subcommand %q", sub)
	}
}
