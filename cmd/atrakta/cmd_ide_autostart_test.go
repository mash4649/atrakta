package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunIDEAutostartDryRun(t *testing.T) {
	projectRoot := t.TempDir()
	raw := captureStdout(t, func() {
		code, err := runIDEAutostart([]string{"--project-root", projectRoot, "--dry-run", "--json"})
		if err != nil {
			t.Fatalf("runIDEAutostart dry-run: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal dry-run output: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["next_allowed_action"] != "approve" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".vscode", "tasks.json")); !os.IsNotExist(err) {
		t.Fatalf(".vscode/tasks.json should not be written during dry-run")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".cursor", "autostart.json")); !os.IsNotExist(err) {
		t.Fatalf(".cursor/autostart.json should not be written during dry-run")
	}
}

func TestRunIDEAutostartApply(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".vscode"), 0o755); err != nil {
		t.Fatalf("mkdir .vscode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".vscode", "tasks.json"), []byte(`{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "existing task",
      "type": "shell",
      "command": "echo existing"
    }
  ]
}
`), 0o644); err != nil {
		t.Fatalf("seed tasks.json: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runIDEAutostart([]string{"--project-root", projectRoot, "--approve", "--json"})
		if err != nil {
			t.Fatalf("runIDEAutostart apply: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal apply output: %v", err)
	}
	if out["status"] != "installed" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["next_allowed_action"] != "done" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}

	tasksPath := filepath.Join(projectRoot, ".vscode", "tasks.json")
	b, err := os.ReadFile(tasksPath)
	if err != nil {
		t.Fatalf("read tasks.json: %v", err)
	}
	var tasksDoc map[string]any
	if err := json.Unmarshal(b, &tasksDoc); err != nil {
		t.Fatalf("unmarshal tasks.json: %v", err)
	}
	tasks, ok := tasksDoc["tasks"].([]any)
	if !ok {
		t.Fatalf("tasks type=%T", tasksDoc["tasks"])
	}
	if len(tasks) != 2 {
		t.Fatalf("task count=%d", len(tasks))
	}
	if got := tasks[0].(map[string]any)["label"]; got != "existing task" {
		t.Fatalf("first task label=%v", got)
	}
	if got := tasks[1].(map[string]any)["label"]; got != "atrakta start" {
		t.Fatalf("second task label=%v", got)
	}

	cursorPath := filepath.Join(projectRoot, ".cursor", "autostart.json")
	cursorBytes, err := os.ReadFile(cursorPath)
	if err != nil {
		t.Fatalf("read autostart.json: %v", err)
	}
	var cursorDoc map[string]any
	if err := json.Unmarshal(cursorBytes, &cursorDoc); err != nil {
		t.Fatalf("unmarshal autostart.json: %v", err)
	}
	if cursorDoc["command"] != "atrakta start" {
		t.Fatalf("cursor command=%v", cursorDoc["command"])
	}
	if cursorDoc["workspace_root"] != projectRoot {
		t.Fatalf("workspace_root=%v", cursorDoc["workspace_root"])
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventIDEAutostartInstall) {
		t.Fatalf("missing %s in %v", runEventIDEAutostartInstall, types)
	}
}
