package run

import (
	"os"
	"testing"
)

func TestAutoStateRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := AutoState{
		InterfaceID:     "cursor",
		InterfaceSource: "flag",
	}
	if err := SaveAutoState(root, in); err != nil {
		t.Fatalf("save auto state: %v", err)
	}
	out, err := LoadAutoState(root)
	if err != nil {
		t.Fatalf("load auto state: %v", err)
	}
	if out.InterfaceID != "cursor" {
		t.Fatalf("interface id=%q", out.InterfaceID)
	}
	if out.SchemaVersion == "" {
		t.Fatal("schema version should be set")
	}
}

func TestLoadAutoStateMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadAutoState(root); err == nil {
		t.Fatal("expected missing auto-state error")
	}
}

func TestAutoStatePath(t *testing.T) {
	root := t.TempDir()
	path := AutoStatePath(root)
	if path == "" {
		t.Fatal("empty auto-state path")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected missing auto-state file")
	}
}
