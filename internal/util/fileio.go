package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileLockOptions struct {
	Timeout      time.Duration
	RetryDelay   time.Duration
	StaleAfter   time.Duration
	LockFilePerm os.FileMode
}

func DefaultFileLockOptions() FileLockOptions {
	return FileLockOptions{
		Timeout:      2500 * time.Millisecond,
		RetryDelay:   20 * time.Millisecond,
		StaleAfter:   45 * time.Second,
		LockFilePerm: 0o600,
	}
}

func WithFileLock(lockPath string, opts FileLockOptions, fn func() error) error {
	if fn == nil {
		return fmt.Errorf("lock function is nil")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultFileLockOptions().Timeout
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = DefaultFileLockOptions().RetryDelay
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = DefaultFileLockOptions().StaleAfter
	}
	if opts.LockFilePerm == 0 {
		opts.LockFilePerm = DefaultFileLockOptions().LockFilePerm
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("mkdir lock dir: %w", err)
	}
	deadline := time.Now().Add(opts.Timeout)
	for {
		fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, opts.LockFilePerm)
		if err == nil {
			_, _ = fd.WriteString(fmt.Sprintf("pid=%d ts=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano)))
			_ = fd.Close()
			defer func() { _ = os.Remove(lockPath) }()
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("acquire lock: %w", err)
		}
		if stale := isLockStale(lockPath, opts.StaleAfter); stale {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("lock timeout for %s after %s", filepath.Base(lockPath), opts.Timeout)
		}
		time.Sleep(opts.RetryDelay)
	}
}

func AtomicWriteFile(filePath string, data []byte, perm os.FileMode) error {
	if perm == 0 {
		perm = 0o644
	}
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(filePath)+"-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	defer cleanup()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func isLockStale(lockPath string, staleAfter time.Duration) bool {
	if staleAfter <= 0 {
		return false
	}
	fi, err := os.Stat(lockPath)
	if err != nil {
		return false
	}
	return time.Since(fi.ModTime()) > staleAfter
}
