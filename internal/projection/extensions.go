package projection

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/util"
)

const extensionInterfaceID = "extensions"

func ExtensionDesired(model CanonicalModel) []Desired {
	if model.Contract.Extensions == nil {
		return nil
	}
	out := make([]Desired, 0)

	appendEntry := func(kind, id string) {
		kind = strings.TrimSpace(strings.ToLower(kind))
		id = strings.TrimSpace(id)
		if kind == "" || id == "" {
			return
		}
		templateID := extensionTemplateID(kind, id)
		content, ok := SyntheticTemplateContent(templateID)
		if !ok {
			return
		}
		path := extensionPath(kind, id)
		contentHash := util.SHA256Tagged([]byte(content))
		out = append(out, Desired{
			Interface:   extensionInterfaceID,
			TemplateID:  templateID,
			Path:        path,
			Source:      ".atrakta/contract.json",
			Target:      "",
			Fingerprint: Fingerprint(model.ContractHash, templateID, contentHash),
		})
	}

	appendSlice := func(kind string, rows []contract.ExtensionEntry) {
		for _, row := range rows {
			if !isExtensionEnabled(row.Enabled) {
				continue
			}
			appendEntry(kind, row.ID)
		}
	}

	appendSlice("mcp", model.Contract.Extensions.MCP)
	appendSlice("plugin", model.Contract.Extensions.Plugins)
	appendSlice("skill", model.Contract.Extensions.Skills)
	appendSlice("workflow", model.Contract.Extensions.Workflows)

	if h := model.Contract.Extensions.Hooks; h != nil {
		if h.Shell != nil {
			if boolEnabled(h.Shell.OnCD) {
				appendEntry("hook", "shell.on_cd")
			}
			if boolEnabled(h.Shell.OnExec) {
				appendEntry("hook", "shell.on_exec")
			}
		}
		if h.Git != nil {
			if boolEnabled(h.Git.PreCommit) {
				appendEntry("hook", "git.pre_commit")
			}
			if boolEnabled(h.Git.PrePush) {
				appendEntry("hook", "git.pre_push")
			}
		}
		if h.IDE != nil {
			if boolEnabled(h.IDE.OnOpen) {
				appendEntry("hook", "ide.on_open")
			}
		}
		if h.Workflow != nil {
			if boolEnabled(h.Workflow.BeforeStart) {
				appendEntry("hook", "workflow.before_start")
			}
			if boolEnabled(h.Workflow.AfterApply) {
				appendEntry("hook", "workflow.after_apply")
			}
		}
	}

	sortDesired(out)
	return out
}

func ParseExtensionTemplateID(templateID string) (kind, id string, ok bool) {
	raw := strings.TrimSpace(templateID)
	if !strings.HasPrefix(raw, "extensions:") || !strings.HasSuffix(raw, "@1") {
		return "", "", false
	}
	body := strings.TrimSuffix(strings.TrimPrefix(raw, "extensions:"), "@1")
	parts := strings.SplitN(body, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	kind = strings.TrimSpace(strings.ToLower(parts[0]))
	if kind == "" {
		return "", "", false
	}
	decoded, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", "", false
	}
	return kind, decoded, true
}

func ExtensionContent(templateID string) (string, bool) {
	kind, id, ok := ParseExtensionTemplateID(templateID)
	if !ok {
		return "", false
	}
	body := strings.TrimSpace(fmt.Sprintf(`
# Atrakta Extension Projection

- kind: %s
- id: %s
- mode: fallback_markdown
- source_of_truth: .atrakta/contract.json

This file is generated deterministically from the extensions contract.
When native integration is unavailable, this fallback is used instead of silent ignore.
`, kind, id)) + "\n"
	return body, true
}

func extensionTemplateID(kind, id string) string {
	return "extensions:" + strings.TrimSpace(strings.ToLower(kind)) + ":" + url.PathEscape(strings.TrimSpace(id)) + "@1"
}

func extensionPath(kind, id string) string {
	k := strings.TrimSpace(strings.ToLower(kind))
	encoded := url.PathEscape(strings.TrimSpace(id)) + ".md"
	dir := "hooks"
	switch k {
	case "mcp":
		dir = "mcp"
	case "plugin":
		dir = "plugins"
	case "skill":
		dir = "skills"
	case "workflow":
		dir = "workflows"
	case "hook":
		dir = "hooks"
	}
	p := filepath.ToSlash(filepath.Join(".extensions", dir, encoded))
	return util.NormalizeRelPath(p)
}

func isExtensionEnabled(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

func boolEnabled(v *bool) bool {
	return v != nil && *v
}
