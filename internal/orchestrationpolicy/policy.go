package orchestrationpolicy

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/harnessprofile"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const SchemaVersion = "selective_orchestration_policy.v1"

// Report captures which orchestration components are load-bearing for the current acceptance/profile combination.
type Report struct {
	SchemaVersion           string            `json:"schema_version"`
	ProjectRoot             string            `json:"project_root"`
	ModelGeneration         string            `json:"model_generation"`
	AcceptanceSpecVersion   string            `json:"acceptance_spec_version"`
	AcceptanceRubricVersion string            `json:"acceptance_rubric_version"`
	ProfileName             string            `json:"profile_name"`
	PlannerEnabled          bool              `json:"planner_enabled"`
	EvaluatorEnabled        bool              `json:"evaluator_enabled"`
	CheckpointEnabled       bool              `json:"checkpoint_enabled"`
	EnabledComponents       []string          `json:"enabled_components"`
	DisabledComponents      []string          `json:"disabled_components"`
	Reasons                 map[string]string `json:"reasons"`
}

// Build creates a deterministic selective orchestration policy report.
func Build(projectRoot string, spec onboarding.AcceptanceSpec, rubric onboarding.AcceptanceRubric, profile harnessprofile.Report) Report {
	reasons := map[string]string{}
	enabled := make([]string, 0, 3)
	disabled := make([]string, 0, 3)

	plannerEnabled := hasAcceptanceWork(spec) && contains(profile.LoadBearingComponents, "planner")
	if plannerEnabled {
		reasons["planner"] = "acceptance artifacts require executable planning and the profile keeps planner load-bearing"
		enabled = append(enabled, "planner")
	} else {
		reasons["planner"] = "planner is not load-bearing for this acceptance/profile combination"
		disabled = append(disabled, "planner")
	}

	evaluatorEnabled := hasRubricWork(rubric) && contains(profile.LoadBearingComponents, "evaluator")
	if evaluatorEnabled {
		reasons["evaluator"] = "acceptance rubric requires browser-backed verification and the profile keeps evaluator load-bearing"
		enabled = append(enabled, "evaluator")
	} else {
		reasons["evaluator"] = "evaluator is not load-bearing for this acceptance/profile combination"
		disabled = append(disabled, "evaluator")
	}

	checkpointEnabled := requiresCheckpoint(spec) && contains(profile.LoadBearingComponents, "reset")
	if checkpointEnabled {
		reasons["checkpoint"] = "acceptance next-action recovery is load-bearing and the profile keeps reset load-bearing"
		enabled = append(enabled, "checkpoint")
	} else {
		if requiresCheckpoint(spec) {
			reasons["checkpoint"] = "reset is retirable for this model generation or acceptance recovery is not required"
		} else {
			reasons["checkpoint"] = "acceptance artifacts do not require checkpoint-backed recovery"
		}
		disabled = append(disabled, "checkpoint")
	}

	sort.Strings(enabled)
	sort.Strings(disabled)

	return Report{
		SchemaVersion:           SchemaVersion,
		ProjectRoot:             projectRoot,
		ModelGeneration:         profile.ModelGeneration,
		AcceptanceSpecVersion:   spec.SchemaVersion,
		AcceptanceRubricVersion: rubric.SchemaVersion,
		ProfileName:             profile.ProfileName,
		PlannerEnabled:          plannerEnabled,
		EvaluatorEnabled:        evaluatorEnabled,
		CheckpointEnabled:       checkpointEnabled,
		EnabledComponents:       enabled,
		DisabledComponents:      disabled,
		Reasons:                 reasons,
	}
}

func hasAcceptanceWork(spec onboarding.AcceptanceSpec) bool {
	return strings.TrimSpace(spec.SchemaVersion) != "" &&
		len(spec.RequiredOutputs) > 0 &&
		len(spec.AcceptanceCriteria) > 0
}

func hasRubricWork(rubric onboarding.AcceptanceRubric) bool {
	return strings.TrimSpace(rubric.SchemaVersion) != "" && len(rubric.Criteria) > 0
}

func requiresCheckpoint(spec onboarding.AcceptanceSpec) bool {
	for _, criterion := range spec.AcceptanceCriteria {
		if strings.Contains(strings.ToLower(criterion), "next actions remain actionable") {
			return true
		}
	}
	return false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
