package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"atrakta/internal/util"
)

type RunCheckpoint struct {
	V          int    `json:"v"`
	UpdatedAt  string `json:"updated_at"`
	FeatureID  string `json:"feature_id,omitempty"`
	Interfaces string `json:"interfaces,omitempty"`
	SyncLevel  string `json:"sync_level,omitempty"`
	Stage      string `json:"stage"`
	Outcome    string `json:"outcome,omitempty"`
	Reason     string `json:"reason,omitempty"`

	DetectReason string `json:"detect_reason,omitempty"`
	PlanID       string `json:"plan_id,omitempty"`
	TaskGraphID  string `json:"task_graph_id,omitempty"`
	ApplyResult  string `json:"apply_result,omitempty"`
}

func SaveLatest(repoRoot string, cp RunCheckpoint) error {
	cp.V = 1
	cp.UpdatedAt = util.NowUTC()

	path := latestPath(repoRoot)
	lock := lockPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir run-checkpoints: %w", err)
	}
	b, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run checkpoint: %w", err)
	}
	b = append(b, '\n')
	if err := util.WithFileLock(lock, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(path, b, 0o644)
	}); err != nil {
		return fmt.Errorf("write run checkpoint: %w", err)
	}
	return nil
}

func LoadLatest(repoRoot string) (RunCheckpoint, error) {
	path := latestPath(repoRoot)
	b, err := os.ReadFile(path)
	if err != nil {
		return RunCheckpoint{}, err
	}
	var cp RunCheckpoint
	if err := json.Unmarshal(b, &cp); err != nil {
		return RunCheckpoint{}, fmt.Errorf("parse run checkpoint: %w", err)
	}
	if cp.V != 1 {
		return RunCheckpoint{}, fmt.Errorf("run checkpoint v must be 1")
	}
	if cp.Stage == "" {
		return RunCheckpoint{}, fmt.Errorf("run checkpoint stage required")
	}
	return cp, nil
}

func latestPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "run-checkpoints", "latest.json")
}

func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", ".locks", "run-checkpoints.latest.lock")
}
