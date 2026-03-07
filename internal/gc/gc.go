package gc

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"atrakta/internal/util"
)

const (
	stateVersion = 1
)

type Request struct {
	RepoRoot string
	Scopes   map[string]bool
	Apply    bool
	Auto     bool
}

type Report struct {
	V          int      `json:"v"`
	StartedAt  string   `json:"started_at"`
	FinishedAt string   `json:"finished_at,omitempty"`
	Auto       bool     `json:"auto"`
	Apply      bool     `json:"apply"`
	Scopes     []string `json:"scopes"`

	Tmp TmpReport `json:"tmp"`
	// Events is proposal-only by design in v0.9.
	Events EventsReport `json:"events"`
}

type TmpReport struct {
	Checked            bool     `json:"checked"`
	SkippedByPolicy    bool     `json:"skipped_by_policy,omitempty"`
	Triggered          bool     `json:"triggered"`
	TotalBytes         int64    `json:"total_bytes"`
	ThresholdBytes     int64    `json:"threshold_bytes"`
	TargetBytes        int64    `json:"target_bytes"`
	DryRunDelete       []string `json:"dry_run_delete,omitempty"`
	DryRunDeleteBytes  int64    `json:"dry_run_delete_bytes,omitempty"`
	AppliedDelete      []string `json:"applied_delete,omitempty"`
	AppliedDeleteBytes int64    `json:"applied_delete_bytes,omitempty"`
	Reason             string   `json:"reason,omitempty"`
}

type EventsReport struct {
	Checked        bool     `json:"checked"`
	ProposalOnly   bool     `json:"proposal_only"`
	TotalBytes     int64    `json:"total_bytes"`
	ThresholdBytes int64    `json:"threshold_bytes"`
	Proposals      []string `json:"proposals,omitempty"`
	Reason         string   `json:"reason,omitempty"`
}

type Config struct {
	TmpMaxBytes            int64
	TmpTargetRatioPercent  int
	TmpRetentionDays       int
	EventsProposalBytes    int64
	AutoMinIntervalMinutes int
}

type QuickStatus struct {
	TmpExists        bool
	TmpTotalBytes    int64
	TmpThreshold     int64
	TmpOverThreshold bool

	EventsExists        bool
	EventsTotalBytes    int64
	EventsThreshold     int64
	EventsOverThreshold bool
}

