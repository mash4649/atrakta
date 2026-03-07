package projection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
	"atrakta/internal/util"
)

type Desired struct {
	Interface   string
	TemplateID  string
	Path        string
	Source      string
	Target      string
	Fingerprint string
}

func RequiredForTargets(repoRoot string, c contract.Contract, reg registry.Registry, targets []string, contractHash, sourceText string) ([]Desired, error) {
	canon := util.NormalizeContentLF(sourceText)
	sourceHash := util.SHA256Tagged([]byte(canon))

	out := []Desired{}
	for _, id := range targets {
		e, ok := reg.Entries[id]
		if !ok || e.ProjectionDir == "" {
			continue
		}
		target := util.NormalizeRelPath(filepath.ToSlash(filepath.Join(e.ProjectionDir, "AGENTS.md")))
		templateID := id + ":agents-md@1"
		fp := Fingerprint(contractHash, templateID, sourceHash)
		out = append(out, Desired{
			Interface:   id,
			TemplateID:  templateID,
			Path:        target,
			Source:      "AGENTS.md",
			Target:      "AGENTS.md",
			Fingerprint: fp,
		})
		opt, err := optionalTemplates(repoRoot, c, id, e.ProjectionDir, contractHash)
		if err != nil {
			return nil, err
		}
		out = append(out, opt...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].TemplateID < out[j].TemplateID
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func optionalTemplates(repoRoot string, c contract.Contract, id, projectionDir, contractHash string) ([]Desired, error) {
	if c.Projections == nil || len(c.Projections.OptionalTemplates) == 0 {
		return nil, nil
	}
	templates := c.Projections.OptionalTemplates[id]
	if len(templates) == 0 {
		return nil, nil
	}
	max := c.Projections.MaxPerInterface
	if max <= 0 {
		max = 3
	}
	if len(templates) > max {
		return nil, fmt.Errorf("optional template count exceeds max_per_interface for %s", id)
	}
	out := make([]Desired, 0, len(templates))
	for _, name := range templates {
		switch name {
		case "contract-json":
			source := ".atrakta/contract.json"
			b, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(source)))
			if err != nil {
				return nil, fmt.Errorf("read optional source %s: %w", source, err)
			}
			contentHash := util.SHA256Tagged([]byte(util.NormalizeContentLF(string(b))))
			templateID := id + ":contract-json@1"
			out = append(out, Desired{
				Interface:   id,
				TemplateID:  templateID,
				Path:        util.NormalizeRelPath(filepath.ToSlash(filepath.Join(projectionDir, "CONTRACT.json"))),
				Source:      source,
				Target:      source,
				Fingerprint: Fingerprint(contractHash, templateID, contentHash),
			})
		case "atrakta-link":
			source := ".atrakta/contract.json"
			marker := "ATRAKTA-LINK\n"
			contentHash := util.SHA256Tagged([]byte(marker))
			templateID := id + ":atrakta-link@1"
			out = append(out, Desired{
				Interface:   id,
				TemplateID:  templateID,
				Path:        util.NormalizeRelPath(filepath.ToSlash(filepath.Join(projectionDir, ".atrakta-link"))),
				Source:      source,
				Target:      source,
				Fingerprint: Fingerprint(contractHash, templateID, contentHash),
			})
		default:
			return nil, fmt.Errorf("unsupported optional template %q", name)
		}
	}
	return out, nil
}

func Fingerprint(contractHash, templateID, canonicalTemplateContentHash string) string {
	payload := contractHash + "|" + templateID + "|" + canonicalTemplateContentHash
	return util.SHA256Tagged([]byte(payload))
}

func ManagedHeader(templateID, fingerprint string) string {
	return ManagedHeaderForPath("AGENTS.md", templateID, fingerprint)
}

func ManagedHeaderForPath(path, templateID, fingerprint string) string {
	prefix := commentPrefix(path)
	if prefix == "" {
		return ""
	}
	lines := []string{
		prefix + " Managed by Atrakta",
		prefix + " template_id: " + templateID,
		prefix + " fingerprint: " + fingerprint,
	}
	return strings.Join(lines, "\n") + "\n"
}

func commentPrefix(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".ts", ".tsx", ".jsx", ".java", ".c", ".cc", ".cpp", ".h", ".hpp", ".rs", ".swift", ".kt":
		return "//"
	case ".py", ".sh", ".rb", ".yml", ".yaml", ".md", ".txt", ".ini", ".toml":
		return "#"
	case ".json":
		return ""
	default:
		return "#"
	}
}

func ManagedContentForPath(path, templateID, fingerprint, sourceText string) string {
	return ManagedHeaderForPath(path, templateID, fingerprint) + util.NormalizeContentLF(sourceText)
}

func ManagedContent(templateID, fingerprint, sourceText string) string {
	return ManagedContentForPath("AGENTS.md", templateID, fingerprint, sourceText)
}
