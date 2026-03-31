package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

func TestRunWrapInstallDryRunJson(t *testing.T) {
	projectRoot := t.TempDir()

	raw := captureStdout(t, func() {
		code, err := runWrap([]string{"install", "--tool", "generic-cli", "--project-root", projectRoot, "--dry-run", "--json"})
		if err != nil {
			t.Fatalf("runWrap dry-run: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal wrap output: %v", err)
	}
	if out["tool_id"] != "generic-cli" {
		t.Fatalf("tool_id=%v", out["tool_id"])
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["next_allowed_action"] != "approve" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	if got := out["installed_path"]; got == "" {
		t.Fatal("installed_path missing")
	}
	if want := filepath.ToSlash(filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")); out["installed_path"] != want {
		t.Fatalf("installed_path=%v want=%q", out["installed_path"], want)
	}
}

func TestRunWrapInstallNeedsApprovalWhenNonInteractive(t *testing.T) {
	projectRoot := t.TempDir()

	raw := captureStdout(t, func() {
		code, err := runWrap([]string{"install", "--tool", "generic-cli", "--project-root", projectRoot, "--non-interactive", "--json"})
		if err != nil {
			t.Fatalf("runWrap install: %v", err)
		}
		if code != exitNeedsApproval {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal wrap output: %v", err)
	}
	if out["status"] != "needs_approval" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["tool_id"] != "generic-cli" {
		t.Fatalf("tool_id=%v", out["tool_id"])
	}
}

func TestRunWrapInstallWritesScriptOnApprove(t *testing.T) {
	projectRoot := t.TempDir()

	code, err := runWrap([]string{"install", "--tool", "generic-cli", "--project-root", projectRoot, "--approve", "--json"})
	if err != nil {
		t.Fatalf("runWrap approve: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}

	scriptPath := filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("script should not be empty")
	}
	if !strings.Contains(string(raw), "tool_id: generic-cli") {
		t.Fatalf("script content=%s", string(raw))
	}
	if !strings.Contains(string(raw), "exec atrakta start \"$@\"") {
		t.Fatalf("script content=%s", string(raw))
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventWrapInstall) {
		t.Fatalf("missing %s in %v", runEventWrapInstall, types)
	}
}

func TestRunWrapUninstallDryRunJson(t *testing.T) {
	projectRoot := t.TempDir()

	if _, err := runWrap([]string{"install", "--tool", "generic-cli", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runWrap([]string{"uninstall", "--tool", "generic-cli", "--project-root", projectRoot, "--dry-run", "--json"})
		if err != nil {
			t.Fatalf("runWrap uninstall dry-run: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal wrap output: %v", err)
	}
	if out["tool_id"] != "generic-cli" {
		t.Fatalf("tool_id=%v", out["tool_id"])
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if got := out["removed_path"]; got == "" {
		t.Fatal("removed_path missing")
	}
	scriptPath := filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("script should remain during dry-run: %v", err)
	}
}

func TestRunWrapUninstallRemovesManagedScript(t *testing.T) {
	projectRoot := t.TempDir()

	if _, err := runWrap([]string{"install", "--tool", "generic-cli", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}

	code, err := runWrap([]string{"uninstall", "--tool", "generic-cli", "--project-root", projectRoot, "--json"})
	if err != nil {
		t.Fatalf("runWrap uninstall: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}

	scriptPath := filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Fatalf("script should be removed, stat err=%v", err)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventWrapUninstall) {
		t.Fatalf("missing %s in %v", runEventWrapUninstall, types)
	}
}

func TestRunWrapUninstallRejectsUnmanagedScript(t *testing.T) {
	projectRoot := t.TempDir()
	scriptPath := filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir wrap dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env sh\necho unmanaged\n"), 0o755); err != nil {
		t.Fatalf("write unmanaged script: %v", err)
	}

	code, err := runWrap([]string{"uninstall", "--tool", "generic-cli", "--project-root", projectRoot})
	if err == nil {
		t.Fatal("expected unmanaged uninstall to fail")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		t.Fatalf("unmanaged script should remain: %v", statErr)
	}
}

func TestRunWrapRunExecutesManagedScript(t *testing.T) {
	projectRoot, err := onboarding.DetectProjectRoot("")
	if err != nil {
		t.Fatalf("detect project root: %v", err)
	}
	scriptPath := filepath.Join(projectRoot, ".atrakta", "wrap", "generic-cli.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir wrap dir: %v", err)
	}

	argsPath := filepath.Join(projectRoot, ".atrakta", "run-args.txt")
	script := strings.Join([]string{
		"#!/usr/bin/env sh",
		"# managed by atrakta wrap install",
		"# tool_id: generic-cli",
		"printf '%s\\n' \"$@\" > \"" + argsPath + "\"",
		"exit 0",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write managed script: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Join(projectRoot, ".atrakta"))
	})

	code, err := runWrap([]string{"run", "generic-cli", "alpha", "beta"})
	if err != nil {
		t.Fatalf("runWrap run: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}

	raw, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read forwarded args: %v", err)
	}
	if got := strings.TrimSpace(string(raw)); got != "alpha\nbeta" {
		t.Fatalf("forwarded args=%q", got)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventWrapRun) {
		t.Fatalf("missing %s in %v", runEventWrapRun, types)
	}
}

func TestRunWrapRunMissingScript(t *testing.T) {
	_, err := onboarding.DetectProjectRoot("")
	if err != nil {
		t.Fatalf("detect project root: %v", err)
	}

	code, err := runWrap([]string{"run", "generic-cli"})
	if err == nil {
		t.Fatal("expected missing script to fail")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
	if !strings.Contains(err.Error(), "run `atrakta wrap install generic-cli`") {
		t.Fatalf("error=%v", err)
	}
}
