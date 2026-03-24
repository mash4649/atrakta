package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type operationsCoveragePolicy struct {
	RequiredTargets map[string]string `json:"required_targets"`
	AllowedUnbound  []string          `json:"allowed_unbound"`
}

// VerifyOperationsSchemaCoverage ensures every operations schema file is either
// a required target or explicitly allowed as unbound.
func VerifyOperationsSchemaCoverage(projectRoot string) error {
	root := projectRoot
	if root == "" {
		var err error
		root, err = resolveProjectRoot()
		if err != nil {
			return err
		}
	}

	policyPath := filepath.Join(root, "schemas", "operations", "coverage-policy.json")
	b, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("read operations coverage policy: %w", err)
	}

	var policy operationsCoveragePolicy
	if err := json.Unmarshal(b, &policy); err != nil {
		return fmt.Errorf("parse operations coverage policy: %w", err)
	}

	covered := map[string]struct{}{}
	for _, p := range policy.RequiredTargets {
		covered[p] = struct{}{}
	}
	for _, p := range policy.AllowedUnbound {
		covered[p] = struct{}{}
	}

	entries, err := os.ReadDir(filepath.Join(root, "schemas", "operations"))
	if err != nil {
		return fmt.Errorf("read operations schema directory: %w", err)
	}

	all := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if filepath.Ext(strings.TrimSuffix(e.Name(), ".json")) != ".schema" {
			continue
		}
		all = append(all, filepath.ToSlash(filepath.Join("schemas/operations", e.Name())))
	}
	sort.Strings(all)

	errs := make([]string, 0)

	for name, target := range policy.RequiredTargets {
		targetAbs := filepath.Join(root, filepath.FromSlash(target))
		if _, err := os.Stat(targetAbs); err != nil {
			errs = append(errs, fmt.Sprintf("required target %q (%s) missing", name, target))
		}
	}

	for _, path := range all {
		if _, ok := covered[path]; !ok {
			errs = append(errs, fmt.Sprintf("uncovered operations schema: %s", path))
		}
	}

	for path := range covered {
		pathAbs := filepath.Join(root, filepath.FromSlash(path))
		if _, err := os.Stat(pathAbs); err != nil {
			errs = append(errs, fmt.Sprintf("coverage policy points to missing schema: %s", path))
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("operations schema coverage check failed: %s", joinErrors(errs))
	}
	return nil
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
