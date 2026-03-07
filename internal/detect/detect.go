package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/registry"
	"atrakta/internal/state"
	"atrakta/internal/util"
)

type Input struct {
	RepoRoot          string
	Contract          contract.Contract
	Registry          registry.Registry
	State             state.State
	Explicit          []string
	StrongS1Validator func(path string, rec state.ManagedRecord) bool
}

func ParseExplicitFlag(flagValue string) []string {
	if strings.TrimSpace(flagValue) == "" {
		return nil
	}
	parts := strings.Split(flagValue, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func Run(in Input) (model.DetectResult, error) {
	supported := contract.SupportedSet(in.Contract)

	if len(in.Explicit) > 0 {
		for _, id := range in.Explicit {
			if _, ok := supported[id]; !ok {
				return model.DetectResult{}, fmt.Errorf("explicit interface %q unsupported", id)
			}
			if _, ok := in.Registry.Entries[id]; !ok {
				return model.DetectResult{}, fmt.Errorf("explicit interface %q disabled by registry overrides", id)
			}
		}
		return model.DetectResult{
			Signals:      map[string]any{"explicit": in.Explicit},
			TargetSet:    uniqSorted(in.Explicit),
			PruneAllowed: true,
			Reason:       model.ReasonExplicit,
		}, nil
	}

	strong := map[string]struct{}{}
	stateIfaces := map[string]struct{}{}
	invalidManaged := false
	for p, rec := range in.State.ManagedPaths {
		stateIfaces[rec.Interface] = struct{}{}
		ok := true
		if in.StrongS1Validator != nil {
			ok = in.StrongS1Validator(p, rec)
		}
		if ok {
			strong[rec.Interface] = struct{}{}
		} else {
			invalidManaged = true
		}
	}

	diskProjectionIfaces := map[string]struct{}{}
	for id, e := range in.Registry.Entries {
		if e.Anchor == "" {
			continue
		}
		anchor := filepath.Join(in.RepoRoot, filepath.FromSlash(util.NormalizeRelPath(e.Anchor)))
		if _, err := os.Stat(anchor); err == nil {
			strong[id] = struct{}{}
			diskProjectionIfaces[id] = struct{}{}
		}
	}

	if isMixed(stateIfaces, invalidManaged, diskProjectionIfaces) {
		fb, err := fallbackTarget(in.Contract)
		if err != nil {
			return model.DetectResult{}, err
		}
		return model.DetectResult{
			Signals:      map[string]any{"observed": toSorted(stateIfaces)},
			TargetSet:    fb,
			PruneAllowed: false,
			Reason:       model.ReasonMixed,
		}, nil
	}

	s := toSorted(strong)
	switch len(s) {
	case 0:
		fb, err := fallbackTarget(in.Contract)
		if err != nil {
			return model.DetectResult{}, err
		}
		return model.DetectResult{Signals: map[string]any{"observed": []string{}}, TargetSet: fb, PruneAllowed: false, Reason: model.ReasonUnknown}, nil
	case 1:
		return model.DetectResult{Signals: map[string]any{"observed": s}, TargetSet: s, PruneAllowed: in.Contract.Interfaces.PruneUnusedDefault, Reason: model.ReasonObservedExact}, nil
	default:
		fb, err := fallbackTarget(in.Contract)
		if err != nil {
			return model.DetectResult{}, err
		}
		return model.DetectResult{Signals: map[string]any{"observed": s}, TargetSet: fb, PruneAllowed: false, Reason: model.ReasonConflict}, nil
	}
}

func fallbackTarget(c contract.Contract) ([]string, error) {
	if c.Interfaces.Fallback == "core" {
		if len(c.Interfaces.CoreSet) == 0 {
			return nil, fmt.Errorf("fallback core_set empty")
		}
		return uniqSorted(c.Interfaces.CoreSet), nil
	}
	if c.Interfaces.Fallback == "all" {
		if len(c.Interfaces.Supported) == 0 {
			return nil, fmt.Errorf("fallback supported empty")
		}
		return uniqSorted(c.Interfaces.Supported), nil
	}
	if c.Interfaces.Fallback == "" {
		return nil, fmt.Errorf("fallback empty")
	}
	return []string{c.Interfaces.Fallback}, nil
}

func uniqSorted(in []string) []string {
	m := map[string]struct{}{}
	for _, s := range in {
		m[s] = struct{}{}
	}
	return toSorted(m)
}

func toSorted(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sameSet(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func isMixed(stateIfaces map[string]struct{}, invalidManaged bool, diskProjectionIfaces map[string]struct{}) bool {
	if len(stateIfaces) >= 2 && invalidManaged {
		return true
	}
	if len(diskProjectionIfaces) >= 2 && !sameSet(diskProjectionIfaces, stateIfaces) {
		return true
	}
	return false
}
