package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/util"
)

type ResolveInput struct {
	RepoRoot string
	StartDir string
	Config   *contract.Context
}

type ResolveReport struct {
	Root              string   `json:"root"`
	Resolved          []string `json:"resolved,omitempty"`
	Imports           []string `json:"imports,omitempty"`
	ConventionsLoaded []string `json:"conventions_loaded,omitempty"`
	Depth             int      `json:"depth"`
	Fingerprint       string   `json:"fingerprint,omitempty"`
}

func Resolve(in ResolveInput) (string, ResolveReport, error) {
	repoRoot := filepath.Clean(in.RepoRoot)
	startDir := strings.TrimSpace(in.StartDir)
	if startDir == "" {
		startDir = repoRoot
	}
	startDir = filepath.Clean(startDir)

	baseChain, err := collectBaseAgents(repoRoot, startDir, in.Config)
	if err != nil {
		return "", ResolveReport{}, err
	}
	if len(baseChain) == 0 {
		return "", ResolveReport{}, fmt.Errorf("AGENTS.md not found")
	}

	maxDepth := 6
	if in.Config != nil && in.Config.MaxImportDepth > 0 {
		maxDepth = in.Config.MaxImportDepth
	}

	resolved := []string{}
	imports := []string{}
	seen := map[string]struct{}{}
	active := map[string]struct{}{}
	var maxSeenDepth int
	var outParts []string

	var visit func(abs string, depth int) error
	visit = func(abs string, depth int) error {
		abs = filepath.Clean(abs)
		if depth > maxDepth {
			return fmt.Errorf("context import depth exceeded (%d)", maxDepth)
		}
		if depth > maxSeenDepth {
			maxSeenDepth = depth
		}
		if _, ok := active[abs]; ok {
			return fmt.Errorf("context import cycle detected: %s", relPath(repoRoot, abs))
		}
		if _, ok := seen[abs]; ok {
			return nil
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("read context source %s: %w", relPath(repoRoot, abs), err)
		}
		active[abs] = struct{}{}
		defer delete(active, abs)
		seen[abs] = struct{}{}
		resolved = append(resolved, relPath(repoRoot, abs))
		outParts = append(outParts, strings.TrimRight(util.NormalizeContentLF(string(b)), "\n"))
		importPaths := parseImportPaths(string(b))
		for _, imp := range importPaths {
			next, ok := resolveImport(repoRoot, filepath.Dir(abs), imp)
			if !ok {
				return fmt.Errorf("invalid context import %q in %s", imp, relPath(repoRoot, abs))
			}
			imports = append(imports, relPath(repoRoot, next))
			if err := visit(next, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	for _, p := range baseChain {
		if err := visit(p, 1); err != nil {
			return "", ResolveReport{}, err
		}
	}
	loadedConventions := []string{}
	budget := conventionsTokenBudget()
	for _, rel := range conventionsFromConfig(in.Config) {
		abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
		fi, err := os.Stat(abs)
		if err != nil || fi.IsDir() {
			continue
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return "", ResolveReport{}, fmt.Errorf("read conventions source %s: %w", relPath(repoRoot, abs), err)
		}
		snippet, usedTokens := buildConventionSnippet(rel, string(b), budget)
		if snippet == "" {
			continue
		}
		if _, ok := seen[abs]; !ok {
			seen[abs] = struct{}{}
			resolved = append(resolved, relPath(repoRoot, abs))
		}
		outParts = append(outParts, strings.TrimRight(util.NormalizeContentLF(snippet), "\n"))
		loadedConventions = append(loadedConventions, rel)
		budget -= usedTokens
		if budget <= 0 {
			break
		}
	}
	text := strings.TrimSpace(strings.Join(outParts, "\n\n")) + "\n"
	report := ResolveReport{
		Root:              relPath(repoRoot, baseChain[0]),
		Resolved:          uniqStable(resolved),
		Imports:           uniqStable(imports),
		ConventionsLoaded: uniqStable(loadedConventions),
		Depth:             maxSeenDepth,
		Fingerprint:       util.SHA256Tagged([]byte(text)),
	}
	return text, report, nil
}

func collectBaseAgents(repoRoot, startDir string, cfg *contract.Context) ([]string, error) {
	resolution := ""
	if cfg != nil {
		resolution = strings.TrimSpace(strings.ToLower(cfg.Resolution))
	}
	rootAgents := filepath.Join(repoRoot, "AGENTS.md")
	if resolution != "nearest_with_import" {
		if _, err := os.Stat(rootAgents); err != nil {
			return nil, err
		}
		return []string{rootAgents}, nil
	}
	// lower -> upper chain
	out := []string{}
	cur := startDir
	for {
		if !isWithin(repoRoot, cur) {
			break
		}
		candidate := filepath.Join(cur, "AGENTS.md")
		if _, err := os.Stat(candidate); err == nil {
			out = append(out, candidate)
		}
		if filepath.Clean(cur) == filepath.Clean(repoRoot) {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	if len(out) == 0 {
		if _, err := os.Stat(rootAgents); err != nil {
			return nil, err
		}
		out = append(out, rootAgents)
	}
	return out, nil
}

func parseImportPaths(text string) []string {
	lines := strings.Split(util.NormalizeContentLF(text), "\n")
	out := make([]string, 0, 8)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(strings.ToLower(line), "import:") {
			continue
		}
		value := strings.TrimSpace(line[len("import:"):])
		if value == "" {
			continue
		}
		parts := strings.Split(value, ",")
		for _, part := range parts {
			p := strings.TrimSpace(part)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}

func resolveImport(repoRoot, baseDir, ref string) (string, bool) {
	p := strings.TrimSpace(strings.ReplaceAll(ref, "\\", "/"))
	if p == "" {
		return "", false
	}
	abs := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(p)))
	if !isWithin(repoRoot, abs) {
		return "", false
	}
	if fi, err := os.Stat(abs); err == nil && !fi.IsDir() {
		return abs, true
	}
	return "", false
}

func isWithin(root, child string) bool {
	root = filepath.Clean(root)
	child = filepath.Clean(child)
	if child == root {
		return true
	}
	prefix := root + string(filepath.Separator)
	return strings.HasPrefix(child, prefix)
}

func relPath(repoRoot, abs string) string {
	rel, err := filepath.Rel(repoRoot, abs)
	if err != nil {
		return util.NormalizeRelPath(abs)
	}
	return util.NormalizeRelPath(rel)
}

func uniqStable(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func conventionsFromConfig(cfg *contract.Context) []string {
	if cfg == nil || len(cfg.Conventions) == 0 {
		return []string{"CONVENTIONS.md", "docs/CONVENTIONS.md"}
	}
	out := make([]string, 0, len(cfg.Conventions))
	seen := map[string]struct{}{}
	for _, p := range cfg.Conventions {
		n := util.NormalizeRelPath(p)
		if n == "" || strings.HasPrefix(n, "../") {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
