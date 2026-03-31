package run

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/bindings"
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

	if detected, ok, err := detectInterfaceFromBindings(projectRoot, assetSet); err != nil {
		return InterfaceResolution{}, err
	} else if ok {
		return detected, nil
	}

	switch {
	case hasAsset(assetSet, ".cursor", ".cursor/rules"):
		return InterfaceResolution{InterfaceID: "cursor", Source: "detect"}, nil
	case hasAsset(assetSet, ".vscode"):
		return InterfaceResolution{InterfaceID: "vscode", Source: "detect"}, nil
	case hasAsset(assetSet, "CLAUDE.md"):
		return InterfaceResolution{InterfaceID: "claude-code", Source: "detect"}, nil
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

type bindingCandidate struct {
	id    string
	score int
}

func detectInterfaceFromBindings(root string, assetSet map[string]struct{}) (InterfaceResolution, bool, error) {
	defs, err := bindings.List()
	if err != nil {
		return InterfaceResolution{}, false, err
	}

	candidates := make([]bindingCandidate, 0, len(defs))
	for _, def := range defs {
		if score := scoreBindingCandidate(root, def, assetSet); score > 0 {
			candidates = append(candidates, bindingCandidate{id: def.ID, score: score})
		}
	}
	if len(candidates) == 0 {
		return InterfaceResolution{}, false, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].id < candidates[j].id
	})

	return InterfaceResolution{InterfaceID: candidates[0].id, Source: "detect"}, true, nil
}

func scoreBindingCandidate(root string, def bindings.Definition, assetSet map[string]struct{}) int {
	score := 0

	if strings.TrimSpace(def.InstallPath) != "" && pathExists(filepath.Join(root, def.InstallPath)) {
		score += 100
	}
	if def.AutostartConfig != nil && strings.TrimSpace(def.AutostartConfig.Path) != "" && pathExists(filepath.Join(root, def.AutostartConfig.Path)) {
		score += 95
	}

	if containsAny(strings.ToLower(def.ID), "cursor") && hasAsset(assetSet, ".cursor", ".cursor/rules", ".cursor/autostart.json") {
		score += 90
	}
	if containsAny(strings.ToLower(def.ID), "vscode") && hasAsset(assetSet, ".vscode", ".vscode/tasks.json") {
		score += 90
	}
	if containsAny(strings.ToLower(def.ID), "github-actions") && hasAsset(assetSet, ".github/workflows") {
		score += 90
	}
	if containsAny(strings.ToLower(def.ID), "claude") && pathExists(filepath.Join(root, "CLAUDE.md")) {
		score += 85
	}
	if containsAny(strings.ToLower(def.ID), "generic-cli") && hasAsset(assetSet, "AGENTS.md") {
		score += 80
	}
	if containsAny(strings.ToLower(def.ID), "copilot") && hasAsset(assetSet, ".github/copilot", "copilot.md") {
		score += 75
	}

	if hasString(def.IngestSources, "workflow_binding") && hasAsset(assetSet, ".github/workflows") {
		score += 40
	}
	if hasString(def.IngestSources, "agents_md") && hasAsset(assetSet, "AGENTS.md") {
		score += 35
	}
	if hasString(def.IngestSources, "ide_rules") && hasAsset(assetSet, ".cursor", ".cursor/rules", ".vscode") {
		score += 30
	}
	if hasString(def.IngestSources, "repo_docs") && hasAsset(assetSet, "docs") {
		score += 20
	}
	if hasString(def.IngestSources, "skill_asset") && hasAsset(assetSet, ".agents/skills", "skills") {
		score += 15
	}

	if hasString(def.Surfaces, "autostart") && def.AutostartConfig != nil && strings.TrimSpace(def.AutostartConfig.Path) != "" && pathExists(filepath.Join(root, def.AutostartConfig.Path)) {
		score += 25
	}
	if hasString(def.Surfaces, "tool_hint") && hasAsset(assetSet, ".cursor", ".vscode", "AGENTS.md", "CLAUDE.md") {
		score += 10
	}
	if hasString(def.Surfaces, "diagnostics") && hasAsset(assetSet, "docs", "AGENTS.md", "CLAUDE.md") {
		score += 5
	}

	return score
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func hasString(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), needle) {
			return true
		}
	}
	return false
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}

func hasAsset(set map[string]struct{}, names ...string) bool {
	for _, name := range names {
		if _, ok := set[name]; ok {
			return true
		}
	}
	return false
}
