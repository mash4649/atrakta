package proof

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/state"
	"atrakta/internal/util"
)

type Expected struct {
	Fingerprint string
	Target      string
	TemplateID  string
	SourceText  string
}

func Revalidate(repoRoot, relPath string, rec state.ManagedRecord, exp Expected) error {
	if rec.Fingerprint == "" || rec.TemplateID == "" {
		return fmt.Errorf("managed record missing fingerprint/template_id")
	}
	if rec.Fingerprint != exp.Fingerprint {
		return fmt.Errorf("fingerprint mismatch")
	}
	if rec.TemplateID != exp.TemplateID {
		return fmt.Errorf("template_id mismatch")
	}
	abs := filepath.Join(repoRoot, filepath.FromSlash(relPath))
	fi, err := os.Lstat(abs)
	if err != nil {
		return fmt.Errorf("lstat: %w", err)
	}
	switch rec.Kind {
	case "link", "junction":
		if fi.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("expected symlink kind=%s", rec.Kind)
		}
		got, err := os.Readlink(abs)
		if err != nil {
			return fmt.Errorf("readlink: %w", err)
		}
		gotAbs := got
		if !filepath.IsAbs(gotAbs) {
			gotAbs = filepath.Join(filepath.Dir(abs), gotAbs)
		}
		expectedAbs := filepath.Join(repoRoot, filepath.FromSlash(exp.Target))
		if filepath.Clean(gotAbs) != filepath.Clean(expectedAbs) {
			return fmt.Errorf("symlink target mismatch")
		}
	case "copy":
		b, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("read copy: %w", err)
		}
		text := util.NormalizeContentLF(string(b))
		headerOK := strings.Contains(text, "# template_id: "+exp.TemplateID) && strings.Contains(text, "# fingerprint: "+exp.Fingerprint)
		if headerOK {
			return nil
		}
		trimmed := strings.TrimSpace(text)
		src := strings.TrimSpace(util.NormalizeContentLF(exp.SourceText))
		if trimmed != src {
			return fmt.Errorf("copy content mismatch")
		}
	default:
		return fmt.Errorf("unsupported kind %q", rec.Kind)
	}
	return nil
}

func IsManagedDestructiveAllowed(repoRoot, relPath string, managed map[string]state.ManagedRecord, exp Expected) error {
	rec, ok := managed[relPath]
	if !ok {
		return fmt.Errorf("path is not managed")
	}
	return Revalidate(repoRoot, relPath, rec, exp)
}
