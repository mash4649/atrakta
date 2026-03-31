package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/harnessprofile"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/orchestrationpolicy"
)

const selectiveOrchestrationPolicyArtifact = "selective-orchestration-policy.report.json"

func maybeWriteSelectiveOrchestrationPolicy(projectRoot, modelGeneration, artifactDir string) error {
	if strings.TrimSpace(artifactDir) == "" {
		return nil
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}

	spec, rubric, ok, err := loadAcceptanceArtifacts(root)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	profile, err := harnessprofile.Generate(root, modelGeneration)
	if err != nil {
		return err
	}

	report := orchestrationpolicy.Build(root, spec, rubric, profile)
	if err := writeArtifact(artifactDir, selectiveOrchestrationPolicyArtifact, report); err != nil {
		return err
	}
	return nil
}

func loadAcceptanceArtifacts(projectRoot string) (onboarding.AcceptanceSpec, onboarding.AcceptanceRubric, bool, error) {
	specPath := filepath.Join(projectRoot, ".atrakta", "generated", "acceptance-spec.generated.json")
	rubricPath := filepath.Join(projectRoot, ".atrakta", "generated", "acceptance-rubric.generated.json")

	specRaw, specErr := os.ReadFile(specPath)
	rubricRaw, rubricErr := os.ReadFile(rubricPath)
	if os.IsNotExist(specErr) && os.IsNotExist(rubricErr) {
		return onboarding.AcceptanceSpec{}, onboarding.AcceptanceRubric{}, false, nil
	}
	if specErr != nil {
		return onboarding.AcceptanceSpec{}, onboarding.AcceptanceRubric{}, false, specErr
	}
	if rubricErr != nil {
		return onboarding.AcceptanceSpec{}, onboarding.AcceptanceRubric{}, false, rubricErr
	}

	var spec onboarding.AcceptanceSpec
	if err := json.Unmarshal(specRaw, &spec); err != nil {
		return onboarding.AcceptanceSpec{}, onboarding.AcceptanceRubric{}, false, err
	}
	var rubric onboarding.AcceptanceRubric
	if err := json.Unmarshal(rubricRaw, &rubric); err != nil {
		return onboarding.AcceptanceSpec{}, onboarding.AcceptanceRubric{}, false, err
	}
	return spec, rubric, true, nil
}
