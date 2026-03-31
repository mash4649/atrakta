package onboarding

import (
	"fmt"
	"strings"
)

const (
	SchemaVersionAcceptanceSpec   = "acceptance_spec.v1"
	SchemaVersionAcceptanceRubric = "acceptance_rubric.v1"
)

// AcceptanceSpec captures the executable spec derived from an onboarding proposal.
type AcceptanceSpec struct {
	SchemaVersion      string   `json:"schema_version"`
	Prompt             string   `json:"prompt"`
	Summary            string   `json:"summary"`
	Mode               string   `json:"mode"`
	DetectedAssets     []string `json:"detected_assets"`
	RequiredOutputs    []string `json:"required_outputs"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

// RubricCriterion defines one scoring dimension for the acceptance rubric.
type RubricCriterion struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Points      int      `json:"points"`
	Evidence    []string `json:"evidence"`
}

// AcceptanceRubric captures the scoring rubric derived from an onboarding proposal.
type AcceptanceRubric struct {
	SchemaVersion string            `json:"schema_version"`
	PassThreshold int               `json:"pass_threshold"`
	MaxScore      int               `json:"max_score"`
	Criteria      []RubricCriterion `json:"criteria"`
}

// BuildAcceptanceArtifacts derives the executable spec and scoring rubric for onboarding acceptance.
func BuildAcceptanceArtifacts(bundle ProposalBundle) (AcceptanceSpec, AcceptanceRubric) {
	spec := AcceptanceSpec{
		SchemaVersion:      SchemaVersionAcceptanceSpec,
		Prompt:             buildAcceptancePrompt(bundle),
		Summary:            buildAcceptanceSummary(bundle),
		Mode:               bundle.InferredMode,
		DetectedAssets:     append([]string(nil), bundle.DetectedAssets...),
		RequiredOutputs:    buildAcceptanceRequiredOutputs(),
		AcceptanceCriteria: buildAcceptanceCriteria(bundle),
	}
	rubric := AcceptanceRubric{
		SchemaVersion: SchemaVersionAcceptanceRubric,
		PassThreshold: 80,
		MaxScore:      100,
		Criteria:      buildAcceptanceRubricCriteria(bundle),
	}
	return spec, rubric
}

func buildAcceptancePrompt(bundle ProposalBundle) string {
	assets := "no detected assets"
	if len(bundle.DetectedAssets) > 0 {
		assets = strings.Join(bundle.DetectedAssets, ", ")
	}
	return fmt.Sprintf("Generate acceptance artifacts for a %s onboarding prompt with detected assets: %s.", bundle.InferredMode, assets)
}

func buildAcceptanceSummary(bundle ProposalBundle) string {
	if len(bundle.Conflicts) == 0 {
		return "Executable acceptance spec and scoring rubric derived from the onboarding proposal."
	}
	return "Conflict-aware executable acceptance spec and scoring rubric derived from the onboarding proposal."
}

func buildAcceptanceRequiredOutputs() []string {
	return []string{
		"acceptance-spec.generated.json",
		"acceptance-rubric.generated.json",
	}
}

func buildAcceptanceCriteria(bundle ProposalBundle) []string {
	criteria := []string{
		"proposal validates against the onboarding contract",
		"executable acceptance spec is written to the generated store",
		"scoring rubric is written to the generated store",
	}
	if len(bundle.Conflicts) > 0 {
		criteria = append(criteria, "conflicts are reflected in the acceptance prompt and rubric")
	}
	if len(bundle.SuggestedNextActions) > 0 {
		criteria = append(criteria, "next actions remain actionable after acceptance")
	}
	return criteria
}

func buildAcceptanceRubricCriteria(bundle ProposalBundle) []RubricCriterion {
	criteria := []RubricCriterion{
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
		{
			ID:          "traceable_prompt",
			Description: "The prompt traces back to the onboarding proposal and detected assets.",
			Points:      20,
			Evidence:    append([]string{"detected_assets", "inferred_mode"}, bundle.DetectedAssets...),
		},
		{
			ID:          "actionable_next_steps",
			Description: "The rubric keeps the next steps actionable after acceptance.",
			Points:      10,
			Evidence:    append([]string{"suggested_next_actions"}, bundle.SuggestedNextActions...),
		},
	}
	return criteria
}
