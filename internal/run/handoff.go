package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const handoffSchemaVersion = "handoff.v1"

type HandoffCheckpoint struct {
	AutoStatePath   string `json:"auto_state_path,omitempty"`
	StartFastPath   string `json:"start_fast_path,omitempty"`
	RunStatePath    string `json:"run_state_path,omitempty"`
	StatePath       string `json:"state_path,omitempty"`
	ProgressPath    string `json:"progress_path,omitempty"`
	TaskGraphPath   string `json:"task_graph_path,omitempty"`
	AuditHeadPath   string `json:"audit_head_path,omitempty"`
	OnboardingState string `json:"onboarding_state_path,omitempty"`
}

type HandoffNextAction struct {
	Command string `json:"command,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type HandoffFeatureSpec struct {
	Summary           string   `json:"summary,omitempty"`
	ResolvedTargets   []string `json:"resolved_projection_targets,omitempty"`
	DegradedSurfaces  []string `json:"degraded_surfaces,omitempty"`
	MissingTargets    []string `json:"missing_projection_targets,omitempty"`
	PortabilityStatus string   `json:"portability_status,omitempty"`
	PortabilityReason string   `json:"portability_reason,omitempty"`
}

type HandoffBundle struct {
	SchemaVersion     string             `json:"schema_version"`
	UpdatedAt         string             `json:"updated_at,omitempty"`
	Command           string             `json:"command"`
	CanonicalState    string             `json:"canonical_state"`
	Status            string             `json:"status"`
	Message           string             `json:"message,omitempty"`
	InterfaceID       string             `json:"interface_id"`
	InterfaceSource   string             `json:"interface_source"`
	ApplyRequested    bool               `json:"apply_requested"`
	Approved          bool               `json:"approved"`
	FastPath          bool               `json:"fast_path,omitempty"`
	PortabilityStatus string             `json:"portability_status,omitempty"`
	PortabilityReason string             `json:"portability_reason,omitempty"`
	NextAllowedAction string             `json:"next_allowed_action,omitempty"`
	NextAction        HandoffNextAction  `json:"next_action,omitempty"`
	FeatureSpec       HandoffFeatureSpec `json:"feature_spec,omitempty"`
	Acceptance        []string           `json:"acceptance,omitempty"`
	PlannedTargets    []string           `json:"planned_targets,omitempty"`
	AppliedTargets    []string           `json:"applied_targets,omitempty"`
	Checkpoint        HandoffCheckpoint  `json:"checkpoint"`
}

func HandoffPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "runtime", "handoff.v1.json")
}

func LoadHandoff(projectRoot string) (HandoffBundle, error) {
	path := HandoffPath(projectRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		return HandoffBundle{}, err
	}
	var out HandoffBundle
	if err := json.Unmarshal(raw, &out); err != nil {
		return HandoffBundle{}, err
	}
	return out, nil
}

func SaveHandoff(projectRoot string, handoff HandoffBundle) error {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(handoff.SchemaVersion) == "" {
		handoff.SchemaVersion = handoffSchemaVersion
	}
	if strings.TrimSpace(handoff.UpdatedAt) == "" {
		handoff.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	path := HandoffPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(handoff, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
