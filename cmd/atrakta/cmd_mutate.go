package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mash4649/atrakta/v0/internal/mutation"
	checkmutationscope "github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
)

func runMutate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: atrakta mutate <inspect|propose|apply> [flags]")
	}
	phase := args[0]
	phaseArgs := args[1:]

	fs := flag.NewFlagSet("mutate "+phase, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var targetPath string
	var declaredScope string
	var assetType string
	var operation string
	var content string
	var contentFile string
	var projectRoot string
	var allow bool
	var artifactDir string
	fs.StringVar(&targetPath, "target", "", "target path")
	fs.StringVar(&declaredScope, "declared-scope", "", "declared scope override")
	fs.StringVar(&assetType, "asset-type", "", "asset type")
	fs.StringVar(&operation, "operation", "", "operation type")
	fs.StringVar(&content, "content", "", "inline content")
	fs.StringVar(&contentFile, "content-file", "", "content file path")
	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&allow, "allow", false, "explicitly allow apply")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(phaseArgs); err != nil {
		return err
	}
	if targetPath == "" {
		return fmt.Errorf("--target is required")
	}

	target := checkmutationscope.Target{
		Path:          targetPath,
		DeclaredScope: declaredScope,
		AssetType:     assetType,
		Operation:     operation,
	}

	resolveContent := func() (string, error) {
		if contentFile != "" {
			b, err := os.ReadFile(contentFile)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
		return content, nil
	}

	var output any
	var err error

	switch phase {
	case "inspect":
		output = mutation.Inspect(target)
	case "propose":
		var body string
		body, err = resolveContent()
		if err != nil {
			return err
		}
		output = mutation.Propose(target, body)
	case "apply":
		var body string
		body, err = resolveContent()
		if err != nil {
			return err
		}
		output, err = mutation.Apply(projectRoot, target, body, allow)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported mutate phase %q", phase)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "mutate."+phase+".json", output); err != nil {
			return err
		}
	}
	return nil
}

