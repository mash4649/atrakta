package wrapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
	"atrakta/internal/runtimecache"
	"atrakta/internal/state"
	"atrakta/internal/util"
)

type CacheEntry struct {
	ContractHash       string   `json:"contract_hash"`
	WorkspaceStamp     string   `json:"workspace_stamp"`
	DeepWorkspaceStamp string   `json:"deep_workspace_stamp,omitempty"`
	HitCount           int      `json:"hit_count,omitempty"`
	LastTargetSet      []string `json:"last_target_set"`
	ProjectionSummary  string   `json:"projection_fingerprint_summary"`
}

type Cache map[string]CacheEntry

const (
	wrapperCacheKey      = "wrapper_cache"
	deepSampleInterval   = 20
	wrapperCacheTTLHours = 24 * 30
)

func Install(selfExe string) error {
	reg := registry.Default()
	binDir, err := userBinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("mkdir wrapper bin dir: %w", err)
	}
	for _, id := range reg.InterfaceIDs() {
		e := reg.Entries[id]
		for _, name := range e.WrapperNames {
			realPath := findRealExecutable(name, binDir)
			scriptPath := filepath.Join(binDir, name)
			content := wrapperScript(selfExe, id, realPath)
			if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
				return fmt.Errorf("write wrapper %s: %w", scriptPath, err)
			}
		}
	}
	if changed, err := ensurePathPriority(binDir); err != nil {
		return err
	} else if changed {
		fmt.Printf("updated shell rc to prioritize %s on PATH\n", binDir)
	}
	fmt.Printf("wrappers installed at %s\n", binDir)
	return nil
}

func Uninstall() error {
	reg := registry.Default()
	binDir, err := userBinDir()
	if err != nil {
		return err
	}
	for _, id := range reg.InterfaceIDs() {
		e := reg.Entries[id]
		for _, name := range e.WrapperNames {
			_ = os.Remove(filepath.Join(binDir, name))
		}
	}
	fmt.Printf("wrappers removed from %s\n", binDir)
	return nil
}

func Run(selfExe, iface, real string, passthroughArgs []string) int {
	if os.Getenv("ATRAKTA_WRAP_DISABLE") == "1" || os.Getenv("ATRAKTA_WRAP_ACTIVE") == "1" {
		return launch(real, passthroughArgs)
	}

	wd, _ := os.Getwd()
	repoRoot, ok := findRepoRoot(wd)
	if !ok {
		return launch(real, passthroughArgs)
	}

	contractPath := filepath.Join(repoRoot, ".atrakta", "contract.json")
	cb, err := os.ReadFile(contractPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wrapper warning: failed reading contract: %v\n", err)
		return launch(real, passthroughArgs)
	}
	var c contract.Contract
	if err := json.Unmarshal(cb, &c); err != nil {
		fmt.Fprintf(os.Stderr, "wrapper warning: bad contract: %v\n", err)
		return launch(real, passthroughArgs)
	}
	stamp := util.WorkspaceStamp(repoRoot, c.Boundary.Include)
	projSummary := projectionSummary(repoRoot, iface)
	cache := loadCache(repoRoot)
	key := iface
	base := CacheEntry{
		ContractHash:      contract.ContractHash(cb),
		WorkspaceStamp:    stamp,
		LastTargetSet:     []string{iface},
		ProjectionSummary: projSummary,
	}
	if ce, ok := cache[key]; ok {
		if ce.ContractHash == base.ContractHash && ce.ProjectionSummary == base.ProjectionSummary && sameTargets(ce.LastTargetSet, base.LastTargetSet) {
			quickEqual := ce.WorkspaceStamp == base.WorkspaceStamp
			if quickEqual {
				needsDeep := ce.HitCount > 0 && ce.HitCount%deepSampleInterval == 0
				if !needsDeep {
					ce.HitCount++
					cache[key] = ce
					saveCache(repoRoot, cache)
					return launch(real, passthroughArgs)
				}
				deep := util.WorkspaceStampDeep(repoRoot)
				if deep == "" || ce.DeepWorkspaceStamp == "" || deep == ce.DeepWorkspaceStamp {
					ce.HitCount++
					if deep != "" {
						ce.DeepWorkspaceStamp = deep
					}
					cache[key] = ce
					saveCache(repoRoot, cache)
					return launch(real, passthroughArgs)
				}
			} else {
				// Fast stamp changed. Re-check with deep stamp before forcing start.
				deep := util.WorkspaceStampDeep(repoRoot)
				if deep != "" && ce.DeepWorkspaceStamp != "" && deep == ce.DeepWorkspaceStamp {
					ce.WorkspaceStamp = base.WorkspaceStamp
					ce.HitCount++
					cache[key] = ce
					saveCache(repoRoot, cache)
					return launch(real, passthroughArgs)
				}
			}
		}
	}

	cmd := exec.Command(selfExe, "start")
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		"ATRAKTA_WRAP_ACTIVE=1",
		"ATRAKTA_TRIGGER_SOURCE=wrapper",
		"ATRAKTA_TRIGGER_INTERFACE="+iface,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wrapper warning: atrakta start failed: %v\n", err)
	}

	base.WorkspaceStamp = util.WorkspaceStamp(repoRoot, c.Boundary.Include)
	base.DeepWorkspaceStamp = util.WorkspaceStampDeep(repoRoot)
	base.ProjectionSummary = projectionSummary(repoRoot, iface)
	base.HitCount = 1
	cache[key] = base
	saveCache(repoRoot, cache)

	return launch(real, passthroughArgs)
}

