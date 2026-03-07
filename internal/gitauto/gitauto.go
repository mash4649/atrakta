package gitauto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/util"
)

const (
	modeOff  = "off"
	modeAuto = "auto"
	modeOn   = "on"
)

type Snapshot struct {
	Available    bool   `json:"available"`
	Reason       string `json:"reason,omitempty"`
	Branch       string `json:"branch,omitempty"`
	Head         string `json:"head,omitempty"`
	Dirty        bool   `json:"dirty"`
	ChangedCount int    `json:"changed_count"`
}

type Checkpoint struct {
	GeneratedAt       string            `json:"generated_at"`
	Mode              string            `json:"mode"`
	Enabled           bool              `json:"enabled"`
	Reason            string            `json:"reason"`
	FeatureID         string            `json:"feature_id,omitempty"`
	PlanID            string            `json:"plan_id,omitempty"`
	Gate              model.GateResult  `json:"gate"`
	Pre               Snapshot          `json:"pre"`
	Post              Snapshot          `json:"post"`
	ManagedPaths      []string          `json:"managed_paths,omitempty"`
	ManagedPathCount  int               `json:"managed_path_count"`
	SuggestedCommit   string            `json:"suggested_commit,omitempty"`
	AdditionalSignals map[string]string `json:"additional_signals,omitempty"`
}

type Setup struct {
	Mode        string   `json:"mode"`
	Initialized bool     `json:"initialized"`
	Performed   bool     `json:"performed"`
	Reason      string   `json:"reason"`
	Suggested   []string `json:"suggested,omitempty"`
}

func ResolveMode(c contract.Contract) string {
	mode := modeAuto
	if c.Autonomy != nil && c.Autonomy.Git != nil {
		mode = normalizeMode(c.Autonomy.Git.Mode, mode)
	}
	mode = normalizeMode(os.Getenv("ATRAKTA_GIT_AUTOMATION"), mode)
	return mode
}

func EnsureSetup(repoRoot, mode string) (Setup, error) {
	m := normalizeMode(mode, modeAuto)
	report := Setup{Mode: m}
	if isGitRepo(repoRoot) {
		report.Initialized = true
		report.Reason = "already_initialized"
		return report, nil
	}
	if m == modeOff {
		report.Reason = "mode_off"
		report.Suggested = []string{
			"git init",
			"git add AGENTS.md .atrakta/contract.json",
			`git commit -m "chore: initialize atrakta"`,
		}
		_ = writeBootstrapGuide(repoRoot, report)
		return report, nil
	}
	if m == modeAuto && hasGitAncestor(repoRoot) {
		report.Reason = "inside_parent_git_repo"
		report.Suggested = []string{
			"Use parent git repository, skip nested git init.",
		}
		_ = writeBootstrapGuide(repoRoot, report)
		return report, nil
	}
	if _, err := exec.LookPath("git"); err != nil {
		report.Reason = "git_unavailable"
		report.Suggested = []string{
			"Install git or run with ATRAKTA_GIT_AUTOMATION=off.",
		}
		_ = writeBootstrapGuide(repoRoot, report)
		if m == modeOn {
			return report, fmt.Errorf("git executable not found in PATH")
		}
		return report, nil
	}
	if _, err := runGit(repoRoot, "init"); err != nil {
		report.Reason = "git_init_failed"
		report.Suggested = []string{
			"git init",
			"Check git executable and repository permissions.",
		}
		_ = writeBootstrapGuide(repoRoot, report)
		return report, fmt.Errorf("git init failed: %w", err)
	}
	if err := ensureGitignore(repoRoot); err != nil {
		return report, err
	}
	report.Initialized = true
	report.Performed = true
	report.Reason = "git_initialized"
	return report, nil
}

func Capture(repoRoot string) Snapshot {
	if !isGitRepo(repoRoot) {
		return Snapshot{Available: false, Reason: "not_git_repo"}
	}
	branch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Snapshot{Available: false, Reason: "git_rev_parse_failed"}
	}
	head, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return Snapshot{Available: false, Reason: "git_head_failed"}
	}
	status, err := runGit(repoRoot, "status", "--porcelain", "-uno")
	if err != nil {
		return Snapshot{Available: false, Reason: "git_status_failed"}
	}
	changed := 0
	if status != "" {
		changed = len(strings.Split(status, "\n"))
	}
	return Snapshot{
		Available:    true,
		Branch:       branch,
		Head:         head,
		Dirty:        changed > 0,
		ChangedCount: changed,
	}
}

