package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

type gcResult struct {
	ProjectRoot  string   `json:"project_root"`
	Scope        string   `json:"scope"`
	Apply        bool     `json:"apply"`
	Candidates   []string `json:"candidates"`
	Removed      []string `json:"removed"`
	TotalBytes   int64    `json:"total_bytes"`
	RemovedBytes int64    `json:"removed_bytes"`
	Message      string   `json:"message"`
}

func runGC(args []string) error {
	fs := flag.NewFlagSet("gc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var scope string
	var apply bool
	var jsonOut bool
	var artifactDir string
	var retentionDays int

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.StringVar(&scope, "scope", "tmp", "cleanup scope: tmp|events")
	fs.BoolVar(&apply, "apply", false, "perform deletion")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	fs.IntVar(&retentionDays, "retention-days", 30, "retention period for events cleanup in days")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if retentionDays < 1 {
		return fmt.Errorf("--retention-days must be >= 1")
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}

	scope = strings.TrimSpace(strings.ToLower(scope))
	auditRoot := filepath.Join(root, ".atrakta", "audit")
	if err := verifyGCAuditIntegrity(auditRoot); err != nil {
		return err
	}

	result := gcResult{
		ProjectRoot: root,
		Scope:       scope,
		Apply:       apply,
	}

	switch scope {
	case "tmp":
		candidates, totalBytes, err := gcTmpCandidates(root)
		if err != nil {
			return err
		}
		result.Candidates = toSlashAll(candidates)
		result.TotalBytes = totalBytes
		if apply {
			runtimeRoot := filepath.Join(root, ".atrakta", "runtime")
			removed, removedBytes, err := gcApplyTmp(runtimeRoot, candidates)
			if err != nil {
				return err
			}
			result.Removed = toSlashAll(removed)
			result.RemovedBytes = removedBytes
		}
	case "events":
		plan, err := buildGCRunEventsPlan(root, retentionDays)
		if err != nil {
			return err
		}
		if plan.totalBytes > 0 {
			result.TotalBytes = plan.totalBytes
		}
		if plan.removed > 0 {
			result.Candidates = []string{filepath.ToSlash(plan.path)}
		}
		if apply && plan.removed > 0 {
			if err := gcRewriteRunEvents(root, plan); err != nil {
				return err
			}
			result.Removed = []string{filepath.ToSlash(plan.path)}
			result.RemovedBytes = plan.removedBytes
		}
	default:
		return fmt.Errorf("unsupported --scope %q (expected tmp or events)", scope)
	}

	if err := appendOperationalRunEvent(root, runEventGCRun, "", map[string]any{
		"command":         "gc",
		"scope":           result.Scope,
		"apply":           result.Apply,
		"candidate_count": len(result.Candidates),
		"removed_count":   len(result.Removed),
		"total_bytes":     result.TotalBytes,
		"removed_bytes":   result.RemovedBytes,
		"retention_days":  retentionDays,
	}); err != nil {
		return err
	}

	switch {
	case apply:
		result.Message = fmt.Sprintf("gc applied: removed %d paths", len(result.Removed))
	case len(result.Candidates) > 0:
		result.Message = fmt.Sprintf("gc dry-run: %d candidate paths", len(result.Candidates))
	default:
		result.Message = "gc dry-run: no candidate paths"
	}

	return emitGCResult(result, jsonOut, artifactDir)
}

func verifyGCAuditIntegrity(auditRoot string) error {
	if err := audit.VerifyIntegrity(auditRoot, audit.LevelA2); err != nil {
		return fmt.Errorf("gc audit integrity verify: %w", err)
	}
	if err := audit.VerifyRunEventsIntegrity(auditRoot, audit.LevelA2); err != nil {
		return fmt.Errorf("gc run-events integrity verify: %w", err)
	}
	return nil
}

func gcTmpCandidates(projectRoot string) ([]string, int64, error) {
	runtimeRoot := filepath.Join(projectRoot, ".atrakta", "runtime")
	entries, err := os.ReadDir(runtimeRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, 0, nil
		}
		return nil, 0, err
	}

	candidates := make([]string, 0)
	var totalBytes int64
	for _, entry := range entries {
		path := filepath.Join(runtimeRoot, entry.Name())
		if entry.IsDir() {
			if err := filepath.WalkDir(path, func(childPath string, d os.DirEntry, walkErr error) error {
				if walkErr != nil || d == nil || d.IsDir() {
					return walkErr
				}
				candidates = append(candidates, childPath)
				size, err := fileSizeOrZero(childPath)
				if err == nil {
					totalBytes += size
				}
				return nil
			}); err != nil {
				return nil, 0, err
			}
			continue
		}
		candidates = append(candidates, path)
		size, err := fileSizeOrZero(path)
		if err == nil {
			totalBytes += size
		}
	}
	return candidates, totalBytes, nil
}

func gcApplyTmp(runtimeRoot string, candidates []string) ([]string, int64, error) {
	removed := make([]string, 0, len(candidates))
	var removedBytes int64
	for _, path := range candidates {
		size, _ := fileSizeOrZero(path)
		if err := os.RemoveAll(path); err != nil {
			return nil, 0, fmt.Errorf("gc remove %s: %w", path, err)
		}
		removed = append(removed, path)
		removedBytes += size
	}
	pruneEmptyDirs(runtimeRoot)
	return removed, removedBytes, nil
}

