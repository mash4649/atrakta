package progress

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"atrakta/internal/util"
)

type Progress struct {
	ActiveFeature     *string  `json:"active_feature"`
	CompletedFeatures []string `json:"completed_features"`
	LastCommitHash    *string  `json:"last_commit_hash"`
	UpdatedAt         string   `json:"updated_at"`
}

func path(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "progress.json")
}

func Empty() Progress {
	return Progress{ActiveFeature: nil, CompletedFeatures: []string{}, LastCommitHash: nil, UpdatedAt: util.NowUTC()}
}

func LoadOrInit(repoRoot string) (Progress, bool, error) {
	p := path(repoRoot)
	b, err := os.ReadFile(p)
	if err == nil {
		var pr Progress
		if err := json.Unmarshal(b, &pr); err != nil {
			return Progress{}, true, fmt.Errorf("parse progress.json: %w", err)
		}
		if pr.CompletedFeatures == nil {
			pr.CompletedFeatures = []string{}
		}
		if pr.UpdatedAt == "" {
			pr.UpdatedAt = util.NowUTC()
		}
		return pr, true, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Progress{}, false, fmt.Errorf("read progress.json: %w", err)
	}
	pr := Empty()
	if err := Save(repoRoot, pr); err != nil {
		return Progress{}, false, err
	}
	return pr, false, nil
}

func Save(repoRoot string, pgr Progress) error {
	pgr.UpdatedAt = util.NowUTC()
	path := path(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir .atrakta: %w", err)
	}
	b, err := json.MarshalIndent(pgr, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal progress.json: %w", err)
	}
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "progress.json.lock")
	if err := util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(path, append(b, '\n'), 0o644)
	}); err != nil {
		return fmt.Errorf("write progress.json: %w", err)
	}
	return nil
}

func ContainsFeature(list []string, feature string) bool {
	for _, f := range list {
		if f == feature {
			return true
		}
	}
	return false
}
