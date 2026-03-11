package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/model"
	"atrakta/internal/util"
)

const (
	projectionManifestPath = ".atrakta/projections/manifest.json"
	extensionManifestPath  = ".atrakta/extensions/manifest.json"
)

type UpdateResult struct {
	ProjectionEntries int
	ExtensionEntries  int
	SourceHash        string
	RenderHash        string
}

func UpdateFromApply(repoRoot string, ap model.ApplyResult, sourceHash string) (UpdateResult, error) {
	now := util.NowUTC()

	pm, err := loadProjectionManifest(repoRoot)
	if err != nil {
		return UpdateResult{}, err
	}

	entryByKey := make(map[string]model.ProjectionManifestEntry, len(pm.Entries))
	for _, e := range pm.Entries {
		if len(e.Files) == 0 {
			continue
		}
		k := projectionKey(e.Interface, e.Kind, e.Files[0])
		entryByKey[k] = e
	}

	for _, op := range ap.Ops {
		if strings.TrimSpace(op.Interface) == "" || strings.TrimSpace(op.Path) == "" {
			continue
		}
		kind := normalizeKind(op.Kind, op.Op)
		k := projectionKey(op.Interface, kind, op.Path)
		switch op.Op {
		case "delete", "unlink":
			delete(entryByKey, k)
		default:
			if op.Status != "ok" && op.Status != "skipped" {
				continue
			}
			entryByKey[k] = model.ProjectionManifestEntry{
				Interface:  op.Interface,
				Kind:       kind,
				Files:      []string{op.Path},
				SourceHash: sourceHash,
				RenderHash: op.Fingerprint,
				Status:     op.Status,
				UpdatedAt:  now,
			}
		}
	}

	pm.V = 1
	pm.Entries = make([]model.ProjectionManifestEntry, 0, len(entryByKey))
	for _, e := range entryByKey {
		pm.Entries = append(pm.Entries, e)
	}
	sort.Slice(pm.Entries, func(i, j int) bool {
		ai := pm.Entries[i]
		aj := pm.Entries[j]
		if ai.Interface != aj.Interface {
			return ai.Interface < aj.Interface
		}
		if ai.Kind != aj.Kind {
			return ai.Kind < aj.Kind
		}
		af := ""
		jf := ""
		if len(ai.Files) > 0 {
			af = ai.Files[0]
		}
		if len(aj.Files) > 0 {
			jf = aj.Files[0]
		}
		return af < jf
	})
	if err := saveProjectionManifest(repoRoot, pm); err != nil {
		return UpdateResult{}, err
	}

	em, err := loadExtensionManifest(repoRoot)
	if err != nil {
		return UpdateResult{}, err
	}
	em.V = 1
	if err := saveExtensionManifest(repoRoot, em); err != nil {
		return UpdateResult{}, err
	}

	renderHash, err := manifestHash(pm)
	if err != nil {
		return UpdateResult{}, err
	}

	return UpdateResult{
		ProjectionEntries: len(pm.Entries),
		ExtensionEntries:  len(em.Entries),
		SourceHash:        sourceHash,
		RenderHash:        renderHash,
	}, nil
}

func projectionKey(iface, kind, path string) string {
	return iface + "|" + kind + "|" + util.NormalizeRelPath(path)
}

func normalizeKind(kind, op string) string {
	k := strings.TrimSpace(kind)
	if k != "" {
		return k
	}
	switch op {
	case "link", "unlink":
		return "link"
	default:
		return "copy"
	}
}

func loadProjectionManifest(repoRoot string) (model.ProjectionManifest, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(projectionManifestPath))
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.ProjectionManifest{V: 1, Entries: []model.ProjectionManifestEntry{}}, nil
		}
		return model.ProjectionManifest{}, fmt.Errorf("read projection manifest: %w", err)
	}
	var m model.ProjectionManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return model.ProjectionManifest{}, fmt.Errorf("parse projection manifest: %w", err)
	}
	if m.Entries == nil {
		m.Entries = []model.ProjectionManifestEntry{}
	}
	if m.V == 0 {
		m.V = 1
	}
	return m, nil
}

func saveProjectionManifest(repoRoot string, m model.ProjectionManifest) error {
	path := filepath.Join(repoRoot, filepath.FromSlash(projectionManifestPath))
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "projections.manifest.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir projection manifest dir: %w", err)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal projection manifest: %w", err)
	}
	b = append(b, '\n')
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(path, b, 0o644)
	})
}

func loadExtensionManifest(repoRoot string) (model.ExtensionManifest, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(extensionManifestPath))
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.ExtensionManifest{V: 1, Entries: []model.ExtensionManifestEntry{}}, nil
		}
		return model.ExtensionManifest{}, fmt.Errorf("read extension manifest: %w", err)
	}
	var m model.ExtensionManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return model.ExtensionManifest{}, fmt.Errorf("parse extension manifest: %w", err)
	}
	if m.Entries == nil {
		m.Entries = []model.ExtensionManifestEntry{}
	}
	if m.V == 0 {
		m.V = 1
	}
	return m, nil
}

func saveExtensionManifest(repoRoot string, m model.ExtensionManifest) error {
	path := filepath.Join(repoRoot, filepath.FromSlash(extensionManifestPath))
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "extensions.manifest.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir extension manifest dir: %w", err)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal extension manifest: %w", err)
	}
	b = append(b, '\n')
	return util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(path, b, 0o644)
	})
}

func manifestHash(pm model.ProjectionManifest) (string, error) {
	b, err := util.MarshalCanonical(pm)
	if err != nil {
		return "", fmt.Errorf("canonical manifest hash: %w", err)
	}
	return util.SHA256Tagged(b), nil
}
