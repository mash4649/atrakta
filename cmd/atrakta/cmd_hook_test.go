package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHookInstallDryRunJson(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runHook([]string{"install", "--hook-type", "pre-commit", "--project-root", projectRoot, "--dry-run", "--json"})
		if err != nil {
			t.Fatalf("runHook dry-run: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if out["hook_type"] != "pre-commit" {
		t.Fatalf("hook_type=%v", out["hook_type"])
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if got := out["hook_path"]; got == "" {
		t.Fatal("hook_path missing")
	}
	want := filepath.ToSlash(filepath.Join(projectRoot, ".git", "hooks", "pre-commit"))
	if out["hook_path"] != want {
		t.Fatalf("hook_path=%v want=%q", out["hook_path"], want)
	}
}

func TestRunHookInstallWritesManagedHook(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	code, err := runHook([]string{"install", "--hook-type", "post-checkout", "--project-root", projectRoot, "--approve", "--json"})
	if err != nil {
		t.Fatalf("runHook approve: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}

	hookPath := filepath.Join(projectRoot, ".git", "hooks", "post-checkout")
	raw, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "# managed by atrakta hook install") {
		t.Fatalf("hook content=%s", content)
	}
	if !strings.Contains(content, "# hook_type: post-checkout") {
		t.Fatalf("hook content=%s", content)
	}
	if !strings.Contains(content, "exec atrakta start \"$@\"") {
		t.Fatalf("hook content=%s", content)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventHookInstall) {
		t.Fatalf("missing %s in %v", runEventHookInstall, types)
	}
}

func TestRunHookInstallRejectsExistingHook(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git", "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env sh\necho keep\n"), 0o755); err != nil {
		t.Fatalf("write existing hook: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runHook([]string{"install", "--hook-type", "pre-commit", "--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("runHook existing: %v", err)
		}
		if code != exitNeedsApproval {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if out["status"] != "needs_approval" {
		t.Fatalf("status=%v", out["status"])
	}
	nextAllowed, _ := out["next_allowed_action"].(string)
	if nextAllowed == "" || !strings.Contains(nextAllowed, "echo keep") {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	rawHook, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read existing hook: %v", err)
	}
	if !strings.Contains(string(rawHook), "echo keep") {
		t.Fatalf("existing hook should remain untouched: %s", string(rawHook))
	}
}

func TestRunHookUninstallDryRunJson(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if _, err := runHook([]string{"install", "--hook-type", "pre-commit", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}
	if _, err := runHook([]string{"install", "--hook-type", "post-checkout", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install post-checkout: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runHook([]string{"uninstall", "--hook-type", "pre-commit", "--project-root", projectRoot, "--dry-run", "--json"})
		if err != nil {
			t.Fatalf("runHook uninstall dry-run: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if out["hook_type"] != "pre-commit" {
		t.Fatalf("hook_type=%v", out["hook_type"])
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("hook should remain during dry-run: %v", err)
	}
}

func TestRunHookUninstallRemovesManagedHook(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if _, err := runHook([]string{"install", "--hook-type", "post-checkout", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}

	code, err := runHook([]string{"uninstall", "--hook-type", "post-checkout", "--project-root", projectRoot, "--json"})
	if err != nil {
		t.Fatalf("runHook uninstall: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}

	hookPath := filepath.Join(projectRoot, ".git", "hooks", "post-checkout")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("hook should be removed, stat err=%v", err)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventHookUninstall) {
		t.Fatalf("missing %s in %v", runEventHookUninstall, types)
	}
}

func TestRunHookUninstallSkipsUnmanagedHook(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git", "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env sh\necho keep\n"), 0o755); err != nil {
		t.Fatalf("write unmanaged hook: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runHook([]string{"uninstall", "--hook-type", "pre-commit", "--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("runHook unmanaged uninstall: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if out["status"] != "skipped" {
		t.Fatalf("status=%v", out["status"])
	}
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("unmanaged hook should remain: %v", err)
	}
}

func TestRunHookStatusReportsManagedAndMissing(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if _, err := runHook([]string{"install", "--hook-type", "pre-commit", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runHook([]string{"status", "--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("runHook status: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out hookStatusResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal hook status output: %v", err)
	}
	if out.Status != "missing" {
		t.Fatalf("status=%q", out.Status)
	}
	if len(out.Hooks) != 2 {
		t.Fatalf("hook count=%d", len(out.Hooks))
	}

	byType := make(map[string]hookStatusItem, len(out.Hooks))
	for _, item := range out.Hooks {
		byType[item.HookType] = item
	}

	preCommit, ok := byType["pre-commit"]
	if !ok {
		t.Fatal("missing pre-commit status item")
	}
	if preCommit.Status != "up_to_date" || !preCommit.Exists || !preCommit.Managed || preCommit.Drift {
		t.Fatalf("pre-commit status=%+v", preCommit)
	}

	postCheckout, ok := byType["post-checkout"]
	if !ok {
		t.Fatal("missing post-checkout status item")
	}
	if postCheckout.Status != "missing" || postCheckout.Exists || postCheckout.Managed || postCheckout.Drift {
		t.Fatalf("post-checkout status=%+v", postCheckout)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventHookStatusCheck) {
		t.Fatalf("missing %s in %v", runEventHookStatusCheck, types)
	}
}

func TestRunHookStatusDetectsDrift(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if _, err := runHook([]string{"install", "--hook-type", "pre-commit", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}
	if _, err := runHook([]string{"install", "--hook-type", "post-checkout", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install post-checkout: %v", err)
	}

	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")
	raw, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	drifted := strings.Replace(string(raw), "exec atrakta start \"$@\"", "exec atrakta start --drift \"$@\"", 1)
	if drifted == string(raw) {
		t.Fatal("failed to introduce drift")
	}
	if err := os.WriteFile(hookPath, []byte(drifted), 0o755); err != nil {
		t.Fatalf("write drifted hook: %v", err)
	}

	rawOut := captureStdout(t, func() {
		code, err := runHook([]string{"status", "--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("runHook status drift: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out hookStatusResponse
	if err := json.Unmarshal(rawOut, &out); err != nil {
		t.Fatalf("unmarshal hook status output: %v", err)
	}
	if out.Status != "drift" {
		t.Fatalf("status=%q", out.Status)
	}

	byType := make(map[string]hookStatusItem, len(out.Hooks))
	for _, item := range out.Hooks {
		byType[item.HookType] = item
	}

	preCommit, ok := byType["pre-commit"]
	if !ok {
		t.Fatal("missing pre-commit status item")
	}
	if preCommit.Status != "drift" || !preCommit.Exists || !preCommit.Managed || !preCommit.Drift {
		t.Fatalf("pre-commit drift status=%+v", preCommit)
	}
}

func TestRunHookRepairRestoresDriftedHook(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if _, err := runHook([]string{"install", "--hook-type", "pre-commit", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install: %v", err)
	}
	if _, err := runHook([]string{"install", "--hook-type", "post-checkout", "--project-root", projectRoot, "--approve"}); err != nil {
		t.Fatalf("seed install post-checkout: %v", err)
	}

	hookPath := filepath.Join(projectRoot, ".git", "hooks", "pre-commit")
	raw, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	drifted := strings.Replace(string(raw), "exec atrakta start \"$@\"", "exec atrakta start --repair \"$@\"", 1)
	if drifted == string(raw) {
		t.Fatal("failed to introduce drift")
	}
	if err := os.WriteFile(hookPath, []byte(drifted), 0o755); err != nil {
		t.Fatalf("write drifted hook: %v", err)
	}

	rawRepair := captureStdout(t, func() {
		code, err := runHook([]string{"repair", "--project-root", projectRoot, "--approve", "--json"})
		if err != nil {
			t.Fatalf("runHook repair: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var repairOut hookRepairResponse
	if err := json.Unmarshal(rawRepair, &repairOut); err != nil {
		t.Fatalf("unmarshal hook repair output: %v", err)
	}
	if repairOut.Status != "repaired" {
		t.Fatalf("repair status=%q", repairOut.Status)
	}
	if len(repairOut.Hooks) != 1 {
		t.Fatalf("repair hook count=%d", len(repairOut.Hooks))
	}
	if !repairOut.Hooks[0].Repaired || repairOut.Hooks[0].Status != "repaired" {
		t.Fatalf("repair item=%+v", repairOut.Hooks[0])
	}

	statusOutRaw := captureStdout(t, func() {
		code, err := runHook([]string{"status", "--project-root", projectRoot, "--json"})
		if err != nil {
			t.Fatalf("runHook status after repair: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var statusOut hookStatusResponse
	if err := json.Unmarshal(statusOutRaw, &statusOut); err != nil {
		t.Fatalf("unmarshal hook status output: %v", err)
	}
	if statusOut.Status != "ok" {
		t.Fatalf("status=%q", statusOut.Status)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventHookRepair) {
		t.Fatalf("missing %s in %v", runEventHookRepair, types)
	}
}
