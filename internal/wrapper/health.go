package wrapper

import (
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/registry"
)

type HealthStatus struct {
	UserBinPath         string
	PathContainsUserBin bool
	PathPrefersUserBin  bool
	WrapperScriptCount  int
}

func Health() (HealthStatus, error) {
	binDir, err := userBinDir()
	if err != nil {
		return HealthStatus{}, err
	}
	status := HealthStatus{UserBinPath: binDir}
	status.PathContainsUserBin, status.PathPrefersUserBin = pathPosition(binDir, os.Getenv("PATH"))
	status.WrapperScriptCount = countWrapperScripts(binDir, registry.Default())
	return status, nil
}

func pathPosition(binDir, envPath string) (contains, prefers bool) {
	dirClean := filepath.Clean(binDir)
	parts := filepath.SplitList(envPath)
	for i, p := range parts {
		if filepath.Clean(strings.TrimSpace(p)) == dirClean {
			contains = true
			if i == 0 {
				prefers = true
			}
			return contains, prefers
		}
	}
	return false, false
}

func countWrapperScripts(binDir string, reg registry.Registry) int {
	n := 0
	seen := map[string]struct{}{}
	for _, id := range reg.InterfaceIDs() {
		e := reg.Entries[id]
		for _, name := range e.WrapperNames {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			p := filepath.Join(binDir, name)
			if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
				n++
			}
		}
	}
	return n
}
