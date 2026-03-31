package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStatePaths(t *testing.T) {
	root := t.TempDir()
	if SessionStatePath(root) == "" {
		t.Fatal("empty state path")
	}
	if SessionProgressPath(root) == "" {
		t.Fatal("empty progress path")
	}
	if SessionTaskGraphPath(root) == "" {
		t.Fatal("empty task-graph path")
	}
}

func TestSaveSessionArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := SaveSessionState(root, SessionState{
		Command:         "start",
		CanonicalState:  StateOnboardingComplete,
		Status:          "ok",
		InterfaceID:     "generic-cli",
		InterfaceSource: "detect",
		PlannedCount:    2,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := SaveSessionProgress(root, SessionProgress{
		Command:           "start",
		Status:            "ok",
		NextAllowedAction: "inspect",
		PlannedCount:      2,
	}); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	if err := SaveSessionTaskGraph(root, SessionTaskGraph{
		Command: "start",
		Nodes: []SessionTaskNode{
			{ID: "plan-1", Kind: "projection_write", Target: ".atrakta/generated/repo-map.generated.json", Status: "planned"},
		},
	}); err != nil {
		t.Fatalf("save task graph: %v", err)
	}

	if _, err := os.Stat(SessionStatePath(root)); err != nil {
		t.Fatalf("state missing: %v", err)
	}
	if _, err := os.Stat(SessionProgressPath(root)); err != nil {
		t.Fatalf("progress missing: %v", err)
	}
	if _, err := os.Stat(SessionTaskGraphPath(root)); err != nil {
		t.Fatalf("task graph missing: %v", err)
	}

	var state SessionState
	stateRaw, err := os.ReadFile(SessionStatePath(root))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if state.SchemaVersion != sessionStateSchemaVersion {
		t.Fatalf("state schema version=%q", state.SchemaVersion)
	}

	var progress SessionProgress
	progressRaw, err := os.ReadFile(SessionProgressPath(root))
	if err != nil {
		t.Fatalf("read progress: %v", err)
	}
	if err := json.Unmarshal(progressRaw, &progress); err != nil {
		t.Fatalf("decode progress: %v", err)
	}
	if progress.SchemaVersion != sessionProgressSchemaVersion {
		t.Fatalf("progress schema version=%q", progress.SchemaVersion)
	}

	graph, err := LoadSessionTaskGraph(root)
	if err != nil {
		t.Fatalf("load task graph: %v", err)
	}
	if graph.SchemaVersion != sessionTaskGraphSchemaVersion {
		t.Fatalf("task graph schema version=%q", graph.SchemaVersion)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("task graph nodes=%d", len(graph.Nodes))
	}
}

func TestLoadSessionArtifactsAcceptLegacyVersions(t *testing.T) {
	root := t.TempDir()
	legacyState := SessionState{
		SchemaVersion:   sessionStateSchemaVersionLegacy,
		Command:         "resume",
		CanonicalState:  StateOnboardingComplete,
		Status:          "ok",
		InterfaceID:     "generic-cli",
		InterfaceSource: "flag",
		ApplyRequested:  false,
		Approved:        false,
		PlannedCount:    1,
		AppliedCount:    0,
	}
	legacyProgress := SessionProgress{
		SchemaVersion:     sessionProgressSchemaVersionLegacy,
		Command:           "resume",
		Status:            "applied",
		NextAllowedAction: "inspect",
		PlannedCount:      1,
		AppliedCount:      1,
	}
	stateRaw, err := json.MarshalIndent(legacyState, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy state: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(SessionStatePath(root)), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(SessionStatePath(root), append(stateRaw, '\n'), 0o644); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}
	progressRaw, err := json.MarshalIndent(legacyProgress, "", "  ")
	if err != nil {
		t.Fatalf("marshal legacy progress: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(SessionProgressPath(root)), 0o755); err != nil {
		t.Fatalf("mkdir progress dir: %v", err)
	}
	if err := os.WriteFile(SessionProgressPath(root), append(progressRaw, '\n'), 0o644); err != nil {
		t.Fatalf("write legacy progress: %v", err)
	}

	state, err := LoadSessionState(root)
	if err != nil {
		t.Fatalf("load legacy state: %v", err)
	}
	if state.SchemaVersion != sessionStateSchemaVersionLegacy {
		t.Fatalf("state schema version=%q", state.SchemaVersion)
	}
	progress, err := LoadSessionProgress(root)
	if err != nil {
		t.Fatalf("load legacy progress: %v", err)
	}
	if progress.SchemaVersion != sessionProgressSchemaVersionLegacy {
		t.Fatalf("progress schema version=%q", progress.SchemaVersion)
	}
}

func TestSaveSessionTaskGraphMergesExistingStatuses(t *testing.T) {
	root := t.TempDir()
	first := SessionTaskGraph{
		Command: "start",
		Nodes: []SessionTaskNode{
			{ID: "task-1", Kind: "managed_mutation", Target: "AGENTS.md", Status: "applied"},
		},
	}
	if err := SaveSessionTaskGraph(root, first); err != nil {
		t.Fatalf("save first graph: %v", err)
	}
	second := SessionTaskGraph{
		Command: "resume",
		Nodes: []SessionTaskNode{
			{ID: "task-1", Kind: "managed_mutation", Target: "AGENTS.md", Status: "planned"},
		},
	}
	if err := SaveSessionTaskGraph(root, second); err != nil {
		t.Fatalf("save second graph: %v", err)
	}
	out, err := LoadSessionTaskGraph(root)
	if err != nil {
		t.Fatalf("load merged graph: %v", err)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("merged nodes=%d", len(out.Nodes))
	}
	if out.Nodes[0].Status != "applied" {
		t.Fatalf("merged status=%q", out.Nodes[0].Status)
	}
	if out.Command != "resume" {
		t.Fatalf("merged command=%q", out.Command)
	}
}

func TestSaveSessionTaskGraphPreservesExistingOnEmptyReplay(t *testing.T) {
	root := t.TempDir()
	first := SessionTaskGraph{
		Command: "start",
		Nodes: []SessionTaskNode{
			{ID: "task-1", Kind: "managed_mutation", Target: "AGENTS.md", Status: "done"},
		},
	}
	if err := SaveSessionTaskGraph(root, first); err != nil {
		t.Fatalf("save first graph: %v", err)
	}
	if err := SaveSessionTaskGraph(root, SessionTaskGraph{Command: "resume"}); err != nil {
		t.Fatalf("save empty graph: %v", err)
	}
	out, err := LoadSessionTaskGraph(root)
	if err != nil {
		t.Fatalf("load replay graph: %v", err)
	}
	if len(out.Nodes) != 1 {
		t.Fatalf("replay nodes=%d", len(out.Nodes))
	}
	if out.Nodes[0].Status != "done" {
		t.Fatalf("replay status=%q", out.Nodes[0].Status)
	}
}

func TestSaveSessionArtifactsRejectInvalidShape(t *testing.T) {
	root := t.TempDir()
	err := SaveSessionState(root, SessionState{
		Command:        "start",
		CanonicalState: StateOnboardingComplete,
		Status:         "ok",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