func wrapperScript(selfExe, iface, real string) string {
	if real == "" {
		real = iface
	}
	return "#!/bin/sh\n" +
		"exec \"" + selfExe + "\" wrap run --interface \"" + iface + "\" --real \"" + real + "\" -- \"$@\"\n"
}

func findRealExecutable(name, userBin string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	if filepath.Clean(filepath.Dir(p)) == filepath.Clean(userBin) {
		return ""
	}
	return p
}

func userBinDir() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "bin"), nil
}

func ensurePathPriority(binDir string) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	rcFiles := []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".bashrc")}
	changed := false
	for _, rc := range rcFiles {
		ok, err := ensurePathSnippet(rc, binDir)
		if err != nil {
			return false, err
		}
		if ok {
			changed = true
		}
	}
	return changed, nil
}

func ensurePathSnippet(rcPath, binDir string) (bool, error) {
	const begin = "# >>> atrakta path >>>"
	const end = "# <<< atrakta path <<<"
	binExpr := shellPathExpr(binDir)
	snippet := begin + "\n" +
		"case \":$PATH:\" in\n" +
		"  *:\"" + binExpr + "\":*) ;;\n" +
		"  *) export PATH=\"" + binExpr + ":$PATH\" ;;\n" +
		"esac\n" +
		end + "\n"

	b, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read %s: %w", rcPath, err)
	}
	text := string(b)
	if strings.Contains(text, begin) && strings.Contains(text, end) {
		return false, nil
	}
	if !strings.HasSuffix(text, "\n") && text != "" {
		text += "\n"
	}
	text += snippet
	if err := os.WriteFile(rcPath, []byte(text), 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", rcPath, err)
	}
	return true, nil
}

func shellPathExpr(binDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return binDir
	}
	cleanHome := filepath.Clean(home)
	cleanBin := filepath.Clean(binDir)
	if cleanBin == filepath.Join(cleanHome, ".local", "bin") {
		return "$HOME/.local/bin"
	}
	if strings.HasPrefix(cleanBin, cleanHome+string(os.PathSeparator)) {
		suffix := strings.TrimPrefix(cleanBin, cleanHome)
		return "$HOME" + filepath.ToSlash(suffix)
	}
	return filepath.ToSlash(cleanBin)
}

func loadCache(repoRoot string) Cache {
	st, err := runtimecache.Load(repoRoot)
	if err != nil {
		return Cache{}
	}
	e, ok := st.Entries[wrapperCacheKey]
	if !ok {
		return Cache{}
	}
	out := Cache{}
	if runtimecache.UnmarshalPayload(e.Payload, &out) != nil {
		return Cache{}
	}
	return out
}

func saveCache(repoRoot string, c Cache) {
	_ = runtimecache.Update(repoRoot, func(st *runtimecache.State) error {
		st.Entries[wrapperCacheKey] = runtimecache.Entry{
			UpdatedAt:  util.NowUTC(),
			TTLSeconds: wrapperCacheTTLHours * 3600,
			Payload:    runtimecache.MarshalPayload(c),
		}
		return nil
	})
}

func sameTargets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func findRepoRoot(start string) (string, bool) {
	cur := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(cur, ".atrakta", "contract.json")); err == nil {
			return cur, true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", false
		}
		cur = parent
	}
}

func projectionSummary(repoRoot, iface string) string {
	s, _, err := state.LoadOrEmpty(repoRoot, "")
	if err != nil {
		return ""
	}
	rows := []string{}
	for p, rec := range s.ManagedPaths {
		if rec.Interface == iface {
			rows = append(rows, p+"|"+rec.Fingerprint)
		}
	}
	sort.Strings(rows)
	return util.SHA256Tagged([]byte(strings.Join(rows, "\n")))
}

func launch(real string, args []string) int {
	if os.Getenv("ATRAKTA_WRAP_SKIP_LAUNCH") == "1" {
		return 0
	}
	if real == "" {
		fmt.Fprintln(os.Stderr, "wrapper error: real executable not found")
		return 127
	}
	cmd := exec.Command(real, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	fmt.Fprintf(os.Stderr, "wrapper launch failed: %v\n", err)
	return 1
}
