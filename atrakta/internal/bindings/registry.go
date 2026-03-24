package bindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
)

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
