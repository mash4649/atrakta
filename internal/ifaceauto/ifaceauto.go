package ifaceauto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"atrakta/internal/util"
)

type UsageStat struct {
	Count      int    `json:"count"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

type State struct {
	V             int                  `json:"v"`
	UpdatedAt     string               `json:"updated_at,omitempty"`
	LastTargetSet []string             `json:"last_target_set,omitempty"`
	LastSource    string               `json:"last_source,omitempty"`
	Usage         map[string]UsageStat `json:"usage,omitempty"`
}

func Empty() State {
	return State{
		V:     1,
		Usage: map[string]UsageStat{},
	}
}

func Load(repoRoot string) (State, error) {
	path := statePath(repoRoot)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Empty(), nil
		}
		return State{}, fmt.Errorf("read auto state: %w", err)
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return State{}, fmt.Errorf("parse auto state: %w", err)
	}
	if st.V != 1 {
		return State{}, fmt.Errorf("auto state v must be 1")
	}
	if st.Usage == nil {
		st.Usage = map[string]UsageStat{}
	}
	st.LastTargetSet = normalizeList(st.LastTargetSet)
	return st, nil
}

func Save(repoRoot string, st State) error {
	st.V = 1
	st.UpdatedAt = util.NowUTC()
	st.LastTargetSet = normalizeList(st.LastTargetSet)
	if st.Usage == nil {
		st.Usage = map[string]UsageStat{}
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal auto state: %w", err)
	}
	b = append(b, '\n')
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "auto-state.v1.lock")
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(statePath(repoRoot), b, 0o644)
	})
}

func Record(st State, targets []string, source string) State {
	out := st
	if out.V == 0 {
		out = Empty()
	}
	if out.Usage == nil {
		out.Usage = map[string]UsageStat{}
	}
	out.LastTargetSet = normalizeList(targets)
	out.LastSource = strings.TrimSpace(source)
	now := util.NowUTC()
	for _, id := range out.LastTargetSet {
		s := out.Usage[id]
		s.Count++
		s.LastUsedAt = now
		out.Usage[id] = s
	}
	return out
}

func LastSingleTarget(st State, allowed map[string]struct{}) (string, bool) {
	if len(st.LastTargetSet) != 1 {
		return "", false
	}
	id := st.LastTargetSet[0]
	if _, ok := allowed[id]; !ok {
		return "", false
	}
	return id, true
}

func SuggestStale(st State, active []string, olderThan time.Duration, now time.Time) []string {
	if olderThan <= 0 {
		return nil
	}
	activeSet := map[string]struct{}{}
	for _, id := range normalizeList(active) {
		activeSet[id] = struct{}{}
	}
	out := []string{}
	for id, stat := range st.Usage {
		if _, ok := activeSet[id]; ok {
			continue
		}
		ts := strings.TrimSpace(stat.LastUsedAt)
		if ts == "" {
			continue
		}
		tm, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			continue
		}
		if now.Sub(tm) >= olderThan {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func statePath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "runtime", "auto-state.v1.json")
}

func normalizeList(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		id := strings.TrimSpace(raw)
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
