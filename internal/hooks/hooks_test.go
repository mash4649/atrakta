package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAndRemoveSourceLine(t *testing.T) {
	rc := filepath.Join(t.TempDir(), ".zshrc")
	script := "/tmp/fake-hook.sh"
	if err := ensureSourceLine(rc, script); err != nil {
		t.Fatalf("ensure line failed: %v", err)
	}
	if err := ensureSourceLine(rc, script); err != nil {
		t.Fatalf("ensure line idempotent failed: %v", err)
	}
	b, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read rc failed: %v", err)
	}
	text := string(b)
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected single source line, got: %q", text)
	}
	if err := removeSourceLine(rc, script); err != nil {
		t.Fatalf("remove line failed: %v", err)
	}
	b2, _ := os.ReadFile(rc)
	if strings.Contains(string(b2), script) {
		t.Fatalf("expected source line removed")
	}
}

func TestBuildHookScriptIncludesPipelineStages(t *testing.T) {
	script := buildHookScript("/tmp/atrakta")
	for _, needle := range []string{
		"_atrakta_run_stage pre_start",
		"_atrakta_run_stage post_start",
		"_atrakta_run_stage on_error",
		"ATRAKTA_HOOK_DISABLE_STAGES",
		"ATRAKTA_HOOK_CONTINUE_ON_ERROR",
		"ATRAKTA_TRIGGER_SOURCE=hook",
		"</dev/null",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected hook script to include %q", needle)
		}
	}
}
