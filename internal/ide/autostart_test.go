package ide

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCheckUninstallRoundtrip(t *testing.T) {
	repo := t.TempDir()

	st := Check(repo)
	if st.FileExists || st.Installed {
		t.Fatalf("expected empty status before install: %#v", st)
	}

	changed, path, err := Install(repo)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !changed {
		t.Fatalf("expected install to create managed task")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("tasks file missing: %v", err)
	}

	st = Check(repo)
	if !st.FileExists || !st.Installed || st.ManagedTaskN != 1 {
		t.Fatalf("unexpected status after install: %#v", st)
	}

	changed, _, err = Install(repo)
	if err != nil {
		t.Fatalf("reinstall failed: %v", err)
	}
	if changed {
		t.Fatalf("expected reinstall to be idempotent")
	}

	changed, _, err = Uninstall(repo)
	if err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}
	if !changed {
		t.Fatalf("expected uninstall to remove managed task")
	}
	st = Check(repo)
	if st.ManagedTaskN != 0 || st.Installed {
		t.Fatalf("expected managed task removed: %#v", st)
	}
}

func TestInstallPreservesNonManagedTasks(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".vscode", "tasks.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "existing",
      "type": "shell",
      "command": "echo hi"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write tasks: %v", err)
	}
	changed, _, err := Install(repo)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !changed {
		t.Fatalf("expected install change")
	}
	st := Check(repo)
	if st.ManagedTaskN != 1 || st.OtherTaskN != 1 {
		t.Fatalf("expected one managed + one existing task: %#v", st)
	}
}
