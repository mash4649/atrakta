package bindings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
)

type AutostartConfig struct {
	Kind    string `json:"kind,omitempty"`
	Path    string `json:"path,omitempty"`
	Command string `json:"command,omitempty"`
	Format  string `json:"format,omitempty"`
}

type Definition struct {
	ID                    string           `json:"id"`
	Kind                  string           `json:"kind,omitempty"`
	Surfaces              []string         `json:"surfaces,omitempty"`
	ProjectionTargets     []string         `json:"projection_targets,omitempty"`
	IngestSources         []string         `json:"ingest_sources,omitempty"`
	ApprovalChannel       string           `json:"approval_channel,omitempty"`
	PortabilityMode       string           `json:"portability_mode,omitempty"`
	CanMutateCoreContract bool             `json:"can_mutate_core_contract,omitempty"`
	InstallPath           string           `json:"install_path,omitempty"`
	ScriptTemplate        string           `json:"script_template,omitempty"`
	Capabilities          []string         `json:"capabilities,omitempty"`
	AutostartConfig       *AutostartConfig `json:"autostart_config,omitempty"`
}

// Load returns binding capabilities for the requested interface.
// Unknown interfaces degrade to an explicit unsupported binding.
func Load(interfaceID string) (resolvesurfaceportability.BindingCapabilities, error) {
	id := normalize(interfaceID)
	if id == "" {
		return resolvesurfaceportability.BindingCapabilities{
			InterfaceID:     "unknown",
			ApprovalChannel: "unsupported",
			PortabilityMode: resolvesurfaceportability.PortabilityModeUnsupported,
		}, nil
	}

	root, err := resolveRepoRoot()
	if err != nil {
		return resolvesurfaceportability.BindingCapabilities{}, err
	}

	path := filepath.Join(root, "adapters", "bindings", id, "binding.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return resolvesurfaceportability.BindingCapabilities{
				InterfaceID:     id,
				ApprovalChannel: "unsupported",
				PortabilityMode: resolvesurfaceportability.PortabilityModeUnsupported,
			}, nil
		}
		return resolvesurfaceportability.BindingCapabilities{}, err
	}

	var caps resolvesurfaceportability.BindingCapabilities
	if err := json.Unmarshal(b, &caps); err != nil {
		return resolvesurfaceportability.BindingCapabilities{}, err
	}
	if caps.InterfaceID == "" {
		caps.InterfaceID = id
	}
	return caps.Normalize(), nil
}

// List returns all binding definitions shipped with the repository.
func List() ([]Definition, error) {
	root, err := resolveRepoRoot()
	if err != nil {
		return nil, err
	}

	paths, err := filepath.Glob(filepath.Join(root, "adapters", "bindings", "*", "binding.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	defs := make([]Definition, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read binding %s: %w", path, err)
		}
		var def Definition
		if err := json.Unmarshal(raw, &def); err != nil {
			return nil, fmt.Errorf("parse binding %s: %w", path, err)
		}
		if strings.TrimSpace(def.ID) == "" {
			def.ID = filepath.Base(filepath.Dir(path))
		}
		defs = append(defs, def)
	}
	return defs, nil
}

func resolveRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
