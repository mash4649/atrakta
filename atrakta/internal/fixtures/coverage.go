package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type resolverFixtureMap struct {
	Entries []resolverFixtureEntry `json:"entries"`
}

type resolverFixtureEntry struct {
	Resolver string   `json:"resolver"`
	Fixtures []string `json:"fixtures"`
}

// VerifyResolverFixtureCoverage ensures each resolver has declared fixture coverage.
func VerifyResolverFixtureCoverage(projectRoot string) error {
	root := projectRoot
	if root == "" {
		var err error
		root, err = resolveProjectRoot()
		if err != nil {
			return err
		}
	}

	resolvers, err := discoverResolvers(root)
	if err != nil {
		return err
	}

	mapPath := filepath.Join(root, "fixtures", "resolver-fixture-map.json")
	b, err := os.ReadFile(mapPath)
	if err != nil {
		return fmt.Errorf("read resolver fixture map: %w", err)
	}

	var m resolverFixtureMap
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("parse resolver fixture map: %w", err)
	}

	declared := map[string]resolverFixtureEntry{}
	for _, e := range m.Entries {
		declared[e.Resolver] = e
	}

	errs := make([]string, 0)

	for resolver := range resolvers {
		entry, ok := declared[resolver]
		if !ok {
			errs = append(errs, fmt.Sprintf("resolver missing fixture mapping: %s", resolver))
			continue
		}
		if len(entry.Fixtures) == 0 {
			errs = append(errs, fmt.Sprintf("resolver fixture mapping empty: %s", resolver))
			continue
		}
		for _, fp := range entry.Fixtures {
			if !strings.HasPrefix(fp, "fixtures/") {
				errs = append(errs, fmt.Sprintf("fixture path must start with fixtures/: %s -> %s", resolver, fp))
				continue
			}
			abs := filepath.Join(root, filepath.FromSlash(fp))
			if _, err := os.Stat(abs); err != nil {
				errs = append(errs, fmt.Sprintf("fixture file missing: %s -> %s", resolver, fp))
			}
		}
	}

	for resolver := range declared {
		if _, ok := resolvers[resolver]; !ok {
			errs = append(errs, fmt.Sprintf("fixture mapping has unknown resolver: %s", resolver))
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("resolver fixture coverage check failed: %s", joinErrors(errs))
	}
	return nil
}

func discoverResolvers(root string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	base := filepath.Join(root, "resolvers")
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "resolver.go" {
			return nil
		}
		rel, err := filepath.Rel(base, filepath.Dir(path))
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "common") {
			return nil
		}
		parts := strings.Split(rel, "/")
		if len(parts) != 2 {
			return nil
		}
		out[rel] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover resolvers: %w", err)
	}
	return out, nil
}

func resolveProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("project root with go.mod not found")
}

func joinErrors(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	out := errs[0]
	for i := 1; i < len(errs); i++ {
		out += "; " + errs[i]
	}
	return out
}
