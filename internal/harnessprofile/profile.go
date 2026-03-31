package harnessprofile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const SchemaVersion = "harness_profile.v1"

type Step struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Weight      int      `json:"weight"`
	Evidence    []string `json:"evidence,omitempty"`
}

type Ablation struct {
	Component     string   `json:"component"`
	BaselineScore int      `json:"baseline_score"`
	AblatedScore  int      `json:"ablated_score"`
	Delta         int      `json:"delta"`
	LoadBearing   bool     `json:"load_bearing"`
	Impact        string   `json:"impact"`
	Evidence      []string `json:"evidence,omitempty"`
}

type Report struct {
	SchemaVersion         string     `json:"schema_version"`
	ProjectRoot           string     `json:"project_root"`
	ModelGeneration       string     `json:"model_generation"`
	ProfileName           string     `json:"profile_name"`
	Threshold             int        `json:"threshold"`
	BaselineScore         int        `json:"baseline_score"`
	Steps                 []Step     `json:"steps"`
	Ablations             []Ablation `json:"ablations"`
	LoadBearingComponents []string   `json:"load_bearing_components"`
	RetirableComponents   []string   `json:"retirable_components"`
}

type weights struct {
	planner   int
	evaluator int
	reset     int
}

// Generate builds a deterministic harness profile and ablation report.
func Generate(projectRoot, modelGeneration string) (Report, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return Report{}, err
	}

	gen := normalizeGeneration(modelGeneration)
	w := weightsForGeneration(gen)
	steps := []Step{
		{
			Name:        "planner",
			Description: "acceptance artifacts and prompt-to-spec generation",
			Weight:      w.planner,
			Evidence: []string{
				"internal/onboarding/acceptance.go",
				"cmd/atrakta/cmd_onboard.go",
				"cmd/atrakta/cmd_accept.go",
			},
		},
		{
			Name:        "evaluator",
			Description: "browser-backed acceptance verification",
			Weight:      w.evaluator,
			Evidence: []string{
				"internal/evaluator/evaluator.go",
				"cmd/atrakta/run-fixtures",
			},
		},
		{
			Name:        "reset",
			Description: "handoff / auto-state resume recovery",
			Weight:      w.reset,
			Evidence: []string{
				"internal/run/handoff.go",
				"internal/run/auto_state.go",
				"cmd/atrakta/cmd_resume.go",
			},
		},
	}

	baseline := 100
	threshold := 80
	ablations := make([]Ablation, 0, len(steps))
	loadBearing := make([]string, 0, len(steps))
	retirable := make([]string, 0, len(steps))
	for _, step := range steps {
		ablated := baseline - step.Weight
		ablation := Ablation{
			Component:     step.Name,
			BaselineScore: baseline,
			AblatedScore:  ablated,
			Delta:         step.Weight,
			LoadBearing:   ablated < threshold,
			Impact:        impactForStep(step.Name, ablated < threshold),
			Evidence:      append([]string(nil), step.Evidence...),
		}
		ablations = append(ablations, ablation)
		if ablation.LoadBearing {
			loadBearing = append(loadBearing, step.Name)
		} else {
			retirable = append(retirable, step.Name)
		}
	}

	sort.Strings(loadBearing)
	sort.Strings(retirable)

	return Report{
		SchemaVersion:         SchemaVersion,
		ProjectRoot:           root,
		ModelGeneration:       gen,
		ProfileName:           profileNameForGeneration(gen),
		Threshold:             threshold,
		BaselineScore:         baseline,
		Steps:                 steps,
		Ablations:             ablations,
		LoadBearingComponents: loadBearing,
		RetirableComponents:   retirable,
	}, nil
}

func normalizeGeneration(gen string) string {
	trimmed := strings.TrimSpace(strings.ToLower(gen))
	if trimmed == "" {
		return "current"
	}
	return trimmed
}

func weightsForGeneration(gen string) weights {
	switch {
	case strings.Contains(gen, "next") || strings.Contains(gen, "gpt-5"):
		return weights{planner: 35, evaluator: 45, reset: 20}
	case strings.Contains(gen, "legacy"):
		return weights{planner: 35, evaluator: 25, reset: 40}
	default:
		return weights{planner: 40, evaluator: 35, reset: 25}
	}
}

func profileNameForGeneration(gen string) string {
	switch {
	case strings.Contains(gen, "next") || strings.Contains(gen, "gpt-5"):
		return "next_generation_harness"
	case strings.Contains(gen, "legacy"):
		return "legacy_harness"
	default:
		return "current_harness"
	}
}

func impactForStep(step string, loadBearing bool) string {
	if loadBearing {
		return fmt.Sprintf("%s remains load-bearing; removing it drops the profile below the benchmark threshold", step)
	}
	return fmt.Sprintf("%s is not load-bearing for the selected model generation", step)
}
