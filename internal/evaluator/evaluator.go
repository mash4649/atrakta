package evaluator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const (
	SchemaVersionAcceptanceResult = "acceptance_result.v1"
	StatusPassed                  = "passed"
	StatusFailed                  = "failed"
	StatusSkipped                 = "skipped"
	SkippedMessagePlaywright      = "VALIDATION_SKIPPED: playwright not found"
)

// Criterion describes one acceptance check.
type Criterion struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	CheckType   string `json:"check_type"`
	Threshold   any    `json:"threshold,omitempty"`
}

// Result captures the evaluation outcome and persisted artifact metadata.
type Result struct {
	SchemaVersion    string      `json:"schema_version"`
	ProjectRoot      string      `json:"project_root"`
	ArtifactPath     string      `json:"artifact_path"`
	Status           string      `json:"status"`
	ExitCode         int         `json:"exit_code,omitempty"`
	Message          string      `json:"message,omitempty"`
	Criteria         []Criterion `json:"criteria,omitempty"`
	PlaywrightBinary string      `json:"playwright_binary,omitempty"`
	Written          []string    `json:"written,omitempty"`
}

// Evaluator defines the acceptance-evaluation contract.
type Evaluator interface {
	Evaluate(artifactPath string, acceptanceCriteria []Criterion) (Result, error)
}

// Runner executes acceptance evaluation via Playwright.
type Runner struct {
	PlaywrightBinary string
	CommandRunner    CommandRunner
}

// CommandRunner executes a subprocess and returns captured output and exit code.
type CommandRunner func(ctx context.Context, name string, args []string, env []string) (stdout string, stderr string, exitCode int, err error)

// NewRunner returns a Runner with default Playwright invocation behavior.
func NewRunner() Runner {
	return Runner{PlaywrightBinary: "playwright"}
}

// Evaluate runs acceptance evaluation with the default runner.
func Evaluate(artifactPath string, acceptanceCriteria []Criterion) (Result, error) {
	return NewRunner().Evaluate(artifactPath, acceptanceCriteria)
}

// Evaluate executes Playwright against the provided artifact path and persists the result.
func (r Runner) Evaluate(artifactPath string, acceptanceCriteria []Criterion) (Result, error) {
	projectRoot, err := onboarding.DetectProjectRoot(filepath.Dir(artifactPath))
	if err != nil {
		return Result{}, err
	}

	result := Result{
		SchemaVersion:    SchemaVersionAcceptanceResult,
		ProjectRoot:      projectRoot,
		ArtifactPath:     artifactPath,
		Criteria:         append([]Criterion(nil), acceptanceCriteria...),
		PlaywrightBinary: r.playwrightBinary(),
	}

	if err := os.MkdirAll(filepath.Dir(acceptanceResultPath(projectRoot)), 0o755); err != nil {
		return Result{}, err
	}

	if _, err := exec.LookPath(result.PlaywrightBinary); err != nil {
		result.Status = StatusSkipped
		result.Message = SkippedMessagePlaywright
		if err := writeResult(projectRoot, result); err != nil {
			return Result{}, err
		}
		result.Written = []string{acceptanceResultPath(projectRoot)}
		return result, nil
	}

	stdout, stderr, exitCode, runErr := r.runPlaywright(projectRoot, artifactPath, acceptanceCriteria)
	if runErr != nil && exitCode == 0 {
		return Result{}, runErr
	}

	result.ExitCode = exitCode
	result.Message = strings.TrimSpace(strings.Join(filterNonEmpty([]string{stdout, stderr}), "\n"))
	if exitCode == 0 {
		result.Status = StatusPassed
	} else {
		result.Status = StatusFailed
		if result.Message == "" {
			result.Message = fmt.Sprintf("playwright exited with code %d", exitCode)
		}
	}
	if errors.Is(runErr, exec.ErrNotFound) {
		result.Status = StatusSkipped
		result.Message = SkippedMessagePlaywright
	}

	if err := writeResult(projectRoot, result); err != nil {
		return Result{}, err
	}
	result.Written = []string{acceptanceResultPath(projectRoot)}
	return result, nil
}

func (r Runner) playwrightBinary() string {
	if strings.TrimSpace(r.PlaywrightBinary) != "" {
		return r.PlaywrightBinary
	}
	return "playwright"
}

func (r Runner) runPlaywright(projectRoot, artifactPath string, acceptanceCriteria []Criterion) (string, string, int, error) {
	runner := r.CommandRunner
	if runner == nil {
		runner = defaultCommandRunner
	}

	criteriaJSON, err := json.Marshal(acceptanceCriteria)
	if err != nil {
		return "", "", 1, err
	}

	env := append(os.Environ(),
		"ATRAKTA_PROJECT_ROOT="+projectRoot,
		"ATRAKTA_ACCEPTANCE_CRITERIA_JSON="+string(criteriaJSON),
	)

	ctx := context.Background()
	return runner(ctx, r.playwrightBinary(), []string{"test", artifactPath}, env)
}

func acceptanceResultPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "state", "acceptance_result.json")
}

func writeResult(projectRoot string, result Result) error {
	path := acceptanceResultPath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func defaultCommandRunner(ctx context.Context, name string, args []string, env []string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return stdout.String(), stderr.String(), 1, err
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}

func filterNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
