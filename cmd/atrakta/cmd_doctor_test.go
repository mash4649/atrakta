package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/projection"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

type doctorTestOutput struct {
	Alias   string       `json:"alias"`
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Doctor  doctorReport `json:"doctor"`
}

func TestRunDoctorHealthy(t *testing.T) {
	projectRoot := t.TempDir()
	prepareDoctorHealthyProject(t, projectRoot)

	raw := captureStdout(t, func() {
		if err := runDoctor([]string{"--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})

	var out doctorTestOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal doctor output: %v", err)
	}
	if out.Status != "ok" {
		t.Fatalf("status=%q message=%q checks=%#v", out.Status, out.Message, out.Doctor.Checks)
	}
	wantIDs := []string{"state-integrity", "projection-parity", "event-chain", "security-model"}
	if len(out.Doctor.Checks) != len(wantIDs) {
		t.Fatalf("checks=%d want=%d", len(out.Doctor.Checks), len(wantIDs))
	}
	for i, wantID := range wantIDs {
		got := out.Doctor.Checks[i]
		if got.CheckID != wantID {
			t.Fatalf("check[%d].id=%q want=%q", i, got.CheckID, wantID)
		}
		if got.Status != "ok" {
			t.Fatalf("check[%d]=%#v", i, got)
		}
	}
}

func TestRunDoctorDetectsStateDrift(t *testing.T) {
	projectRoot := t.TempDir()
	prepareDoctorHealthyProject(t, projectRoot)
	writeDoctorSessionFile(t, filepath.Join(projectRoot, ".atrakta", "state.json"), "session-state.v0", "start", runpkg.StateOnboardingComplete)

	raw := captureStdout(t, func() {
		if err := runDoctor([]string{"--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})

	var out doctorTestOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal doctor output: %v", err)
	}
	if out.Status != "needs_attention" {
		t.Fatalf("status=%q message=%q", out.Status, out.Message)
	}
	if out.Doctor.Checks[0].CheckID != "state-integrity" || out.Doctor.Checks[0].Status != "needs_attention" {
		t.Fatalf("state check=%#v", out.Doctor.Checks[0])
	}
}

func TestRunDoctorDetectsProjectionDrift(t *testing.T) {
	projectRoot := t.TempDir()
	prepareDoctorHealthyProject(t, projectRoot)
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# drift\n"), 0o644); err != nil {
		t.Fatalf("tamper AGENTS.md: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runDoctor([]string{"--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})

	var out doctorTestOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal doctor output: %v", err)
	}
	if out.Doctor.Checks[1].CheckID != "projection-parity" || out.Doctor.Checks[1].Status != "needs_attention" {
		t.Fatalf("projection check=%#v", out.Doctor.Checks[1])
	}
}

func TestRunDoctorDetectsEventChainTamper(t *testing.T) {
	projectRoot := t.TempDir()
	prepareDoctorHealthyProject(t, projectRoot)
	runEventsPath := filepath.Join(projectRoot, ".atrakta", "audit", "events", "run-events.jsonl")
	if err := os.WriteFile(runEventsPath, []byte("not-json\n"), 0o644); err != nil {
		t.Fatalf("tamper run-events: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runDoctor([]string{"--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})

	var out doctorTestOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal doctor output: %v", err)
	}
	if out.Doctor.Checks[2].CheckID != "event-chain" || out.Doctor.Checks[2].Status != "needs_attention" {
		t.Fatalf("event-chain check=%#v", out.Doctor.Checks[2])
	}
}

func TestRunDoctorDetectsSecurityDrift(t *testing.T) {
	projectRoot := t.TempDir()
	prepareDoctorHealthyProject(t, projectRoot)
	if err := os.WriteFile(filepath.Join(projectRoot, ".atrakta", "contract.json"), []byte(`{
  "v": 1,
  "project_id": "test",
  "interfaces": {"supported": ["generic-cli"], "fallback": "generic-cli"},
  "boundary": {"managed_root": ".atrakta/"},
  "tools": {"allow": ["create", "edit", "run"]},
  "security": {
    "destructive": "allow",
    "external_send": "deny",
    "approval": "explicit",
    "permission_model": "proposal_only"
  },
  "routing": {"default": {"worker": "general"}}
}
`), 0o644); err != nil {
		t.Fatalf("tamper contract: %v", err)
	}

	raw := captureStdout(t, func() {
		if err := runDoctor([]string{"--project-root", projectRoot, "--json"}); err != nil {
			t.Fatalf("runDoctor: %v", err)
		}
	})

	var out doctorTestOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal doctor output: %v", err)
	}
	if out.Doctor.Checks[3].CheckID != "security-model" || out.Doctor.Checks[3].Status != "needs_attention" {
		t.Fatalf("security check=%#v", out.Doctor.Checks[3])
	}
}

func prepareDoctorHealthyProject(t *testing.T, root string) {
	t.Helper()
	dirs := []string{
		filepath.Join(root, ".atrakta", "canonical", "policies", "registry"),
		filepath.Join(root, ".atrakta", "state"),
		filepath.Join(root, ".atrakta", "audit", "events"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".atrakta", "canonical", "policies", "registry", "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write canonical index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".atrakta", "state", "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}
	writeDoctorSessionFile(t, filepath.Join(root, ".atrakta", "state.json"), "session-state.v1", "start", runpkg.StateOnboardingComplete)
	writeDoctorTaskGraphFile(t, filepath.Join(root, ".atrakta", "task-graph.json"), "session-task-graph.v1", "start")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Atrakta Projection\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	writeTestMachineContract(t, root)
	if _, err := projection.Write(root, "agents_md", ""); err != nil {
		t.Fatalf("projection write: %v", err)
	}
	if _, err := audit.AppendRunEventAndVerify(filepath.Join(root, ".atrakta", "audit"), audit.LevelA2, "start.begin", map[string]any{
		"status": "ok",
	}, audit.RunEventOptions{Actor: "kernel"}); err != nil {
		t.Fatalf("append run event: %v", err)
	}
}

func writeDoctorSessionFile(t *testing.T, path, schemaVersion, command, canonicalState string) {
	t.Helper()
	payload := runpkg.SessionState{
		SchemaVersion:   schemaVersion,
		Command:         command,
		CanonicalState:  canonicalState,
		Status:          "ok",
		InterfaceID:     "generic-cli",
		InterfaceSource: "detect",
		ApplyRequested:  false,
		Approved:        false,
		PlannedCount:    0,
		AppliedCount:    0,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

func writeDoctorTaskGraphFile(t *testing.T, path, schemaVersion, command string) {
	t.Helper()
	payload := runpkg.SessionTaskGraph{
		SchemaVersion: schemaVersion,
		Command:       command,
		Nodes: []runpkg.SessionTaskNode{
			{ID: "n1", Kind: "projection", Target: "AGENTS.md", Status: "done"},
		},
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal task graph: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write task graph: %v", err)
	}
}
