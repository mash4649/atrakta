package projection

import (
	"fmt"
	"sort"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
	"atrakta/internal/util"
)

// CanonicalModel is the normalized renderer input shared across interfaces.
type CanonicalModel struct {
	Contract     contract.Contract
	ContractHash string
	SourceText   map[string]string
	SourceHash   map[string]string
}

type Renderer interface {
	Render(repoRoot string, model CanonicalModel, interfaceID, projectionDir string) ([]Desired, error)
}

type Engine struct {
	defaultRenderer    Renderer
	interfaceRenderers map[string]Renderer
}

func DefaultEngine() Engine {
	return Engine{
		defaultRenderer: agentsRenderer{},
		interfaceRenderers: map[string]Renderer{
			"claude_code": claudeRenderer{},
			"codex_cli":   codexRenderer{},
		},
	}
}

func BuildCanonicalModel(c contract.Contract, contractHash, sourceAGENTS string) CanonicalModel {
	canon := contract.CanonicalizeBoundary(c)
	def := contract.Default(".")
	if canon.Parity == nil {
		canon.Parity = def.Parity
	}
	if canon.Extensions == nil {
		canon.Extensions = def.Extensions
	}
	agents := util.NormalizeContentLF(sourceAGENTS)
	agentsHash := util.SHA256Tagged([]byte(agents))
	return CanonicalModel{
		Contract:     canon,
		ContractHash: contractHash,
		SourceText: map[string]string{
			"AGENTS.md": agents,
		},
		SourceHash: map[string]string{
			"AGENTS.md": agentsHash,
		},
	}
}

func (e Engine) RenderTargets(repoRoot string, model CanonicalModel, reg registry.Registry, targets []string) ([]Desired, error) {
	if e.defaultRenderer == nil {
		return nil, fmt.Errorf("projection renderer is not configured")
	}
	ids := uniqueSortedTargets(targets)
	out := make([]Desired, 0, len(ids))
	for _, id := range ids {
		entry, ok := reg.Entries[id]
		if !ok {
			continue
		}
		r, hasCustom := e.interfaceRenderers[id]
		if !hasCustom {
			if entry.ProjectionDir == "" {
				continue
			}
			r = e.defaultRenderer
		}
		rows, err := r.Render(repoRoot, model, id, entry.ProjectionDir)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	sortDesired(out)
	return out, nil
}

func StableRenderHash(rows []Desired) (string, error) {
	normalized := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		normalized = append(normalized, map[string]any{
			"interface":   strings.TrimSpace(r.Interface),
			"template_id": strings.TrimSpace(r.TemplateID),
			"path":        util.NormalizeRelPath(r.Path),
			"source":      util.NormalizeRelPath(r.Source),
			"target":      util.NormalizeRelPath(r.Target),
			"fingerprint": strings.TrimSpace(r.Fingerprint),
		})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		a := normalized[i]
		b := normalized[j]
		if a["path"] != b["path"] {
			return a["path"].(string) < b["path"].(string)
		}
		if a["template_id"] != b["template_id"] {
			return a["template_id"].(string) < b["template_id"].(string)
		}
		return a["interface"].(string) < b["interface"].(string)
	})
	b, err := util.MarshalCanonical(map[string]any{"entries": normalized})
	if err != nil {
		return "", fmt.Errorf("canonical render hash: %w", err)
	}
	return util.SHA256Tagged(b), nil
}

type agentsRenderer struct{}

