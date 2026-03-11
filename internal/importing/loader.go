package importing

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/util"
)

// LoadRepository deterministically loads files from a local directory only.
func LoadRepository(path string) (LoadResult, error) {
	if strings.TrimSpace(path) == "" {
		return LoadResult{}, fmt.Errorf("import path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return LoadResult{}, fmt.Errorf("resolve path: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return LoadResult{}, fmt.Errorf("stat path: %w", err)
	}
	if !fi.IsDir() {
		return LoadResult{}, fmt.Errorf("local directory only: %s", abs)
	}

	files := []LoadedFile{}
	if err := filepath.WalkDir(abs, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == ".git" || name == ".atrakta" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(abs, p)
		if err != nil {
			return err
		}
		rel = util.NormalizeRelPath(rel)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		text := util.NormalizeContentLF(string(b))
		f := LoadedFile{
			RelPath:     rel,
			AbsPath:     p,
			ContentHash: util.SHA256Tagged([]byte(text)),
			ByteSize:    int64(len(b)),
			Binary:      isBinaryBlob(b),
			Executable:  isExecutableBlob(rel, b),
			SecretLike:  isSecretLikePath(rel),
			Content:     text,
		}
		files = append(files, f)
		return nil
	}); err != nil {
		return LoadResult{}, fmt.Errorf("walk import dir: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})
	batchID := deterministicBatchID(files)
	return LoadResult{
		SourceType:    "local_directory",
		SourcePath:    abs,
		ImportBatchID: batchID,
		Files:         files,
	}, nil
}

func deterministicBatchID(files []LoadedFile) string {
	parts := make([]string, 0, len(files))
	for _, f := range files {
		parts = append(parts, f.RelPath+":"+f.ContentHash)
	}
	sum := util.SHA256Hex([]byte(strings.Join(parts, "\n")))
	if len(sum) > 16 {
		sum = sum[:16]
	}
	return "import-" + sum
}

func isSecretLikePath(rel string) bool {
	lower := strings.ToLower(rel)
	if strings.Contains(lower, ".env") {
		return true
	}
	name := strings.ToLower(filepath.Base(rel))
	needles := []string{"secret", "secrets", "credential", "credentials", "token", "id_rsa", "id_dsa", "pem", "p12"}
	for _, n := range needles {
		if strings.Contains(name, n) {
			return true
		}
	}
	return false
}

func isBinaryBlob(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	max := len(b)
	if max > 1024 {
		max = 1024
	}
	for i := 0; i < max; i++ {
		if b[i] == 0 {
			return true
		}
	}
	return false
}

func isExecutableBlob(rel string, b []byte) bool {
	lower := strings.ToLower(rel)
	if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".dll") || strings.HasSuffix(lower, ".so") || strings.HasSuffix(lower, ".dylib") || strings.HasSuffix(lower, ".bat") || strings.HasSuffix(lower, ".ps1") {
		return true
	}
	if len(b) >= 2 && b[0] == '#' && b[1] == '!' {
		return true
	}
	return false
}
