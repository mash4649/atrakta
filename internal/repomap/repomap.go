package repomap

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"atrakta/internal/runtimecache"
	"atrakta/internal/util"
)

type Config struct {
	MaxTokens      int
	RefreshSeconds int
	Includes       []string
	Excludes       []string
}

type Report struct {
	Summary     string `json:"summary,omitempty"`
	UsedTokens  int    `json:"used_tokens"`
	FileCount   int    `json:"file_count"`
	Stamp       string `json:"stamp,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
	Refreshed   bool   `json:"refreshed"`
	Cached      bool   `json:"cached"`
}

type cachePayload struct {
	Summary     string `json:"summary,omitempty"`
	UsedTokens  int    `json:"used_tokens"`
	FileCount   int    `json:"file_count"`
	GeneratedAt string `json:"generated_at,omitempty"`
}

func LoadOrRefresh(repoRoot string, cfg Config) (Report, error) {
	cfg = normalizeConfig(cfg)
	now := time.Now().UTC()
	stamp := workspaceStamp(repoRoot, cfg)
	cfgHash := configHash(cfg)

	st, err := runtimecache.Load(repoRoot)
	if err == nil {
		if e, ok := st.Entries["repo_map"]; ok && runtimecache.IsFresh(e, stamp, cfgHash, now) {
			var p cachePayload
			if runtimecache.UnmarshalPayload(e.Payload, &p) == nil {
				return Report{
					Summary:     p.Summary,
					UsedTokens:  p.UsedTokens,
					FileCount:   p.FileCount,
					Stamp:       stamp,
					GeneratedAt: p.GeneratedAt,
					Cached:      true,
				}, nil
			}
		}
	}

	summary, used, total, err := buildSummary(repoRoot, cfg)
	if err != nil {
		return Report{}, err
	}
	payload := cachePayload{
		Summary:     summary,
		UsedTokens:  used,
		FileCount:   total,
		GeneratedAt: util.NowUTC(),
	}
	if err := runtimecache.Update(repoRoot, func(st *runtimecache.State) error {
		st.Entries["repo_map"] = runtimecache.Entry{
			UpdatedAt:  util.NowUTC(),
			Stamp:      stamp,
			ConfigHash: cfgHash,
			TTLSeconds: cfg.RefreshSeconds,
			Payload:    runtimecache.MarshalPayload(payload),
		}
		return nil
	}); err != nil {
		return Report{}, err
	}
	return Report{
		Summary:     payload.Summary,
		UsedTokens:  payload.UsedTokens,
		FileCount:   payload.FileCount,
		Stamp:       stamp,
		GeneratedAt: payload.GeneratedAt,
		Refreshed:   true,
	}, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 1200
	}
	if cfg.RefreshSeconds <= 0 {
		cfg.RefreshSeconds = 300
	}
	if len(cfg.Includes) == 0 {
		cfg.Includes = []string{""}
	}
	cfg.Excludes = append([]string{}, cfg.Excludes...)
	cfg.Excludes = append(cfg.Excludes, ".git/", ".atrakta/", ".tmp/")
	norm := make([]string, 0, len(cfg.Excludes))
	seen := map[string]struct{}{}
	for _, e := range cfg.Excludes {
		n := util.NormalizeRelPath(e)
		if n != "" && !strings.HasSuffix(n, "/") {
			n += "/"
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		norm = append(norm, n)
	}
	cfg.Excludes = norm
	return cfg
}

func buildSummary(repoRoot string, cfg Config) (string, int, int, error) {
	maxChars := cfg.MaxTokens * 4
	type fileRow struct {
		path string
		size int64
	}
	rows := make([]fileRow, 0, 256)
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		relRaw, rerr := filepath.Rel(repoRoot, path)
		if rerr != nil {
			return nil
		}
		rel := util.NormalizeRelPath(filepath.ToSlash(relRaw))
		if rel == "" {
			return nil
		}
		if isExcluded(rel, cfg.Excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !isIncluded(rel, cfg.Includes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fi, ferr := d.Info()
		if ferr != nil {
			return nil
		}
		rows = append(rows, fileRow{path: rel, size: fi.Size()})
		return nil
	})
	if err != nil {
		return "", 0, 0, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].path < rows[j].path })
	var b strings.Builder
	b.WriteString("# Repository Map\n")
	b.WriteString("format: path (bytes)\n")
	for _, r := range rows {
		line := fmt.Sprintf("%s (%d)\n", r.path, r.size)
		if b.Len()+len(line) > maxChars {
			break
		}
		b.WriteString(line)
	}
	s := b.String()
	usedTokens := len(s) / 4
	if usedTokens == 0 && len(s) > 0 {
		usedTokens = 1
	}
	return s, usedTokens, len(rows), nil
}

func workspaceStamp(repoRoot string, cfg Config) string {
	type row struct {
		path string
		mod  int64
		size int64
	}
	items := make([]row, 0, 256)
	_ = filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		relRaw, rerr := filepath.Rel(repoRoot, path)
		if rerr != nil {
			return nil
		}
		rel := util.NormalizeRelPath(filepath.ToSlash(relRaw))
		if rel == "" {
			return nil
		}
		if isExcluded(rel, cfg.Excludes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !isIncluded(rel, cfg.Includes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fi, ferr := d.Info()
		if ferr != nil {
			return nil
		}
		items = append(items, row{path: rel, mod: fi.ModTime().UnixNano(), size: fi.Size()})
		return nil
	})
	sort.Slice(items, func(i, j int) bool { return items[i].path < items[j].path })
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.path)
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(it.mod, 10))
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(it.size, 10))
		b.WriteByte('\n')
	}
	return util.SHA256Tagged([]byte(b.String()))
}

func isIncluded(rel string, includes []string) bool {
	if len(includes) == 0 {
		return true
	}
	for _, inc := range includes {
		n := util.NormalizeRelPath(inc)
		if n == "" {
			return true
		}
		if strings.HasPrefix(rel, n+"/") || rel == n {
			return true
		}
	}
	return false
}

func isExcluded(rel string, excludes []string) bool {
	for _, ex := range excludes {
		n := util.NormalizeRelPath(ex)
		if n != "" && !strings.HasSuffix(n, "/") {
			n += "/"
		}
		if n == "" {
			continue
		}
		if strings.HasPrefix(rel, n) || rel == strings.TrimSuffix(n, "/") {
			return true
		}
	}
	return false
}

func RemoveCache(repoRoot string) error {
	return runtimecache.Update(repoRoot, func(st *runtimecache.State) error {
		delete(st.Entries, "repo_map")
		return nil
	})
}

func configHash(cfg Config) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(cfg.MaxTokens))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(cfg.RefreshSeconds))
	b.WriteByte('|')
	for _, v := range cfg.Includes {
		b.WriteString(v)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for _, v := range cfg.Excludes {
		b.WriteString(v)
		b.WriteByte(',')
	}
	return util.SHA256Tagged([]byte(b.String()))
}
