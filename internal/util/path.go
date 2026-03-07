package util

import (
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeRelPath returns a normalized repo-relative path with forward slashes.
func NormalizeRelPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	cleaned := filepath.ToSlash(filepath.Clean(p))
	if cleaned == "." {
		return ""
	}
	return strings.TrimPrefix(cleaned, "./")
}

func NormalizeContentLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}
