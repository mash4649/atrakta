package orchestrationpolicy

import (
	"testing"

	"github.com/mash4649/atrakta/v0/internal/harnessprofile"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

func TestBuildCurrentGenerationPolicyEnablesAllComponents(t *testing.T) {
	spec, rubric := acceptanceInputs()
	profile, err := harnessprofile.Generate(t.TempDir(), "current")
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}

	report := Build("/tmp/project", spec, rubric, profile)
	if !report.PlannerEnabled || !report.EvaluatorEnabled || !report.CheckpointEnabled {
		t.Fatalf("expected all orchestration components enabled: %+v", report)
	}
	if len(report.EnabledComponents) != 3 {
		t.Fatalf("enabled components=%v", report.EnabledComponents)
	}
	if len(report.DisabledComponents) != 0 {
		t.Fatalf("disabled components=%v", report.DisabledComponents)
	}
}

func TestBuildNextGenerationPolicyRetiresCheckpoint(t *testing.T) {
	spec, rubric := acceptanceInputs()
	profile, err := harnessprofile.Generate(t.TempDir(), "gpt-5.4")
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}

	report := Build("/tmp/project", spec, rubric, profile)
	if !report.PlannerEnabled {
		t.Fatalf("planner should remain enabled: %+v", report)
	}
	if !report.EvaluatorEnabled {
		t.Fatalf("evaluator should remain enabled: %+v", report)
	}
	if report.CheckpointEnabled {
		t.Fatalf("checkpoint should be retired for next generation: %+v", report)
	}
	if len(report.EnabledComponents) != 2 || len(report.DisabledComponents) != 1 || report.DisabledComponents[0] != "checkpoint" {
		t.Fatalf("component partition mismatch: %+v", report)
	}
}

func acceptanceInputs() (onboarding.AcceptanceSpec, onboarding.AcceptanceRubric) {
	spec := onboarding.AcceptanceSpec{
		SchemaVersion: onboarding.SchemaVersionAcceptanceSpec,
		Prompt:        "Generate acceptance artifacts for a brownfield onboarding prompt with detected assets: AGENTS.md, docs.",
		Summary:       "Executable acceptance spec and scoring rubric derived from the onboarding proposal.",
		Mode:          onboarding.ModeBrownfield,
		DetectedAssets: []string{
			"AGENTS.md",
			"docs",
		},
		RequiredOutputs: []string{
			"acceptance-spec.generated.json",
			"acceptance-rubric.generated.json",
		},
		AcceptanceCriteria: []string{
			"proposal validates against the onboarding contract",
			"executable acceptance spec is written to the generated store",
			"scoring rubric is written to the generated store",
			"next actions remain actionable after acceptance",
		},
	}
	rubric := onboarding.AcceptanceRubric{
		SchemaVersion: onboarding.SchemaVersionAcceptanceRubric,
		PassThreshold: 80,
		MaxScore:      100,
		Criteria: []onboarding.RubricCriterion{
			{
				ID:          "spec_written",
				Description: "The executable spec is generated and persisted.",
				Points:      40,
				Evidence:    []string{"generated/acceptance-spec.generated.json"},
			},
			{
				ID:          "rubric_written",
				Description: "The scoring rubric is generated and persisted.",
				Points:      30,
				Evidence:    []string{"generated/acceptance-rubric.generated.json"},
			},
		},
	}
	return spec, rubric
}
