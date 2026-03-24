package run

import (
	"errors"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

var ErrInterfaceUnresolved = errors.New("interface unresolved")

type InterfaceResolution struct {
	InterfaceID string `json:"interface_id"`
	Source      string `json:"source"`
}

// ResolveInterface resolves interface ID from explicit input, adapter trigger, or project detection.
func ResolveInterface(projectRoot, explicit, trigger string) (InterfaceResolution, error) {
	if explicit != "" {
		return InterfaceResolution{InterfaceID: explicit, Source: "flag"}, nil
	}
	if trigger != "" {
		return InterfaceResolution{InterfaceID: trigger, Source: "env"}, nil
	}

	assets, err := onboarding.DetectAssets(projectRoot)
	if err != nil {
		return InterfaceResolution{}, err
	}
	assetSet := make(map[string]struct{}, len(assets))
	for _, a := range assets {
		assetSet[a] = struct{}{}
	}

	switch {
	case hasAsset(assetSet, ".cursor", ".cursor/rules"):
		return InterfaceResolution{InterfaceID: "cursor", Source: "detect"}, nil
	case hasAsset(assetSet, ".vscode"):
		return InterfaceResolution{InterfaceID: "vscode", Source: "detect"}, nil
	case hasAsset(assetSet, ".mcp.json", "mcp.json"):
		return InterfaceResolution{InterfaceID: "mcp", Source: "detect"}, nil
	case hasAsset(assetSet, "AGENTS.md"):
		return InterfaceResolution{InterfaceID: "generic-cli", Source: "detect"}, nil
	case hasAsset(assetSet, ".github/workflows"):
		return InterfaceResolution{InterfaceID: "github-actions", Source: "detect"}, nil
	default:
		return InterfaceResolution{}, ErrInterfaceUnresolved
	}
}

func hasAsset(set map[string]struct{}, names ...string) bool {
	for _, name := range names {
		if _, ok := set[name]; ok {
			return true
		}
	}
	return false
}