func fileSizeOrZero(path string) (int64, error) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if !st.IsDir() {
		return st.Size(), nil
	}
	var total int64
	err = filepath.Walk(path, func(_ string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

type gcRunEventsState struct {
	path         string
	level        string
	totalBytes   int64
	removed      int
	removedBytes int64
	kept         []audit.RunEvent
}

func buildGCRunEventsPlan(projectRoot string, retentionDays int) (gcRunEventsState, error) {
	path := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return gcRunEventsState{path: path, level: audit.LevelA2}, nil
		}
		return gcRunEventsState{}, err
	}

	events, err := readGCRunEvents(path)
	if err != nil {
		return gcRunEventsState{}, err
	}
	level := audit.LevelA2
	if len(events) > 0 && strings.TrimSpace(events[0].Integrity) != "" {
		level = events[0].Integrity
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	kept := make([]audit.RunEvent, 0, len(events))
	var removedCount int
	var removedBytes int64
	for _, ev := range events {
		ts, err := time.Parse(time.RFC3339, ev.Timestamp)
		if err != nil {
			return gcRunEventsState{}, fmt.Errorf("parse run-event timestamp: %w", err)
		}
		if ts.Before(cutoff) {
			removedCount++
			removedBytes += estimatedRunEventLineBytes(ev)
			continue
		}
		kept = append(kept, ev)
	}

	return gcRunEventsState{
		path:         path,
		level:        level,
		totalBytes:   int64(len(raw)),
		removed:      removedCount,
		removedBytes: removedBytes,
		kept:         kept,
	}, nil
}

func gcRewriteRunEvents(projectRoot string, plan gcRunEventsState) error {
	path := plan.path
	level := normalizeAuditLevel(plan.level)
	if level == "" {
		level = audit.LevelA2
	}

	if len(plan.kept) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := writeGCRunEvents(path, plan.kept, level); err != nil {
			return err
		}
	}

	if level == audit.LevelA3 {
		cp := audit.Checkpoint{Level: level}
		if len(plan.kept) > 0 {
			last := plan.kept[len(plan.kept)-1]
			cp.LastSeq = last.Seq
			cp.LastHash = last.Hash
		}
		cpBytes, err := json.MarshalIndent(cp, "", "  ")
		if err != nil {
			return err
		}
		checkpointPath := filepath.Join(projectRoot, ".atrakta", "audit", "checkpoints", "run-head.json")
		if err := os.MkdirAll(filepath.Dir(checkpointPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(checkpointPath, append(cpBytes, '\n'), 0o644); err != nil {
			return err
		}
	}

	if err := verifyGCAuditIntegrity(filepath.Join(projectRoot, ".atrakta", "audit")); err != nil {
		return err
	}
	return nil
}

func readGCRunEvents(path string) ([]audit.RunEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []audit.RunEvent{}, nil
		}
		return nil, err
	}
	defer f.Close()

	out := make([]audit.RunEvent, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev audit.RunEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, fmt.Errorf("parse run-events entry: %w", err)
		}
		out = append(out, ev)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func writeGCRunEvents(path string, events []audit.RunEvent, level string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	prevHash := ""
	for i := range events {
		ev := events[i]
		ev.Seq = i + 1
		ev.Integrity = level
		if ev.Payload == nil {
			ev.Payload = map[string]any{}
		}
		payloadBytes, err := json.Marshal(ev.Payload)
		if err != nil {
			return err
		}
		if levelRank(level) >= levelRank(audit.LevelA1) {
			ev.PayloadHash = hashHex(payloadBytes)
		}
		if levelRank(level) >= levelRank(audit.LevelA2) {
			ev.PrevHash = prevHash
			ev.Hash = computeChainHash(ev.Seq, ev.EventType, ev.PayloadHash, ev.PrevHash)
			prevHash = ev.Hash
		} else {
			ev.PayloadHash = ""
			ev.PrevHash = ""
			ev.Hash = ""
		}
		line, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func estimatedRunEventLineBytes(ev audit.RunEvent) int64 {
	line, err := json.Marshal(ev)
	if err != nil {
		return 0
	}
	return int64(len(line) + 1)
}

func normalizeAuditLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case audit.LevelA0, audit.LevelA1, audit.LevelA2, audit.LevelA3:
		return strings.ToUpper(strings.TrimSpace(level))
	default:
		return ""
	}
}

func levelRank(level string) int {
	switch normalizeAuditLevel(level) {
	case audit.LevelA0:
		return 0
	case audit.LevelA1:
		return 1
	case audit.LevelA2:
		return 2
	case audit.LevelA3:
		return 3
	default:
		return -1
	}
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func computeChainHash(seq int, action, payloadHash, prevHash string) string {
	raw := fmt.Sprintf("%d|%s|%s|%s", seq, action, payloadHash, prevHash)
	return hashHex([]byte(raw))
}

func pruneEmptyDirs(root string) {
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || !d.IsDir() || path == root {
			return nil
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil || len(entries) > 0 {
			return nil
		}
		_ = os.Remove(path)
		return nil
	})
}

func emitGCResult(result gcResult, jsonOut bool, artifactDir string) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
	} else {
		fmt.Printf("project_root: %s\n", result.ProjectRoot)
		fmt.Printf("scope: %s\n", result.Scope)
		fmt.Printf("apply: %t\n", result.Apply)
		fmt.Printf("candidates: %d\n", len(result.Candidates))
		fmt.Printf("removed: %d\n", len(result.Removed))
		fmt.Printf("message: %s\n", result.Message)
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "gc.result.json", result); err != nil {
			return err
		}
	}
	return nil
}

func toSlashAll(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, filepath.ToSlash(v))
	}
	return out
}
