package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFaultInjectionWithFileLockTimeoutOnContention(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".locks", "busy.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir lock dir failed: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("occupied\n"), 0o600); err != nil {
		t.Fatalf("write lock file failed: %v", err)
	}
	err := WithFileLock(lockPath, FileLockOptions{
		Timeout:      40 * time.Millisecond,
		RetryDelay:   10 * time.Millisecond,
		StaleAfter:   24 * time.Hour,
		LockFilePerm: 0o600,
	}, func() error { return nil })
	if err == nil {
		t.Fatalf("expected lock timeout under contention")
	}
	if !strings.Contains(err.Error(), "lock timeout") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFaultInjectionWithFileLockRecoversStaleLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), ".locks", "stale.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir lock dir failed: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte("stale\n"), 0o600); err != nil {
		t.Fatalf("write lock file failed: %v", err)
	}
	staleTS := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(lockPath, staleTS, staleTS); err != nil {
		t.Fatalf("chtimes lock file failed: %v", err)
	}
	called := false
	err := WithFileLock(lockPath, FileLockOptions{
		Timeout:      100 * time.Millisecond,
		RetryDelay:   10 * time.Millisecond,
		StaleAfter:   20 * time.Millisecond,
		LockFilePerm: 0o600,
	}, func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected stale lock recovery, got: %v", err)
	}
	if !called {
		t.Fatalf("expected lock callback to be called")
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock to be removed after callback")
	}
}
