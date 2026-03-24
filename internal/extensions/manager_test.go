package extensions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "extensions", "manifests")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{
  "name":"default",
  "items":[
    {"id":"policy-1","kind":"policy","enabled":true},
    {"id":"workflow-1","kind":"workflow","enabled":true},
    {"id":"skill-1","kind":"skill","enabled":true},
    {"id":"hook-1","kind":"hook","enabled":false}
  ]
}`
	if err := os.WriteFile(filepath.Join(manifestDir, "default.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	out, err := Resolve(root)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(out.OrderedIDs) == 0 {
		t.Fatalf("ordered ids required")
	}
	if out.OrderedIDs[0] != "policy-1" {
		t.Fatalf("expected policy-1 first, got %q", out.OrderedIDs[0])
	}
}
