package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/util"
)

func ValidateMutationPath(repoRoot, rel string, b contract.Boundary) error {
	rel = util.NormalizeRelPath(rel)
	if rel == "" {
		return fmt.Errorf("empty path")
	}
	if rel == "." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
		return fmt.Errorf("path traversal blocked: %s", rel)
	}
	abs := filepath.Clean(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	repoClean := filepath.Clean(repoRoot)
	if abs != repoClean && !strings.HasPrefix(abs, repoClean+string(filepath.Separator)) {
		return fmt.Errorf("path escapes repo root: %s", rel)
	}
	norm := rel
	if !isInBoundary(norm, b.Include) {
		return fmt.Errorf("path outside include boundary: %s", rel)
	}
	if isInBoundary(norm, b.Exclude) {
		return fmt.Errorf("path inside exclude boundary: %s", rel)
	}
	if err := rejectExternalSymlinkTraversal(repoRoot, rel); err != nil {
		return err
	}
	return nil
}

func isInBoundary(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}
	for _, p := range prefixes {
		n := util.NormalizeRelPath(p)
		if n == "" {
			return true
		}
		if strings.HasPrefix(path+"/", n+"/") || path == n {
			return true
		}
	}
	return false
}

func rejectExternalSymlinkTraversal(repoRoot, rel string) error {
	parts := strings.Split(util.NormalizeRelPath(rel), "/")
	cur := filepath.Clean(repoRoot)
	repoClean := filepath.Clean(repoRoot)
	for i := 0; i < len(parts)-1; i++ {
		cur = filepath.Join(cur, parts[i])
		fi, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("lstat %s: %w", cur, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			continue
		}
		resolved, err := filepath.EvalSymlinks(cur)
		if err != nil {
			return fmt.Errorf("eval symlink %s: %w", cur, err)
		}
		resolved = filepath.Clean(resolved)
		if resolved != repoClean && !strings.HasPrefix(resolved, repoClean+string(filepath.Separator)) {
			return fmt.Errorf("external symlink traversal blocked: %s", util.NormalizeRelPath(cur[len(repoRoot):]))
		}
	}
	return nil
}
