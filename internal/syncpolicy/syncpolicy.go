package syncpolicy

import (
	"bufio"
	"fmt"
	"sort"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/model"
)

type Level int

const (
	Level0 Level = 0
	Level1 Level = 1
	Level2 Level = 2
)

func ParseLevel(raw string) Level {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(strings.ToLower(s), "level")
	s = strings.TrimPrefix(s, "=")
	s = strings.TrimSpace(s)
	switch s {
	case "1":
		return Level1
	case "2":
		return Level2
	default:
		return Level0
	}
}

func ProposeFromAGENTS(c contract.Contract, agentsText string) (model.SyncProposal, contract.Contract, error) {
	proposed := c
	if proposed.Hints == nil {
		proposed.Hints = &contract.Hints{}
	}
	prefer, disable := parseDirectives(agentsText)
	prefer = filterSupported(prefer, c.Interfaces.Supported)
	disable = filterSupported(disable, c.Interfaces.Supported)

	changed := false
	if !sameStringSet(prefer, proposed.Hints.Prefer) {
		proposed.Hints.Prefer = prefer
		changed = true
	}
	if !sameStringSet(disable, proposed.Hints.DisableInterfaces) {
		proposed.Hints.DisableInterfaces = disable
		changed = true
	}
	if err := contract.Validate(proposed); err != nil {
		return model.SyncProposal{}, c, fmt.Errorf("proposal invalid: %w", err)
	}

	summary := "no changes"
	if changed {
		summary = "contract hints proposal available"
	}
	return model.SyncProposal{
		Needed:           changed,
		Prefer:           prefer,
		Disable:          disable,
		RequiresApproval: changed,
		Summary:          summary,
	}, proposed, nil
}

func parseDirectives(text string) (prefer []string, disable []string) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "sync.prefer_interfaces:") {
			prefer = parseCSV(strings.TrimSpace(strings.TrimPrefix(line, "sync.prefer_interfaces:")))
		}
		if strings.HasPrefix(line, "sync.disable_interfaces:") {
			disable = parseCSV(strings.TrimSpace(strings.TrimPrefix(line, "sync.disable_interfaces:")))
		}
	}
	prefer = uniqSorted(prefer)
	disable = uniqSorted(disable)
	return prefer, disable
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func uniqSorted(in []string) []string {
	m := map[string]struct{}{}
	for _, s := range in {
		m[s] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func filterSupported(in []string, supported []string) []string {
	allow := map[string]struct{}{}
	for _, s := range supported {
		allow[s] = struct{}{}
	}
	out := []string{}
	for _, s := range in {
		if _, ok := allow[s]; ok {
			out = append(out, s)
		}
	}
	return uniqSorted(out)
}

func sameStringSet(a, b []string) bool {
	a = uniqSorted(a)
	b = uniqSorted(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
