package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func WorkspaceStamp(repoRoot string, includes []string) string {
	if s := gitStamp(repoRoot); s != "" {
		return s
	}
	return fsStamp(repoRoot, includes)
}

func WorkspaceStampDeep(repoRoot string) string {
	gitMeta := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitMeta); err != nil {
		return ""
	}
	head, err1 := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	status, err2 := exec.Command("git", "-C", repoRoot, "status", "--porcelain", "-uno").Output()
	if err1 != nil || err2 != nil {
		return ""
	}
	joined := strings.TrimSpace(string(head)) + "\n" + strings.TrimSpace(string(status))
	return SHA256Tagged([]byte(joined))
}

func gitStamp(repoRoot string) string {
	gitMeta := filepath.Join(repoRoot, ".git")
	gitDir, ok := resolveGitDir(gitMeta, repoRoot)
	if !ok {
		return ""
	}
	headPath := filepath.Join(gitDir, "HEAD")
	head, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	indexPath := filepath.Join(gitDir, "index")
	var idxMod, idxSize int64
	if fi, err := os.Stat(indexPath); err == nil {
		idxMod = fi.ModTime().UnixNano()
		idxSize = fi.Size()
	}
	joined := strings.TrimSpace(string(head)) + fmt.Sprintf("\nindex:%d:%d", idxMod, idxSize)
	return SHA256Tagged([]byte(joined))
}

func fsStamp(repoRoot string, includes []string) string {
	type sample struct {
		path string
		mod  int64
		size int64
	}
	items := []sample{}
	for _, inc := range includes {
		ninc := NormalizeRelPath(inc)
		base := repoRoot
		if ninc != "" {
			base = filepath.Join(repoRoot, filepath.FromSlash(ninc))
		}
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := strings.TrimSpace(e.Name())
			if name == ".atrakta" || name == ".git" || name == ".tmp" {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			items = append(items, sample{path: filepath.Join(ninc, e.Name()), mod: fi.ModTime().UnixNano(), size: fi.Size()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].path < items[j].path })
	var b strings.Builder
	for _, it := range items {
		fmt.Fprintf(&b, "%s:%d:%d\n", it.path, it.mod, it.size)
	}
	return SHA256Tagged([]byte(b.String()))
}

func resolveGitDir(gitMetaPath, repoRoot string) (string, bool) {
	fi, err := os.Stat(gitMetaPath)
	if err != nil {
		return "", false
	}
	if fi.IsDir() {
		return gitMetaPath, true
	}
	b, err := os.ReadFile(gitMetaPath)
	if err != nil {
		return "", false
	}
	line := strings.TrimSpace(string(b))
	const prefix = "gitdir:"
	if !strings.HasPrefix(strings.ToLower(line), prefix) {
		return "", false
	}
	ref := strings.TrimSpace(line[len(prefix):])
	if ref == "" {
		return "", false
	}
	if filepath.IsAbs(ref) {
		return ref, true
	}
	return filepath.Clean(filepath.Join(repoRoot, ref)), true
}
