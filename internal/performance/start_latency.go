package performance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mash4649/atrakta/v0/internal/startfast"
)

const SchemaVersionStartLatency = "start_latency_benchmark.v1"

// Sample captures one start invocation measurement.
type Sample struct {
	Iteration  int    `json:"iteration"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	FastPath   bool   `json:"fast_path"`
	ResultPath string `json:"result_path,omitempty"`
}

// Report captures a start latency benchmark run.
type Report struct {
	SchemaVersion string   `json:"schema_version"`
	Scenario      string   `json:"scenario"`
	Command       string   `json:"command"`
	InterfaceID   string   `json:"interface_id"`
	ProjectRoot   string   `json:"project_root"`
	Iterations    int      `json:"iterations"`
	Samples       []Sample `json:"samples"`
	AverageMS     int64    `json:"average_ms"`
	MinMS         int64    `json:"min_ms"`
	MaxMS         int64    `json:"max_ms"`
	MedianMS      int64    `json:"median_ms"`
	FastPathHits  int      `json:"fast_path_hits"`
}

// Input describes the benchmark run configuration.
type Input struct {
	Iterations       int
	InterfaceID      string
	WorkspaceProfile string
}

// Result is the observed outcome for one benchmark sample.
type Result struct {
	ExitCode   int
	FastPath   bool
	ResultPath string
}

// Runner executes one benchmarked start command invocation.
type Runner func(args []string) (Result, error)

// MeasureStartLatency benchmarks the start command against synthetic fast-path fixtures.
func MeasureStartLatency(input Input, runner Runner) (Report, error) {
	if runner == nil {
		return Report{}, fmt.Errorf("benchmark runner required")
	}
	iterations := input.Iterations
	if iterations <= 0 {
		iterations = 1
	}
	interfaceID := strings.TrimSpace(input.InterfaceID)
	if interfaceID == "" {
		interfaceID = "generic-cli"
	}
	workspaceProfile := normalizeWorkspaceProfile(input.WorkspaceProfile)

	samples := make([]Sample, 0, iterations)
	fastPathHits := 0
	projectRoot := "synthetic:start-latency"
	scenario := "start_fast_path"
	if workspaceProfile == "monorepo" {
		projectRoot = "synthetic:start-large-repo"
		scenario = "start_large_repo"
	}

	for i := 0; i < iterations; i++ {
		root, err := prepareBenchmarkFixture(interfaceID, workspaceProfile)
		if err != nil {
			return Report{}, err
		}
		artifactDir := filepath.Join(root, "artifacts")
		start := time.Now()
		result, err := runner([]string{
			"--project-root", root,
			"--interface", interfaceID,
			"--non-interactive",
			"--json",
			"--artifact-dir", artifactDir,
		})
		elapsed := time.Since(start)
		if err != nil {
			_ = os.RemoveAll(root)
			return Report{}, err
		}
		if result.ExitCode != 0 {
			_ = os.RemoveAll(root)
			return Report{}, fmt.Errorf("start benchmark sample %d failed with exit code %d", i+1, result.ExitCode)
		}
		if result.FastPath {
			fastPathHits++
		}
		samples = append(samples, Sample{
			Iteration:  i + 1,
			DurationMS: elapsed.Milliseconds(),
			ExitCode:   result.ExitCode,
			FastPath:   result.FastPath,
			ResultPath: result.ResultPath,
		})
		_ = os.RemoveAll(root)
	}

	stats := summarizeDurations(samples)
	return Report{
		SchemaVersion: SchemaVersionStartLatency,
		Scenario:      scenario,
		Command:       "start",
		InterfaceID:   interfaceID,
		ProjectRoot:   projectRoot,
		Iterations:    iterations,
		Samples:       samples,
		AverageMS:     stats.average,
		MinMS:         stats.min,
		MaxMS:         stats.max,
		MedianMS:      stats.median,
		FastPathHits:  fastPathHits,
	}, nil
}

func prepareBenchmarkFixture(interfaceID, workspaceProfile string) (string, error) {
	root, err := os.MkdirTemp("", "atrakta-start-latency-*")
	if err != nil {
		return "", err
	}

	write := func(rel string, payload any) error {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		raw, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(abs, append(raw, '\n'), 0o644)
	}

	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# benchmark\n"), 0o644); err != nil {
		_ = os.RemoveAll(root)
		return "", err
	}
	if err := write(".atrakta/canonical/policies/registry/index.json", map[string]any{
		"entries": []any{},
	}); err != nil {
		_ = os.RemoveAll(root)
		return "", err
	}
	if workspaceProfile == "monorepo" {
		if err := populateMonorepoFixture(root); err != nil {
			_ = os.RemoveAll(root)
			return "", err
		}
	}
	if err := write(".atrakta/contract.json", map[string]any{
		"v": 1,
		"interfaces": map[string]any{
			"supported": []string{interfaceID},
			"fallback":  interfaceID,
		},
		"boundary": map[string]any{
			"managed_root": ".atrakta/",
		},
		"tools": map[string]any{
			"allow": []string{"create", "edit", "run"},
		},
		"security": map[string]any{
			"destructive":      "deny",
			"external_send":    "deny",
			"approval":         "explicit",
			"permission_model": "proposal_only",
		},
		"routing": map[string]any{
			"default": map[string]any{"worker": "general"},
		},
	}); err != nil {
		_ = os.RemoveAll(root)
		return "", err
	}

	key, err := startfast.ComputeKey(root, interfaceID, false)
	if err != nil {
		_ = os.RemoveAll(root)
		return "", err
	}
	if err := startfast.SaveSnapshot(root, startfast.Snapshot{
		Key:                 key.Key,
		ContractHash:        key.ContractHash,
		CanonicalPolicyHash: key.CanonicalPolicyHash,
		WorkspaceStamp:      key.WorkspaceStamp,
		InterfaceID:         interfaceID,
		ApplyRequested:      false,
	}); err != nil {
		_ = os.RemoveAll(root)
		return "", err
	}

	return root, nil
}

func populateMonorepoFixture(root string) error {
	writeFile := func(rel, content string) error {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		return os.WriteFile(abs, []byte(content), 0o644)
	}

	if err := writeFile("go.work", "go 1.22\nuse ./apps/web ./packages/core\n"); err != nil {
		return err
	}
	if err := writeFile("package.json", "{\n  \"name\": \"atrakta-monorepo\"\n}\n"); err != nil {
		return err
	}
	if err := writeFile("pyproject.toml", "[project]\nname = \"atrakta-monorepo\"\n"); err != nil {
		return err
	}
	if err := writeFile("Cargo.toml", "[package]\nname = \"atrakta-monorepo\"\nversion = \"0.1.0\"\n"); err != nil {
		return err
	}
	if err := writeFile(".github/workflows/ci.yml", "name: ci\n"); err != nil {
		return err
	}
	if err := writeFile("scripts/monorepo-check.sh", "#!/bin/sh\n"); err != nil {
		return err
	}
	if err := writeFile("AGENTS.md", "# benchmark\n"); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		for j := 0; j < 60; j++ {
			rel := filepath.Join("packages", fmt.Sprintf("pkg%02d", i), "src", fmt.Sprintf("file%03d.txt", j))
			if err := writeFile(rel, fmt.Sprintf("package %02d file %03d\n", i, j)); err != nil {
				return err
			}
		}
	}
	for i := 0; i < 5; i++ {
		for j := 0; j < 20; j++ {
			rel := filepath.Join("apps", fmt.Sprintf("app%02d", i), "src", fmt.Sprintf("asset%03d.txt", j))
			if err := writeFile(rel, fmt.Sprintf("app %02d asset %03d\n", i, j)); err != nil {
				return err
			}
		}
	}
	return nil
}

type durationStats struct {
	average int64
	min     int64
	max     int64
	median  int64
}

func summarizeDurations(samples []Sample) durationStats {
	if len(samples) == 0 {
		return durationStats{}
	}
	values := make([]int64, 0, len(samples))
	var sum int64
	min := samples[0].DurationMS
	max := samples[0].DurationMS
	for _, sample := range samples {
		values = append(values, sample.DurationMS)
		sum += sample.DurationMS
		if sample.DurationMS < min {
			min = sample.DurationMS
		}
		if sample.DurationMS > max {
			max = sample.DurationMS
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}
	return durationStats{
		average: sum / int64(len(values)),
		min:     min,
		max:     max,
		median:  median,
	}
}

func normalizeWorkspaceProfile(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "", "small", "fast", "synthetic":
		return "small"
	case "monorepo", "large", "large_repo":
		return "monorepo"
	default:
		return "small"
	}
}
