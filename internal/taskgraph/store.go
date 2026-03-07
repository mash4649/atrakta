package taskgraph

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"atrakta/internal/util"
)

func Path(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "task-graph.json")
}

func lockPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", ".locks", "task-graph.json.lock")
}

func Save(repoRoot string, g Graph) error {
	if g.V == 0 {
		g.V = 1
	}
	if g.Digest == "" {
		g.Digest = digest(g)
	}
	if g.GraphID == "" {
		g.GraphID = g.Digest
	}
	path := Path(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir .atrakta: %w", err)
	}
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task graph: %w", err)
	}
	b = append(b, '\n')
	return util.WithFileLock(lockPath(repoRoot), util.DefaultFileLockOptions(), func() error {
		if err := util.AtomicWriteFile(path, b, 0o644); err != nil {
			return fmt.Errorf("write task graph: %w", err)
		}
		return nil
	})
}

func Load(repoRoot string) (Graph, bool, error) {
	b, err := os.ReadFile(Path(repoRoot))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Graph{}, false, nil
		}
		return Graph{}, false, fmt.Errorf("read task graph: %w", err)
	}
	var g Graph
	if err := json.Unmarshal(b, &g); err != nil {
		return Graph{}, true, fmt.Errorf("parse task graph: %w", err)
	}
	if g.V != 1 {
		return Graph{}, true, fmt.Errorf("task graph v must be 1")
	}
	if err := Validate(g); err != nil {
		return Graph{}, true, err
	}
	return g, true, nil
}
