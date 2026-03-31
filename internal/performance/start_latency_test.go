package performance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareBenchmarkFixture(t *testing.T) {
	root, err := prepareBenchmarkFixture("generic-cli", "small")
	if err != nil {
		t.Fatalf("prepare benchmark fixture: %v", err)
	}
	defer os.RemoveAll(root)

	required := []string{
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, ".atrakta", "contract.json"),
		filepath.Join(root, ".atrakta", "canonical", "policies", "registry", "index.json"),
		filepath.Join(root, ".atrakta", "runtime", "start-fast.v1.json"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing fixture path %s: %v", path, err)
		}
	}
}

func TestPrepareBenchmarkFixtureMonorepo(t *testing.T) {
	root, err := prepareBenchmarkFixture("generic-cli", "monorepo")
	if err != nil {
		t.Fatalf("prepare monorepo fixture: %v", err)
	}
	defer os.RemoveAll(root)

	fileCount := 0
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".txt") {
			fileCount++
		}
		return nil
	}); err != nil {
		t.Fatalf("walk monorepo fixture: %v", err)
	}
	if fileCount < 1000 {
		t.Fatalf("file count=%d want >= 1000", fileCount)
	}
}

func TestMeasureStartLatency(t *testing.T) {
	report, err := MeasureStartLatency(Input{
		Iterations:  2,
		InterfaceID: "generic-cli",
	}, func(args []string) (Result, error) {
		return Result{ExitCode: 0, FastPath: true}, nil
	})
	if err != nil {
		t.Fatalf("measure start latency: %v", err)
	}
	if report.SchemaVersion != SchemaVersionStartLatency {
		t.Fatalf("schema version=%q", report.SchemaVersion)
	}
	if report.Iterations != 2 {
		t.Fatalf("iterations=%d", report.Iterations)
	}
	if report.FastPathHits != 2 {
		t.Fatalf("fast path hits=%d", report.FastPathHits)
	}
	if len(report.Samples) != 2 {
		t.Fatalf("sample count=%d", len(report.Samples))
	}
	if report.AverageMS < 0 || report.MinMS < 0 || report.MaxMS < 0 || report.MedianMS < 0 {
		t.Fatalf("unexpected negative stats: %+v", report)
	}
}

func TestMeasureStartLatencyMonorepoScenario(t *testing.T) {
	report, err := MeasureStartLatency(Input{
		Iterations:       1,
		InterfaceID:      "generic-cli",
		WorkspaceProfile: "monorepo",
	}, func(args []string) (Result, error) {
		return Result{ExitCode: 0, FastPath: true}, nil
	})
	if err != nil {
		t.Fatalf("measure monorepo latency: %v", err)
	}
	if report.Scenario != "start_large_repo" {
		t.Fatalf("scenario=%q", report.Scenario)
	}
	if report.ProjectRoot != "synthetic:start-large-repo" {
		t.Fatalf("project root=%q", report.ProjectRoot)
	}
}
