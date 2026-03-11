package hooks

import (
	"os"
	"path/filepath"
	"reflect"
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
		"_atrakta_run_stage shell.on_cd",
		"_atrakta_run_stage shell.on_exec",
		"add-zsh-hook preexec _atrakta_preexec_hook",
		"ATRAKTA_HOOK_DISABLE_STAGES",
		"ATRAKTA_HOOK_CONTINUE_ON_ERROR",
		"ATRAKTA_TRIGGER_SOURCE=hook",
		"ATRAKTA_NONINTERACTIVE=1",
		"</dev/null",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("expected hook script to include %q", needle)
		}
	}
}

func TestNormalizeSurfaces(t *testing.T) {
	got, err := normalizeSurfaces(nil, []string{"shell.on_cd"})
	if err != nil {
		t.Fatalf("normalize default failed: %v", err)
	}
	want := []string{"shell.on_cd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected defaults: got=%v want=%v", got, want)
	}

	got, err = normalizeSurfaces([]string{"git.pre_push,shell.on_exec", "git.pre_push"}, nil)
	if err != nil {
		t.Fatalf("normalize split failed: %v", err)
	}
	want = []string{"git.pre_push", "shell.on_exec"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized list: got=%v want=%v", got, want)
	}

	if _, err := normalizeSurfaces([]string{"shell.unknown"}, nil); err == nil {
		t.Fatalf("expected unsupported surface error")
	}
}

func TestManagedBlockUpsertAndRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hook")
	initial := "#!/bin/sh\n" +
		"echo user-start\n"
	if err := os.WriteFile(path, []byte(initial), 0o755); err != nil {
		t.Fatalf("write initial hook: %v", err)
	}
	start := managedPrefix + " git.pre_commit START"
	end := managedSuffix + " git.pre_commit END"
	body := "echo atrakta-managed"

	if err := upsertManagedBlock(path, start, end, body); err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	if err := upsertManagedBlock(path, start, end, body); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after upsert: %v", err)
	}
	text := string(b)
	if strings.Count(text, start) != 1 || strings.Count(text, end) != 1 {
		t.Fatalf("managed block should be idempotent, content=%q", text)
	}
	if !strings.Contains(text, "echo user-start") {
		t.Fatalf("user content should remain")
	}

	if err := removeManagedBlock(path, start, end); err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after remove: %v", err)
	}
	text = string(b)
	if strings.Contains(text, start) || strings.Contains(text, end) {
		t.Fatalf("managed markers should be removed")
	}
	if !strings.Contains(text, "echo user-start") {
		t.Fatalf("user content should remain after remove")
	}
}

func TestInstallAndUninstallGitSurfacePreservesExistingHook(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repoRoot := t.TempDir()
	hookDir := filepath.Join(repoRoot, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatalf("mkdir hook dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "pre-commit")
	initial := "#!/bin/sh\n" +
		"echo user-hook\n"
	if err := os.WriteFile(hookPath, []byte(initial), 0o755); err != nil {
		t.Fatalf("write initial hook: %v", err)
	}

	if err := InstallForRepo(repoRoot, "/tmp/atrakta", []string{"git.pre_commit"}); err != nil {
		t.Fatalf("install git surface failed: %v", err)
	}
	if err := InstallForRepo(repoRoot, "/tmp/atrakta", []string{"git.pre_commit"}); err != nil {
		t.Fatalf("reinstall git surface failed: %v", err)
	}
	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read installed hook: %v", err)
	}
	text := string(b)
	start := managedPrefix + " git.pre_commit START"
	end := managedSuffix + " git.pre_commit END"
	if strings.Count(text, start) != 1 || strings.Count(text, end) != 1 {
		t.Fatalf("managed block should not duplicate, content=%q", text)
	}
	if !strings.Contains(text, "echo user-hook") {
		t.Fatalf("existing user hook content should be preserved")
	}

	if err := UninstallForRepo(repoRoot, []string{"git.pre_commit"}); err != nil {
		t.Fatalf("uninstall git surface failed: %v", err)
	}
	b, err = os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read uninstalled hook: %v", err)
	}
	text = string(b)
	if strings.Contains(text, start) || strings.Contains(text, end) {
		t.Fatalf("managed block should be removed")
	}
	if !strings.Contains(text, "echo user-hook") {
		t.Fatalf("existing user hook content should remain after uninstall")
	}
}
