package brownfield

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"atrakta/internal/projection"
	"atrakta/internal/util"
)

type Detection struct {
	AGENTS       bool     `json:"agents"`
	CLAUDE       bool     `json:"claude"`
	CursorRules  bool     `json:"cursor_rules"`
	Tasks        []string `json:"tasks,omitempty"`
	ShellRC      []string `json:"shell_rc,omitempty"`
	PluginConfig []string `json:"plugin_config,omitempty"`
}

type Conflict struct {
	Path      string `json:"path"`
	Interface string `json:"interface,omitempty"`
	Reason    string `json:"reason"`
}

type ProposalInput struct {
	Mode          string     `json:"mode"`
	MergeStrategy string     `json:"merge_strategy"`
	AgentsMode    string     `json:"agents_mode"`
	NoOverwrite   bool       `json:"no_overwrite"`
	Interfaces    []string   `json:"interfaces"`
	Detection     Detection  `json:"detection"`
	Conflicts     []Conflict `json:"conflicts"`
}

func Detect(repoRoot string) (Detection, error) {
	out := Detection{
		Tasks:        []string{},
		ShellRC:      []string{},
		PluginConfig: []string{},
	}
	has, err := exists(filepath.Join(repoRoot, "AGENTS.md"))
	if err != nil {
		return Detection{}, err
	}
	out.AGENTS = has
	has, err = exists(filepath.Join(repoRoot, "CLAUDE.md"))
	if err != nil {
		return Detection{}, err
	}
	out.CLAUDE = has
	has, err = exists(filepath.Join(repoRoot, ".cursor", "rules"))
	if err != nil {
		return Detection{}, err
	}
	out.CursorRules = has

	for _, p := range []string{"tasks.json", filepath.Join(".vscode", "tasks.json")} {
		ok, err := exists(filepath.Join(repoRoot, filepath.FromSlash(p)))
		if err != nil {
			return Detection{}, err
		}
		if ok {
			out.Tasks = append(out.Tasks, p)
		}
	}

	home, _ := os.UserHomeDir()
	if strings.TrimSpace(home) != "" {
		for _, p := range []string{".zshrc", ".zprofile", ".bashrc", ".bash_profile"} {
			abs := filepath.Join(home, p)
			ok, err := exists(abs)
			if err != nil {
				return Detection{}, err
			}
			if ok {
				out.ShellRC = append(out.ShellRC, abs)
			}
		}
	}

	for _, p := range []string{
		filepath.Join(".cursor", "extensions.json"),
		filepath.Join(".vscode", "extensions.json"),
		filepath.Join(".claude", "mcp.json"),
		filepath.Join(".codex", "config.toml"),
	} {
		ok, err := exists(filepath.Join(repoRoot, filepath.FromSlash(p)))
		if err != nil {
			return Detection{}, err
		}
		if ok {
			out.PluginConfig = append(out.PluginConfig, p)
		}
	}
	if rows, err := filepath.Glob(filepath.Join(repoRoot, "plugins", "*.json")); err == nil {
		for _, abs := range rows {
			rel, _ := filepath.Rel(repoRoot, abs)
			out.PluginConfig = append(out.PluginConfig, filepath.ToSlash(rel))
		}
	}

	sort.Strings(out.Tasks)
	sort.Strings(out.ShellRC)
	sort.Strings(out.PluginConfig)
	return out, nil
}

func FindConflicts(repoRoot string, desired []projection.Desired, noOverwrite bool) ([]Conflict, error) {
	if !noOverwrite {
		return []Conflict{}, nil
	}
	conflicts := make([]Conflict, 0)
	seen := map[string]struct{}{}
	for _, d := range desired {
		path := util.NormalizeRelPath(d.Path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}

		abs := filepath.Join(repoRoot, filepath.FromSlash(path))
		ok, err := exists(abs)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if path == "AGENTS.md" && d.Interface == "codex_cli" {
			continue
		}
		managed, err := isManagedProjection(abs)
		if err != nil {
			return nil, err
		}
		if managed {
			continue
		}
		conflicts = append(conflicts, Conflict{
			Path:      path,
			Interface: d.Interface,
			Reason:    "existing user-managed file would be overwritten",
		})
	}
	sort.SliceStable(conflicts, func(i, j int) bool {
		if conflicts[i].Path != conflicts[j].Path {
			return conflicts[i].Path < conflicts[j].Path
		}
		return conflicts[i].Interface < conflicts[j].Interface
	})
	return conflicts, nil
}

func WriteProposalPatch(repoRoot string, in ProposalInput) (string, error) {
	dir := filepath.Join(repoRoot, ".atrakta", "proposals")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir proposals dir: %w", err)
	}
	name := fmt.Sprintf("brownfield-init-%s.patch", time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, name)

	lines := []string{
		"# Atrakta brownfield init proposal",
		"# This file is generated when --mode brownfield --no-overwrite detects merge conflicts.",
		fmt.Sprintf("# mode=%s merge_strategy=%s agents_mode=%s no_overwrite=%v", in.Mode, in.MergeStrategy, in.AgentsMode, in.NoOverwrite),
		fmt.Sprintf("# interfaces=%s", strings.Join(in.Interfaces, ",")),
		"",
		"# Detection",
		fmt.Sprintf("# AGENTS.md=%v CLAUDE.md=%v cursor_rules=%v", in.Detection.AGENTS, in.Detection.CLAUDE, in.Detection.CursorRules),
	}
	if len(in.Detection.Tasks) > 0 {
		lines = append(lines, "# tasks="+strings.Join(in.Detection.Tasks, ","))
	}
	if len(in.Detection.PluginConfig) > 0 {
		lines = append(lines, "# plugin_config="+strings.Join(in.Detection.PluginConfig, ","))
	}
	lines = append(lines, "", "# Conflicts")
	for _, c := range in.Conflicts {
		lines = append(lines,
			fmt.Sprintf("# - path=%s interface=%s reason=%s", c.Path, c.Interface, c.Reason),
			fmt.Sprintf("--- a/%s", c.Path),
			fmt.Sprintf("+++ b/%s", c.Path),
			"@@",
			"+# TODO: merge Atrakta managed projection content for this path",
			"",
		)
	}
	lines = append(lines,
		"# Suggested commands",
		"# atrakta projection status --json",
		"# atrakta projection render --all",
		"# atrakta doctor --parity --json",
		"",
	)

	body := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write proposal patch: %w", err)
	}
	return path, nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func isManagedProjection(absPath string) (bool, error) {
	fi, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	if !fi.Mode().IsRegular() {
		return false, nil
	}
	b, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}
	text := string(b)
	if strings.Contains(text, "Managed by Atrakta") {
		return true, nil
	}
	if strings.Contains(text, "ATRAKTA_MANAGED:START") {
		return true, nil
	}
	return false, nil
}
