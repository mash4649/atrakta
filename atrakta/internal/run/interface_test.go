package run

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInterface(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	t.Run("explicit flag", func(t *testing.T) {
		got, err := ResolveInterface(root, "manual", "")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "manual" || got.Source != "flag" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("env trigger", func(t *testing.T) {
		got, err := ResolveInterface(root, "", "cursor")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "cursor" || got.Source != "env" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("detect", func(t *testing.T) {
		got, err := ResolveInterface(root, "", "")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "generic-cli" || got.Source != "detect" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("unresolved", func(t *testing.T) {
		empty := t.TempDir()
		_, err := ResolveInterface(empty, "", "")
		if err == nil {
			t.Fatal("expected unresolved error")
		}
		if err != ErrInterfaceUnresolved {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
