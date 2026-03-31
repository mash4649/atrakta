package run

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveInterface(t *testing.T) {
	t.Run("explicit flag", func(t *testing.T) {
		root := t.TempDir()
		got, err := ResolveInterface(root, "manual", "")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "manual" || got.Source != "flag" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("env trigger", func(t *testing.T) {
		root := t.TempDir()
		got, err := ResolveInterface(root, "", "cursor")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "cursor" || got.Source != "env" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("binding detect from wrapper install path", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".atrakta", "wrap"), 0o755); err != nil {
			t.Fatalf("mkdir wrap dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, ".atrakta", "wrap", "cursor.sh"), []byte("# wrapper\n"), 0o755); err != nil {
			t.Fatalf("write wrapper: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
			t.Fatalf("write AGENTS.md: %v", err)
		}
		got, err := ResolveInterface(root, "", "")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "cursor" || got.Source != "detect" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("binding detect from autostart config", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".vscode"), 0o755); err != nil {
			t.Fatalf("mkdir .vscode: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, ".vscode", "tasks.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write tasks.json: %v", err)
		}
		got, err := ResolveInterface(root, "", "")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "vscode" || got.Source != "detect" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("binding detect from claude docs", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# test\n"), 0o644); err != nil {
			t.Fatalf("write CLAUDE.md: %v", err)
		}
		got, err := ResolveInterface(root, "", "")
		if err != nil {
			t.Fatalf("resolve interface: %v", err)
		}
		if got.InterfaceID != "claude-code" || got.Source != "detect" {
			t.Fatalf("unexpected resolution: %+v", got)
		}
	})

	t.Run("legacy detect fallback", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
			t.Fatalf("write AGENTS.md: %v", err)
		}
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
