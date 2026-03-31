package evaluator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEvaluateSkipsWhenPlaywrightMissing(t *testing.T) {
	root := setupEvaluatorProject(t)
	artifact := filepath.Join(root, "dist", "app.html")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	if err := os.WriteFile(artifact, []byte("<html></html>\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	r := Runner{PlaywrightBinary: filepath.Join(root, "bin", "missing-playwright")}
	got, err := r.Evaluate(artifact, []Criterion{{ID: "render", CheckType: "file_exists"}})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got.Status != StatusSkipped {
		t.Fatalf("status=%q", got.Status)
	}
	if !strings.Contains(got.Message, SkippedMessagePlaywright) {
		t.Fatalf("message=%q", got.Message)
	}
	if got.Written == nil || len(got.Written) != 1 {
		t.Fatalf("written=%v", got.Written)
	}
	if _, err := os.Stat(filepath.Join(root, ".atrakta", "state", "acceptance_result.json")); err != nil {
		t.Fatalf("missing acceptance_result.json: %v", err)
	}
}

func TestEvaluateRunsPlaywrightAndWritesResult(t *testing.T) {
	root := setupEvaluatorProject(t)
	artifact := filepath.Join(root, "dist", "app.html")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	if err := os.WriteFile(artifact, []byte("<html></html>\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	ranMarker := filepath.Join(root, ".atrakta", "state", "playwright-ran.txt")
	script := writeShellScript(t, root, "playwright-success.sh", `#!/usr/bin/env sh
set -eu
printf '%s\n' "$ATRAKTA_ACCEPTANCE_CRITERIA_JSON" > "$ATRAKTA_PROJECT_ROOT/.atrakta/state/playwright-ran.txt"
exit 0
`)

	r := Runner{PlaywrightBinary: script}
	got, err := r.Evaluate(artifact, []Criterion{{ID: "render", CheckType: "file_exists", Description: "artifact exists"}})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got.Status != StatusPassed {
		t.Fatalf("status=%q", got.Status)
	}
	if got.ExitCode != 0 {
		t.Fatalf("exit_code=%d", got.ExitCode)
	}
	if _, err := os.Stat(ranMarker); err != nil {
		t.Fatalf("missing playwright marker: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".atrakta", "state", "acceptance_result.json"))
	if err != nil {
		t.Fatalf("read acceptance result: %v", err)
	}
	var out Result
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out.Status != StatusPassed || out.ArtifactPath != artifact {
		t.Fatalf("unexpected persisted result: %+v", out)
	}
}

func TestEvaluateMapsNonZeroExitToFailed(t *testing.T) {
	root := setupEvaluatorProject(t)
	artifact := filepath.Join(root, "dist", "app.html")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	if err := os.WriteFile(artifact, []byte("<html></html>\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	script := writeShellScript(t, root, "playwright-fail.sh", `#!/usr/bin/env sh
set -eu
exit 2
`)

	r := Runner{PlaywrightBinary: script}
	got, err := r.Evaluate(artifact, []Criterion{{ID: "render", CheckType: "file_exists"}})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got.Status != StatusFailed {
		t.Fatalf("status=%q", got.Status)
	}
	if got.ExitCode != 2 {
		t.Fatalf("exit_code=%d", got.ExitCode)
	}
}

func setupEvaluatorProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/evaluator-test\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("shell script test helper is unix-only")
	}
	return root
}

func writeShellScript(t *testing.T, root, name, content string) string {
	t.Helper()
	script := filepath.Join(root, name)
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return script
}
