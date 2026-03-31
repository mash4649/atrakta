package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const autoStateSchemaVersion = "auto-state.v1"

type AutoState struct {
	SchemaVersion   string `json:"schema_version"`
	InterfaceID     string `json:"interface_id"`
	InterfaceSource string `json:"interface_source"`
}

func AutoStatePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "runtime", "auto-state.v1.json")
}

func LoadAutoState(projectRoot string) (AutoState, error) {
	path := AutoStatePath(projectRoot)
	raw, err := os.ReadFile(path)
	if err != nil {
		return AutoState{}, err
	}
	var out AutoState
	if err := json.Unmarshal(raw, &out); err != nil {
		return AutoState{}, err
	}
	return out, nil
}

func SaveAutoState(projectRoot string, state AutoState) error {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	if strings.TrimSpace(state.SchemaVersion) == "" {
		state.SchemaVersion = autoStateSchemaVersion
	}
	path := AutoStatePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
