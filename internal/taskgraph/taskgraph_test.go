package taskgraph

import (
	"testing"

	"atrakta/internal/model"
)

func TestBuildFromOpsAssignsDependenciesAndIsAcyclic(t *testing.T) {
	ops := []model.Operation{
		{Op: "link", Path: ".cursor/AGENTS.md"},
		{Op: "write", Path: ".cursor/AGENTS.md"},
		{Op: "link", Path: ".windsurf/AGENTS.md"},
		{Op: "delete", Path: ".cursor"},
	}
	annotated, g, err := BuildFromOps("plan-1", ops)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(annotated) != len(ops) {
		t.Fatalf("unexpected annotated size")
	}
	if g.TaskCount != 4 {
		t.Fatalf("unexpected task count: %d", g.TaskCount)
	}
	if annotated[0].TaskID == "" || annotated[3].TaskID == "" {
		t.Fatalf("task ids were not assigned")
	}
	if len(annotated[1].TaskBlockedBy) == 0 {
		t.Fatalf("expected write op to depend on prior conflicting op")
	}
	if len(annotated[3].TaskBlockedBy) == 0 {
		t.Fatalf("expected destructive dir delete to have dependencies")
	}
	ordered, err := TopoOrder(annotated)
	if err != nil {
		t.Fatalf("topological order failed: %v", err)
	}
	if len(ordered) != len(annotated) {
		t.Fatalf("unexpected ordered size")
	}
}

func TestTopoOrderDetectsCycle(t *testing.T) {
	ops := []model.Operation{
		{TaskID: "a", TaskBlockedBy: []string{"b"}, Op: "link", Path: "a"},
		{TaskID: "b", TaskBlockedBy: []string{"a"}, Op: "link", Path: "b"},
	}
	if _, err := TopoOrder(ops); err == nil {
		t.Fatalf("expected cycle detection error")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	repo := t.TempDir()
	_, g, err := BuildFromOps("plan-1", []model.Operation{{Op: "link", Path: ".cursor/AGENTS.md"}})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if err := Save(repo, g); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, ok, err := Load(repo)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected graph file to exist")
	}
	if loaded.GraphID == "" || loaded.PlanID != "plan-1" {
		t.Fatalf("unexpected loaded graph: %#v", loaded)
	}
}