type runtimeState struct {
	V          int    `json:"v"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	LastAutoAt string `json:"last_auto_at,omitempty"`
}

type tmpEntry struct {
	path    string
	size    int64
	modTime time.Time
}

func DefaultConfig() Config {
	return Config{
		TmpMaxBytes:            parseInt64Env("ATRAKTA_GC_TMP_MAX_BYTES", 2*1024*1024*1024),
		TmpTargetRatioPercent:  parseIntEnv("ATRAKTA_GC_TMP_TARGET_RATIO", 70),
		TmpRetentionDays:       parseIntEnv("ATRAKTA_GC_TMP_RETENTION_DAYS", 7),
		EventsProposalBytes:    parseInt64Env("ATRAKTA_GC_EVENTS_PROPOSAL_BYTES", 200*1024*1024),
		AutoMinIntervalMinutes: parseIntEnv("ATRAKTA_GC_AUTO_MIN_INTERVAL_MIN", 60),
	}
}

func Run(req Request, cfg Config) (Report, error) {
	now := time.Now().UTC()
	report := Report{
		V:         stateVersion,
		StartedAt: now.Format(time.RFC3339Nano),
		Auto:      req.Auto,
		Apply:     req.Apply,
		Scopes:    normalizeScopes(req.Scopes),
	}
	if req.RepoRoot == "" {
		return report, fmt.Errorf("repo root required")
	}
	if !scopeEnabled(req.Scopes, "tmp") && !scopeEnabled(req.Scopes, "events") {
		return report, fmt.Errorf("at least one scope required")
	}
	if req.Auto {
		st, _ := loadRuntimeState(req.RepoRoot)
		if shouldSkipAutoByInterval(st, cfg, now) {
			report.Tmp.Checked = false
			report.Tmp.SkippedByPolicy = true
			report.Tmp.Reason = "auto interval guard"
			report.Events.Checked = false
			report.Events.ProposalOnly = true
			report.Events.Reason = "auto interval guard"
			report.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
			_ = appendGCLog(req.RepoRoot, report)
			return report, nil
		}
	}

	if scopeEnabled(req.Scopes, "tmp") {
		tmpRep, err := runTmp(req.RepoRoot, req.Apply, req.Auto, cfg)
		if err != nil {
			return report, err
		}
		report.Tmp = tmpRep
	}
	if scopeEnabled(req.Scopes, "events") {
		evRep, err := runEvents(req.RepoRoot, cfg)
		if err != nil {
			return report, err
		}
		report.Events = evRep
	}

	report.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := appendGCLog(req.RepoRoot, report); err != nil {
		return report, err
	}
	if req.Auto {
		st, _ := loadRuntimeState(req.RepoRoot)
		st.V = stateVersion
		st.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		st.LastAutoAt = st.UpdatedAt
		_ = saveRuntimeState(req.RepoRoot, st)
	}
	return report, nil
}

func SpawnAuto(selfExe, repoRoot string) error {
	cmd := exec.Command(selfExe, "gc", "--auto", "--apply", "--scope", "tmp,events")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "ATRAKTA_GC_CHILD=1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd.Start()
}

func ShouldRunAuto(repoRoot string, cfg Config) bool {
	st, err := loadRuntimeState(repoRoot)
	if err != nil {
		return true
	}
	return !shouldSkipAutoByInterval(st, cfg, time.Now().UTC())
}

func Check(repoRoot string, cfg Config) (QuickStatus, error) {
	out := QuickStatus{
		TmpThreshold:    cfg.TmpMaxBytes,
		EventsThreshold: cfg.EventsProposalBytes,
	}
	entries, total, err := scanTmpEntries(filepath.Join(repoRoot, ".tmp"))
	if err != nil {
		return out, err
	}
	out.TmpExists = len(entries) > 0
	out.TmpTotalBytes = total
	out.TmpOverThreshold = total > cfg.TmpMaxBytes

	eventsPath := filepath.Join(repoRoot, ".atrakta", "events.jsonl")
	fi, err := os.Stat(eventsPath)
	if err == nil {
		out.EventsExists = true
		out.EventsTotalBytes = fi.Size()
		out.EventsOverThreshold = out.EventsTotalBytes > cfg.EventsProposalBytes
	} else if err != nil && !os.IsNotExist(err) {
		return out, err
	}
	return out, nil
}

func runTmp(repoRoot string, apply, auto bool, cfg Config) (TmpReport, error) {
	rep := TmpReport{
		Checked:        true,
		ThresholdBytes: cfg.TmpMaxBytes,
		TargetBytes:    cfg.TmpMaxBytes * int64(max(cfg.TmpTargetRatioPercent, 10)) / 100,
	}
	tmpRoot := filepath.Join(repoRoot, ".tmp")
	entries, total, err := scanTmpEntries(tmpRoot)
	if err != nil {
		return rep, err
	}
	rep.TotalBytes = total
	rep.Triggered = total > cfg.TmpMaxBytes
	if auto && !rep.Triggered {
		rep.SkippedByPolicy = true
		rep.Reason = "below threshold"
		return rep, nil
	}
	if len(entries) == 0 {
		rep.Reason = "no entries"
		return rep, nil
	}
	candidates := pickTmpCandidates(entries, total, rep.TargetBytes, cfg.TmpRetentionDays)
	rep.DryRunDelete = make([]string, 0, len(candidates))
	for _, c := range candidates {
		rep.DryRunDelete = append(rep.DryRunDelete, toRel(repoRoot, c.path))
		rep.DryRunDeleteBytes += c.size
	}
	if !apply {
		return rep, nil
	}
	rep.AppliedDelete = []string{}
	for _, c := range candidates {
		if err := os.RemoveAll(c.path); err != nil {
			continue
		}
		rep.AppliedDelete = append(rep.AppliedDelete, toRel(repoRoot, c.path))
		rep.AppliedDeleteBytes += c.size
	}
	return rep, nil
}

func runEvents(repoRoot string, cfg Config) (EventsReport, error) {
	rep := EventsReport{
		Checked:        true,
		ProposalOnly:   true,
		ThresholdBytes: cfg.EventsProposalBytes,
	}
	p := filepath.Join(repoRoot, ".atrakta", "events.jsonl")
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			rep.Reason = "events file not found"
			return rep, nil
		}
		return rep, err
	}
	rep.TotalBytes = fi.Size()
	if rep.TotalBytes <= cfg.EventsProposalBytes {
		rep.Reason = "below threshold"
		return rep, nil
	}
	rep.Proposals = append(rep.Proposals,
		"events log exceeds threshold; proposal-only in v0.9",
		"manual archive example: cp .atrakta/events.jsonl .atrakta/events.jsonl.bak && gzip -9 .atrakta/events.jsonl.bak",
		"after archive, run: atrakta doctor",
	)
	return rep, nil
}

func scanTmpEntries(tmpRoot string) ([]tmpEntry, int64, error) {
	fi, err := os.Stat(tmpRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	if !fi.IsDir() {
		return nil, 0, nil
	}
	children, err := os.ReadDir(tmpRoot)
	if err != nil {
		return nil, 0, err
	}
	out := make([]tmpEntry, 0, len(children))
	var total int64
	for _, child := range children {
		p := filepath.Join(tmpRoot, child.Name())
		size, mt, err := dirSizeAndModTime(p)
		if err != nil {
			continue
		}
		out = append(out, tmpEntry{path: p, size: size, modTime: mt})
		total += size
	}
	return out, total, nil
}

func dirSizeAndModTime(path string) (int64, time.Time, error) {
	var total int64
	var newest time.Time
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if fi.Mode().IsRegular() {
			total += fi.Size()
		}
		if fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
		return nil
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	return total, newest, nil
}

func pickTmpCandidates(entries []tmpEntry, total, target int64, retentionDays int) []tmpEntry {
	if len(entries) == 0 {
		return nil
	}
	now := time.Now().UTC()
	oldCutoff := now.Add(-time.Duration(max(retentionDays, 1)) * 24 * time.Hour)

	byOldest := append([]tmpEntry{}, entries...)
	sort.SliceStable(byOldest, func(i, j int) bool {
		if byOldest[i].modTime.Equal(byOldest[j].modTime) {
			return byOldest[i].path < byOldest[j].path
		}
		return byOldest[i].modTime.Before(byOldest[j].modTime)
	})
	candidates := []tmpEntry{}
	seen := map[string]struct{}{}
	remaining := total

	for _, e := range byOldest {
		if e.modTime.Before(oldCutoff) {
			candidates = append(candidates, e)
			seen[e.path] = struct{}{}
			remaining -= e.size
		}
	}
	for _, e := range byOldest {
		if remaining <= target {
			break
		}
		if _, ok := seen[e.path]; ok {
			continue
		}
		candidates = append(candidates, e)
		seen[e.path] = struct{}{}
		remaining -= e.size
	}
	return candidates
}

func normalizeScopes(sc map[string]bool) []string {
	if len(sc) == 0 {
		return []string{"tmp", "events"}
	}
	out := []string{}
	for _, k := range []string{"tmp", "events"} {
		if scopeEnabled(sc, k) {
			out = append(out, k)
		}
	}
	return out
}

func scopeEnabled(sc map[string]bool, key string) bool {
	if len(sc) == 0 {
		return true
	}
	return sc[key]
}

func runtimeDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "runtime")
}

func runtimeStatePath(repoRoot string) string {
	return filepath.Join(runtimeDir(repoRoot), "gc-state.v1.json")
}

func runtimeLogPath(repoRoot string) string {
	return filepath.Join(runtimeDir(repoRoot), "gc-log.jsonl")
}

func loadRuntimeState(repoRoot string) (runtimeState, error) {
	p := runtimeStatePath(repoRoot)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return runtimeState{V: stateVersion}, nil
		}
		return runtimeState{}, err
	}
	var st runtimeState
	if err := json.Unmarshal(b, &st); err != nil {
		return runtimeState{}, err
	}
	return st, nil
}

func saveRuntimeState(repoRoot string, st runtimeState) error {
	st.V = stateVersion
	if st.UpdatedAt == "" {
		st.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	p := runtimeStatePath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	lockPath := filepath.Join(runtimeDir(repoRoot), ".locks", "gc-state.v1.lock")
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(p, b, 0o644)
	})
}

func appendGCLog(repoRoot string, report Report) error {
	p := runtimeLogPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	lockPath := filepath.Join(runtimeDir(repoRoot), ".locks", "gc-log.v1.lock")
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		fd, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer fd.Close()
		b, err := json.Marshal(report)
		if err != nil {
			return err
		}
		_, err = fd.Write(append(b, '\n'))
		return err
	})
}

func shouldSkipAutoByInterval(st runtimeState, cfg Config, now time.Time) bool {
	if cfg.AutoMinIntervalMinutes <= 0 {
		return false
	}
	last := strings.TrimSpace(st.LastAutoAt)
	if last == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, last)
	if err != nil {
		return false
	}
	return now.Sub(t) < time.Duration(cfg.AutoMinIntervalMinutes)*time.Minute
}

func parseIntEnv(key string, fallback int) int {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func parseInt64Env(key string, fallback int64) int64 {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return fallback
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func toRel(repoRoot, abs string) string {
	rel, err := filepath.Rel(repoRoot, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(rel)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