func (agentsRenderer) Render(repoRoot string, model CanonicalModel, interfaceID, projectionDir string) ([]Desired, error) {
	agentsHash := model.SourceHash["AGENTS.md"]
	if agentsHash == "" {
		agentsHash = util.SHA256Tagged([]byte(util.NormalizeContentLF(model.SourceText["AGENTS.md"])))
	}
	target := util.NormalizeRelPath(projectionDir + "/AGENTS.md")
	templateID := interfaceID + ":agents-md@1"
	out := []Desired{{
		Interface:   interfaceID,
		TemplateID:  templateID,
		Path:        target,
		Source:      "AGENTS.md",
		Target:      "AGENTS.md",
		Fingerprint: Fingerprint(model.ContractHash, templateID, agentsHash),
	}}
	opt, err := optionalTemplates(repoRoot, model.Contract, interfaceID, projectionDir, model.ContractHash)
	if err != nil {
		return nil, err
	}
	out = append(out, opt...)
	sortDesired(out)
	return out, nil
}

type claudeRenderer struct{}

func (claudeRenderer) Render(repoRoot string, model CanonicalModel, interfaceID, projectionDir string) ([]Desired, error) {
	agentsHash := model.SourceHash["AGENTS.md"]
	if agentsHash == "" {
		agentsHash = util.SHA256Tagged([]byte(util.NormalizeContentLF(model.SourceText["AGENTS.md"])))
	}
	out := []Desired{
		{
			Interface:   interfaceID,
			TemplateID:  interfaceID + ":claude-md@1",
			Path:        "CLAUDE.md",
			Source:      "AGENTS.md",
			Target:      "AGENTS.md",
			Fingerprint: Fingerprint(model.ContractHash, interfaceID+":claude-md@1", agentsHash),
		},
	}
	for _, item := range []struct {
		templateID string
		path       string
	}{
		{templateID: interfaceID + ":settings-json@1", path: ".claude/settings.json"},
		{templateID: interfaceID + ":mcp-json@1", path: ".claude/mcp.json"},
		{templateID: interfaceID + ":agents-md@1", path: ".claude/agents/atrakta.md"},
	} {
		content, _ := SyntheticTemplateContent(item.templateID)
		contentHash := util.SHA256Tagged([]byte(content))
		out = append(out, Desired{
			Interface:   interfaceID,
			TemplateID:  item.templateID,
			Path:        item.path,
			Source:      "AGENTS.md",
			Target:      "",
			Fingerprint: Fingerprint(model.ContractHash, item.templateID, contentHash),
		})
	}
	sortDesired(out)
	return out, nil
}

type codexRenderer struct{}

func (codexRenderer) Render(repoRoot string, model CanonicalModel, interfaceID, projectionDir string) ([]Desired, error) {
	agentsHash := model.SourceHash["AGENTS.md"]
	if agentsHash == "" {
		agentsHash = util.SHA256Tagged([]byte(util.NormalizeContentLF(model.SourceText["AGENTS.md"])))
	}
	out := []Desired{
		{
			Interface:   interfaceID,
			TemplateID:  interfaceID + ":agents-md@1",
			Path:        "AGENTS.md",
			Source:      "AGENTS.md",
			Target:      "AGENTS.md",
			Fingerprint: Fingerprint(model.ContractHash, interfaceID+":agents-md@1", agentsHash),
		},
	}
	configTemplateID := interfaceID + ":config-toml@1"
	content, _ := SyntheticTemplateContent(configTemplateID)
	contentHash := util.SHA256Tagged([]byte(content))
	out = append(out, Desired{
		Interface:   interfaceID,
		TemplateID:  configTemplateID,
		Path:        ".codex/config.toml",
		Source:      "AGENTS.md",
		Target:      "",
		Fingerprint: Fingerprint(model.ContractHash, configTemplateID, contentHash),
	})
	sortDesired(out)
	return out, nil
}

func uniqueSortedTargets(targets []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(targets))
	for _, t := range targets {
		id := strings.TrimSpace(t)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func sortDesired(rows []Desired) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Path != rows[j].Path {
			return rows[i].Path < rows[j].Path
		}
		if rows[i].TemplateID != rows[j].TemplateID {
			return rows[i].TemplateID < rows[j].TemplateID
		}
		return rows[i].Interface < rows[j].Interface
	})
}
