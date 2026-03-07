package registry

import (
	"sort"

	"atrakta/internal/contract"
)

type Entry struct {
	InterfaceID   string
	Surface       string
	Provider      string
	Anchor        string
	ProjectionDir string
	WrapperNames  []string
	FamilyID      string
}

type Registry struct {
	Entries map[string]Entry
}

func Default() Registry {
	entries := []Entry{
		{InterfaceID: "vscode", Surface: "editor", Provider: "unknown", Anchor: ".vscode/", ProjectionDir: ".vscode/", WrapperNames: []string{"code", "vscode"}, FamilyID: "vscode_family"},
		{InterfaceID: "cursor", Surface: "editor", Provider: "unknown", Anchor: ".cursor/", ProjectionDir: ".cursor/", WrapperNames: []string{"cursor"}, FamilyID: "vscode_family"},
		{InterfaceID: "windsurf", Surface: "editor", Provider: "unknown", Anchor: ".windsurf/", ProjectionDir: ".windsurf/", WrapperNames: []string{"windsurf"}, FamilyID: "vscode_family"},
		{InterfaceID: "trae", Surface: "editor", Provider: "unknown", Anchor: ".trae/", ProjectionDir: ".trae/", WrapperNames: []string{"trae"}, FamilyID: "vscode_family"},
		{InterfaceID: "antigravity", Surface: "editor", Provider: "unknown", Anchor: ".antigravity/", ProjectionDir: ".antigravity/", WrapperNames: []string{"antigravity"}, FamilyID: "vscode_family"},
		{InterfaceID: "github_copilot", Surface: "editor", Provider: "github", Anchor: ".vscode/", ProjectionDir: ".vscode/", WrapperNames: []string{"copilot"}, FamilyID: "vscode_family"},
		{InterfaceID: "aider", Surface: "cli", Provider: "unknown", Anchor: "", ProjectionDir: "", WrapperNames: []string{"aider"}, FamilyID: "terminal_cli_family"},
		{InterfaceID: "codex_cli", Surface: "cli", Provider: "openai", Anchor: "", ProjectionDir: "", WrapperNames: []string{"codex", "codex-cli"}, FamilyID: "terminal_cli_family"},
		{InterfaceID: "gemini_cli", Surface: "cli", Provider: "google", Anchor: "", ProjectionDir: "", WrapperNames: []string{"gemini", "gemini-cli"}, FamilyID: "terminal_cli_family"},
		{InterfaceID: "claude_code", Surface: "cli", Provider: "anthropic", Anchor: "", ProjectionDir: "", WrapperNames: []string{"claude", "claude-code"}, FamilyID: "terminal_cli_family"},
		{InterfaceID: "opencode", Surface: "cli", Provider: "unknown", Anchor: "", ProjectionDir: "", WrapperNames: []string{"opencode"}, FamilyID: "terminal_cli_family"},
	}
	m := make(map[string]Entry, len(entries))
	for _, e := range entries {
		m[e.InterfaceID] = e
	}
	return Registry{Entries: m}
}

func ApplyOverrides(reg Registry, c contract.Contract) Registry {
	out := Registry{Entries: map[string]Entry{}}
	for k, v := range reg.Entries {
		out.Entries[k] = v
	}
	supported := contract.SupportedSet(c)
	for id := range out.Entries {
		if _, ok := supported[id]; !ok {
			delete(out.Entries, id)
		}
	}
	if c.Hints != nil {
		for _, id := range c.Hints.DisableInterfaces {
			delete(out.Entries, id)
		}
		for id, anchor := range c.Hints.Anchors {
			e, ok := out.Entries[id]
			if !ok {
				continue
			}
			e.Anchor = anchor
			out.Entries[id] = e
		}
	}
	return out
}

func (r Registry) InterfaceIDs() []string {
	ids := make([]string, 0, len(r.Entries))
	for id := range r.Entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r Registry) ResolveByWrapperName(name string) (Entry, bool) {
	for _, e := range r.Entries {
		for _, w := range e.WrapperNames {
			if w == name {
				return e, true
			}
		}
	}
	return Entry{}, false
}