func WriteCheckpoint(repoRoot, featureID string, mode string, pre Snapshot, post Snapshot, pl model.PlanResult, ap model.ApplyResult, gt model.GateResult) (Checkpoint, bool, error) {
	managed := managedPathsFromApply(ap)
	cp := Checkpoint{
		GeneratedAt:      util.NowUTC(),
		Mode:             normalizeMode(mode, modeAuto),
		FeatureID:        strings.TrimSpace(featureID),
		PlanID:           pl.ID,
		Gate:             gt,
		Pre:              pre,
		Post:             post,
		ManagedPaths:     managed,
		ManagedPathCount: len(managed),
	}
	cp.Enabled, cp.Reason = shouldEnable(cp.Mode, pre, post, cp.ManagedPathCount, gt)
	if !cp.Enabled {
		return cp, false, nil
	}
	cp.SuggestedCommit = buildCommitMessage(cp.FeatureID, cp.ManagedPathCount, gt)
	cp.AdditionalSignals = map[string]string{
		"automation": "non_destructive",
		"intent":     "checkpoint_and_recovery_hint",
	}
	outPath := filepath.Join(repoRoot, ".atrakta", "git", "checkpoint-latest.json")
	b, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return Checkpoint{}, false, fmt.Errorf("marshal git checkpoint: %w", err)
	}
	b = append(b, '\n')
	if err := util.AtomicWriteFile(outPath, b, 0o644); err != nil {
		return Checkpoint{}, false, fmt.Errorf("write git checkpoint: %w", err)
	}
	return cp, true, nil
}

func shouldEnable(mode string, pre Snapshot, post Snapshot, managedCount int, gt model.GateResult) (bool, string) {
	switch normalizeMode(mode, modeAuto) {
	case modeOff:
		return false, "mode_off"
	case modeOn:
		if !post.Available {
			return false, "git_unavailable"
		}
		return true, "mode_on"
	default:
		if !post.Available {
			return false, "git_unavailable"
		}
		if managedCount == 0 {
			return false, "no_managed_changes"
		}
		if gt.Safety == model.GateFail || gt.Quick == model.GateFail {
			return false, "gate_failed"
		}
		if pre.Head != "" && post.Head != "" && pre.Head != post.Head {
			return true, "head_changed_during_run"
		}
		if post.Dirty {
			return true, "workspace_dirty_after_apply"
		}
		return false, "auto_no_checkpoint_needed"
	}
}

func managedPathsFromApply(ap model.ApplyResult) []string {
	set := map[string]struct{}{}
	for _, op := range ap.Ops {
		if op.Path == "" {
			continue
		}
		if op.Status != "ok" && op.Status != "skipped" {
			continue
		}
		switch op.Op {
		case "adopt", "link", "copy", "write", "delete", "unlink":
			set[op.Path] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func buildCommitMessage(featureID string, managedCount int, gt model.GateResult) string {
	if strings.TrimSpace(featureID) == "" {
		featureID = "adhoc"
	}
	return fmt.Sprintf("atrakta: checkpoint %s (managed=%d, safety=%s, quick=%s)", featureID, managedCount, gt.Safety, gt.Quick)
}

func isGitRepo(repoRoot string) bool {
	fi, err := os.Stat(filepath.Join(repoRoot, ".git"))
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return true
	}
	// git worktree may use .git file
	return !fi.IsDir()
}

func runGit(repoRoot string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func normalizeMode(raw, fallback string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case modeOff:
		return modeOff
	case modeOn:
		return modeOn
	case modeAuto:
		return modeAuto
	default:
		return fallback
	}
}

func hasGitAncestor(repoRoot string) bool {
	cur := filepath.Clean(filepath.Dir(repoRoot))
	for {
		if cur == "" || cur == "." || cur == string(filepath.Separator) {
			return false
		}
		if exists(filepath.Join(cur, ".git")) {
			return true
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return false
		}
		cur = parent
	}
}

func ensureGitignore(repoRoot string) error {
	const marker = "# Atrakta managed"
	lines := []string{
		marker,
		"/.atrakta/git/checkpoint-latest.json",
		"/.atrakta/git/bootstrap.md",
	}
	path := filepath.Join(repoRoot, ".gitignore")
	existing := ""
	if b, err := os.ReadFile(path); err == nil {
		existing = string(b)
	}
	changed := false
	var builder strings.Builder
	if existing != "" {
		builder.WriteString(existing)
		if !strings.HasSuffix(existing, "\n") {
			builder.WriteString("\n")
		}
	}
	for _, line := range lines {
		if strings.Contains("\n"+existing+"\n", "\n"+line+"\n") {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
		changed = true
	}
	if !changed {
		return nil
	}
	return util.AtomicWriteFile(path, []byte(builder.String()), 0o644)
}

func writeBootstrapGuide(repoRoot string, report Setup) error {
	outPath := filepath.Join(repoRoot, ".atrakta", "git", "bootstrap.md")
	var b strings.Builder
	b.WriteString("# Atrakta Git Bootstrap\n\n")
	b.WriteString("status: ")
	b.WriteString(report.Reason)
	b.WriteString("\n\n")
	if len(report.Suggested) > 0 {
		b.WriteString("## Suggested Actions\n")
		for _, s := range report.Suggested {
			b.WriteString("- ")
			b.WriteString(s)
			b.WriteString("\n")
		}
	}
	return util.AtomicWriteFile(outPath, []byte(b.String()), 0o644)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
