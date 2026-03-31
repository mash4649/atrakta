package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProjectionRenderWritesTarget(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	if err := runProjection([]string{"render", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
		t.Fatalf("runProjection render: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(projectRoot, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "# Atrakta Projection") {
		t.Fatalf("rendered projection missing heading: %s", got)
	}
	if !strings.Contains(got, "## Contract Summary") {
		t.Fatalf("rendered projection missing contract summary: %s", got)
	}
}

func TestRunProjectionStatusClassifiesMissing(t *testing.T) {
	projectRoot := t.TempDir()

	raw := captureStdout(t, func() {
		if err := runProjection([]string{"status", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runProjection status: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal status output: %v", err)
	}
	if out["status"] != "drift" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["projection_status"] != "missing" {
		t.Fatalf("projection_status=%v", out["projection_status"])
	}
	types := readRunEventTypes(t, filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl"))
	if !containsString(types, runEventProjectionStatusCheck) {
		t.Fatalf("missing %s in %v", runEventProjectionStatusCheck, types)
	}
}

func TestRunProjectionStatusClassifiesUpToDate(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := runProjection([]string{"render", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
		t.Fatalf("seed projection render: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runProjection([]string{"status", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runProjection status: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal status output: %v", err)
	}
	if out["status"] != "ok" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["projection_status"] != "up_to_date" {
		t.Fatalf("projection_status=%v", out["projection_status"])
	}
}

func TestRunProjectionStatusClassifiesStale(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := runProjection([]string{"render", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
		t.Fatalf("seed projection render: %v", err)
	}
	targetPath := filepath.Join(projectRoot, "AGENTS.md")
	raw, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	stale := strings.Replace(string(raw), "## Workspace", "## Workspace\n- note: stale copy", 1)
	if stale == string(raw) {
		t.Fatal("failed to introduce stale drift")
	}
	if err := os.WriteFile(targetPath, []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale AGENTS.md: %v", err)
	}

	rawOut := captureStdout(t, func() {
		if err := runProjection([]string{"status", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runProjection status: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(rawOut, &out); err != nil {
		t.Fatalf("unmarshal status output: %v", err)
	}
	if out["status"] != "drift" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["projection_status"] != "stale" {
		t.Fatalf("projection_status=%v", out["projection_status"])
	}
}

func TestRunProjectionStatusClassifiesModifiedExternally(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# handwritten\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	rawOut := captureStdout(t, func() {
		if err := runProjection([]string{"status", "--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runProjection status: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(rawOut, &out); err != nil {
		t.Fatalf("unmarshal status output: %v", err)
	}
	if out["status"] != "drift" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["projection_status"] != "modified_externally" {
		t.Fatalf("projection_status=%v", out["projection_status"])
	}
}

func TestRunProjectionRepairRestoresMissingTarget(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := runProjection([]string{"render", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
		t.Fatalf("seed projection render: %v", err)
	}
	targetPath := filepath.Join(projectRoot, "AGENTS.md")
	if err := os.Remove(targetPath); err != nil {
		t.Fatalf("remove AGENTS.md: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runProjection([]string{"repair", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
			t.Fatalf("runProjection repair: %v", err)
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal repair output: %v", err)
	}
	if out["status"] != "repaired" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["projection_status"] != "up_to_date" {
		t.Fatalf("projection_status=%v", out["projection_status"])
	}

	b, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(b), "# Atrakta Projection") {
		t.Fatalf("repair did not restore rendered content: %s", b)
	}

	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventProjectionRepaired) {
		t.Fatalf("missing %s in %v", runEventProjectionRepaired, types)
	}
}

func TestRunProjectionRepairRestoresStaleTarget(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := runProjection([]string{"render", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
		t.Fatalf("seed projection render: %v", err)
	}
	targetPath := filepath.Join(projectRoot, "AGENTS.md")
	raw, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	stale := strings.Replace(string(raw), "## Workspace", "## Workspace\n- note: stale copy", 1)
	if err := os.WriteFile(targetPath, []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale AGENTS.md: %v", err)
	}

	rawRepair := captureStdout(t, func() {
		if err := runProjection([]string{"repair", "--project-root", projectRoot, "--approve", "--json"}); err != nil {
			t.Fatalf("runProjection repair: %v", err)
		}
	})
	var repairOut map[string]any
	if err := json.Unmarshal(rawRepair, &repairOut); err != nil {
		t.Fatalf("unmarshal repair output: %v", err)
	}
	if repairOut["status"] != "repaired" {
		t.Fatalf("status=%v", repairOut["status"])
	}
	if repairOut["projection_status"] != "up_to_date" {
		t.Fatalf("projection_status=%v", repairOut["projection_status"])
	}

	b, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(b), "# Atrakta Projection") {
		t.Fatalf("repair did not restore rendered content: %s", b)
	}
}

func TestRunProjectionRepairBlocksExternallyModifiedTarget(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# handwritten\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	raw := captureStdout(t, func() {
		err := runProjection([]string{"repair", "--project-root", projectRoot, "--approve", "--json"})
		if err == nil {
			t.Fatal("expected repair to fail for externally modified target")
		}
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal blocked repair output: %v", err)
	}
	if out["status"] != "blocked" {
		t.Fatalf("status=%v", out["status"])
	}
	if out["projection_status"] != "modified_externally" {
		t.Fatalf("projection_status=%v", out["projection_status"])
	}
	errPayload, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("error payload missing: %#v", out["error"])
	}
	if errPayload["code"] != "ERR_BLOCKED" {
		t.Fatalf("error code=%v", errPayload["code"])
	}
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	types := readRunEventTypes(t, runEventsPath)
	if !containsString(types, runEventProjectionRepaired) {
		t.Fatalf("missing %s in %v", runEventProjectionRepaired, types)
	}
}
