package importing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/util"
)

func registryPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".atrakta", "capabilities", "registry.json")
}

func reportPath(repoRoot, batchID string) string {
	return filepath.Join(repoRoot, ".atrakta", "imports", batchID+".json")
}

func LoadRegistry(repoRoot string) (CapabilityRegistry, bool, error) {
	p := registryPath(repoRoot)
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CapabilityRegistry{V: 1, Entries: []CapabilityEntry{}}, false, nil
		}
		return CapabilityRegistry{}, false, fmt.Errorf("read capability registry: %w", err)
	}
	var r CapabilityRegistry
	if err := json.Unmarshal(b, &r); err != nil {
		return CapabilityRegistry{}, true, fmt.Errorf("parse capability registry: %w", err)
	}
	if r.V == 0 {
		r.V = 1
	}
	if r.V != 1 {
		return CapabilityRegistry{}, true, fmt.Errorf("capability registry v must be 1")
	}
	if r.Entries == nil {
		r.Entries = []CapabilityEntry{}
	}
	canonicalizeRegistry(&r)
	return r, true, nil
}

func SaveRegistry(repoRoot string, r CapabilityRegistry) error {
	if r.V == 0 {
		r.V = 1
	}
	if r.Entries == nil {
		r.Entries = []CapabilityEntry{}
	}
	canonicalizeRegistry(&r)
	p := registryPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("mkdir capability registry: %w", err)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal capability registry: %w", err)
	}
	b = append(b, '\n')
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "capability.registry.lock")
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(p, b, 0o644)
	})
}

func SaveImportReport(repoRoot string, rep ImportReport) error {
	if rep.V == 0 {
		rep.V = 1
	}
	if strings.TrimSpace(rep.ImportBatchID) == "" {
		return fmt.Errorf("import report requires import_batch_id")
	}
	p := reportPath(repoRoot, rep.ImportBatchID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("mkdir import reports: %w", err)
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal import report: %w", err)
	}
	b = append(b, '\n')
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "capability.imports.lock")
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(p, b, 0o644)
	})
}

func LoadImportReport(repoRoot, batchID string) (ImportReport, error) {
	p := reportPath(repoRoot, batchID)
	b, err := os.ReadFile(p)
	if err != nil {
		return ImportReport{}, fmt.Errorf("read import report: %w", err)
	}
	var rep ImportReport
	if err := json.Unmarshal(b, &rep); err != nil {
		return ImportReport{}, fmt.Errorf("parse import report: %w", err)
	}
	if rep.V == 0 {
		rep.V = 1
	}
	return rep, nil
}

func ListImportReports(repoRoot string) ([]ImportReport, error) {
	dir := filepath.Join(repoRoot, ".atrakta", "imports")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ImportReport{}, nil
		}
		return nil, fmt.Errorf("read imports dir: %w", err)
	}
	out := make([]ImportReport, 0, len(entries))
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		batchID := strings.TrimSuffix(ent.Name(), ".json")
		rep, err := LoadImportReport(repoRoot, batchID)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ImportedAt == out[j].ImportedAt {
			return out[i].ImportBatchID < out[j].ImportBatchID
		}
		return out[i].ImportedAt > out[j].ImportedAt
	})
	return out, nil
}

func canonicalizeRegistry(r *CapabilityRegistry) {
	for i := range r.Entries {
		r.Entries[i].Path = normalizePath(r.Entries[i].Path)
		if sp := strings.TrimSpace(r.Entries[i].SourcePath); sp != "" {
			r.Entries[i].SourcePath = filepath.ToSlash(filepath.Clean(sp))
		}
		r.Entries[i].Provenance.SourcePath = normalizePath(r.Entries[i].Provenance.SourcePath)
		if rp := strings.TrimSpace(r.Entries[i].Provenance.RootPath); rp != "" {
			r.Entries[i].Provenance.RootPath = filepath.ToSlash(filepath.Clean(rp))
		}
	}
	sort.SliceStable(r.Entries, func(i, j int) bool {
		if r.Entries[i].ID != r.Entries[j].ID {
			return r.Entries[i].ID < r.Entries[j].ID
		}
		if r.Entries[i].Path != r.Entries[j].Path {
			return r.Entries[i].Path < r.Entries[j].Path
		}
		return r.Entries[i].Kind < r.Entries[j].Kind
	})
}

func upsertEntries(reg *CapabilityRegistry, entries []CapabilityEntry) {
	idx := map[string]int{}
	for i, e := range reg.Entries {
		idx[e.ID] = i
	}
	for _, e := range entries {
		if i, ok := idx[e.ID]; ok {
			reg.Entries[i] = e
			continue
		}
		idx[e.ID] = len(reg.Entries)
		reg.Entries = append(reg.Entries, e)
	}
	canonicalizeRegistry(reg)
}

func findEntry(reg *CapabilityRegistry, id string) (*CapabilityEntry, int) {
	for i := range reg.Entries {
		if reg.Entries[i].ID == id {
			return &reg.Entries[i], i
		}
	}
	return nil, -1
}
