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

type directives struct {
	Prefer           []string
	Disable          []string
	ApprovalRequired []string
	HasApproval      bool
	QuickChecks      []string
	HasQuickChecks   bool
	HeavyChecks      []string
	HasHeavyChecks   bool
	PromptRequired   *bool
	PromptGoalLabel  *string
	PlanFormat       *string
	ErrorFormat      *string
	HookShellOnCD    *bool
	HookIDEOnOpen    *bool
	DeniedDirectives []model.SyncFieldDiff
}

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

	d := parseDirectives(agentsText)
	d.Prefer = filterSupported(d.Prefer, c.Interfaces.Supported)
	d.Disable = filterSupported(d.Disable, c.Interfaces.Supported)

	allowed := make([]model.SyncFieldDiff, 0)
	denied := append([]model.SyncFieldDiff{}, d.DeniedDirectives...)
	changed := false

	addAllowed := func(field string, from, to any, reason string) {
		allowed = append(allowed, model.SyncFieldDiff{
			Field:  field,
			Status: "allowed",
			From:   from,
			To:     to,
			Reason: reason,
		})
		changed = true
	}

	if !sameStringSet(d.Prefer, proposed.Hints.Prefer) {
		before := append([]string{}, proposed.Hints.Prefer...)
		proposed.Hints.Prefer = append([]string{}, d.Prefer...)
		addAllowed("hints.prefer", before, proposed.Hints.Prefer, "allowlisted reverse-sync field")
	}
	if !sameStringSet(d.Disable, proposed.Hints.DisableInterfaces) {
		before := append([]string{}, proposed.Hints.DisableInterfaces...)
		proposed.Hints.DisableInterfaces = append([]string{}, d.Disable...)
		addAllowed("hints.disable_interfaces", before, proposed.Hints.DisableInterfaces, "allowlisted reverse-sync field")
	}
	if d.HasApproval && !sameStringSet(d.ApprovalRequired, proposed.Tools.ApprovalRequiredFor) {
		before := append([]string{}, proposed.Tools.ApprovalRequiredFor...)
		proposed.Tools.ApprovalRequiredFor = append([]string{}, d.ApprovalRequired...)
		addAllowed("tools.approval_required_for", before, proposed.Tools.ApprovalRequiredFor, "allowlisted reverse-sync field")
	}

	if d.HasQuickChecks || d.HasHeavyChecks {
		if proposed.Quality == nil {
			def := contract.Default(".")
			proposed.Quality = def.Quality
		}
		if d.HasQuickChecks && !sameStringSet(d.QuickChecks, proposed.Quality.QuickChecks) {
			before := append([]string{}, proposed.Quality.QuickChecks...)
			proposed.Quality.QuickChecks = append([]string{}, d.QuickChecks...)
			addAllowed("quality.quick_checks", before, proposed.Quality.QuickChecks, "allowlisted reverse-sync field")
		}
		if d.HasHeavyChecks && !sameStringSet(d.HeavyChecks, proposed.Quality.HeavyChecks) {
			before := append([]string{}, proposed.Quality.HeavyChecks...)
			proposed.Quality.HeavyChecks = append([]string{}, d.HeavyChecks...)
			addAllowed("quality.heavy_checks", before, proposed.Quality.HeavyChecks, "allowlisted reverse-sync field")
		}
	}

	if d.PromptRequired != nil || d.PromptGoalLabel != nil {
		if proposed.Policies == nil {
			proposed.Policies = &contract.Policies{}
		}
		if proposed.Policies.PromptMin == nil {
			def := contract.Default(".")
			proposed.Policies.PromptMin = def.Policies.PromptMin
		}
		if strings.TrimSpace(proposed.Policies.PromptMin.Ref) == "" {
			proposed.Policies.PromptMin.Ref = ".atrakta/policies/prompt-min.json"
		}
		if strings.TrimSpace(proposed.Policies.PromptMin.Apply) == "" {
			proposed.Policies.PromptMin.Apply = "conditional"
		}
		if d.PromptRequired != nil && proposed.Policies.PromptMin.Required != *d.PromptRequired {
			before := proposed.Policies.PromptMin.Required
			proposed.Policies.PromptMin.Required = *d.PromptRequired
			addAllowed("policies.prompt_min.required", before, proposed.Policies.PromptMin.Required, "allowlisted reverse-sync field")
		}
		if d.PromptGoalLabel != nil && proposed.Policies.PromptMin.GoalLabel != *d.PromptGoalLabel {
			before := proposed.Policies.PromptMin.GoalLabel
			proposed.Policies.PromptMin.GoalLabel = *d.PromptGoalLabel
			addAllowed("policies.prompt_min.goal_label", before, proposed.Policies.PromptMin.GoalLabel, "allowlisted reverse-sync field")
		}
	}

	if d.PlanFormat != nil || d.ErrorFormat != nil {
		if proposed.Parity == nil {
			def := contract.Default(".")
			proposed.Parity = def.Parity
		}
		if d.PlanFormat != nil && proposed.Parity.OutputSurface.PlanFormat != *d.PlanFormat {
			before := proposed.Parity.OutputSurface.PlanFormat
			proposed.Parity.OutputSurface.PlanFormat = *d.PlanFormat
			addAllowed("parity.output_surface.plan_format", before, proposed.Parity.OutputSurface.PlanFormat, "allowlisted reverse-sync field")
		}
		if d.ErrorFormat != nil && proposed.Parity.OutputSurface.ErrorFormat != *d.ErrorFormat {
			before := proposed.Parity.OutputSurface.ErrorFormat
			proposed.Parity.OutputSurface.ErrorFormat = *d.ErrorFormat
			addAllowed("parity.output_surface.error_format", before, proposed.Parity.OutputSurface.ErrorFormat, "allowlisted reverse-sync field")
		}
	}

	if d.HookShellOnCD != nil || d.HookIDEOnOpen != nil {
		if proposed.Extensions == nil {
			def := contract.Default(".")
			proposed.Extensions = def.Extensions
		}
		if proposed.Extensions.Hooks == nil {
			proposed.Extensions.Hooks = &contract.HooksExtension{}
		}
		if d.HookShellOnCD != nil {
			if proposed.Extensions.Hooks.Shell == nil {
				proposed.Extensions.Hooks.Shell = &contract.ShellHooks{}
			}
			before := boolPtrValue(proposed.Extensions.Hooks.Shell.OnCD)
			after := *d.HookShellOnCD
			if before != after {
				proposed.Extensions.Hooks.Shell.OnCD = boolPtr(after)
				addAllowed("extensions.hooks.shell.on_cd", before, after, "allowlisted reverse-sync field")
			}
		}
		if d.HookIDEOnOpen != nil {
			if proposed.Extensions.Hooks.IDE == nil {
				proposed.Extensions.Hooks.IDE = &contract.IDEHooks{}
			}
			before := boolPtrValue(proposed.Extensions.Hooks.IDE.OnOpen)
			after := *d.HookIDEOnOpen
			if before != after {
				proposed.Extensions.Hooks.IDE.OnOpen = boolPtr(after)
				addAllowed("extensions.hooks.ide.on_open", before, after, "allowlisted reverse-sync field")
			}
		}
	}

	if err := contract.Validate(proposed); err != nil {
		return model.SyncProposal{}, c, fmt.Errorf("proposal invalid: %w", err)
	}

	summary := "no changes"
	switch {
	case changed && len(denied) > 0:
		summary = fmt.Sprintf("%d allowlisted field update(s), %d denied field(s)", len(allowed), len(denied))
	case changed:
		summary = fmt.Sprintf("%d allowlisted field update(s)", len(allowed))
	case len(denied) > 0:
		summary = fmt.Sprintf("no applicable updates; %d field(s) denied", len(denied))
	}

	return model.SyncProposal{
		Needed:           changed,
		Prefer:           proposed.Hints.Prefer,
		Disable:          proposed.Hints.DisableInterfaces,
		Allowed:          allowed,
		Denied:           denied,
		RequiresApproval: changed,
		Summary:          summary,
	}, proposed, nil
}

func parseDirectives(text string) directives {
	out := directives{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "sync.") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "sync.prefer_interfaces":
			out.Prefer = parseCSV(val)
		case "sync.disable_interfaces":
			out.Disable = parseCSV(val)
		case "sync.approval_required_for":
			out.ApprovalRequired = parseCSV(val)
			out.HasApproval = true
		case "sync.quick_checks":
			out.QuickChecks = parseCSV(val)
			out.HasQuickChecks = true
		case "sync.heavy_checks":
			out.HeavyChecks = parseCSV(val)
			out.HasHeavyChecks = true
		case "sync.prompt_min.required":
			if b, ok := parseBool(val); ok {
				out.PromptRequired = boolPtr(b)
			} else {
				out.DeniedDirectives = append(out.DeniedDirectives, deniedDirective(key, val, "invalid boolean value"))
			}
		case "sync.prompt_min.goal_label":
			s := strings.TrimSpace(val)
			out.PromptGoalLabel = &s
		case "sync.parity.output.plan_format":
			s := strings.TrimSpace(strings.ToLower(val))
			out.PlanFormat = &s
		case "sync.parity.output.error_format":
			s := strings.TrimSpace(strings.ToLower(val))
			out.ErrorFormat = &s
		case "sync.extensions.hooks.shell.on_cd":
			if b, ok := parseBool(val); ok {
				out.HookShellOnCD = boolPtr(b)
			} else {
				out.DeniedDirectives = append(out.DeniedDirectives, deniedDirective(key, val, "invalid boolean value"))
			}
		case "sync.extensions.hooks.ide.on_open":
			if b, ok := parseBool(val); ok {
				out.HookIDEOnOpen = boolPtr(b)
			} else {
				out.DeniedDirectives = append(out.DeniedDirectives, deniedDirective(key, val, "invalid boolean value"))
			}
		default:
			reason := "field is not allowlisted for reverse sync"
			if isProtectedDirective(key) {
				reason = "protected field; reverse sync is proposal-only and denied"
			}
			out.DeniedDirectives = append(out.DeniedDirectives, deniedDirective(key, val, reason))
		}
	}
	out.Prefer = uniqSorted(out.Prefer)
	out.Disable = uniqSorted(out.Disable)
	out.ApprovalRequired = uniqSorted(out.ApprovalRequired)
	out.QuickChecks = uniqSorted(out.QuickChecks)
	out.HeavyChecks = uniqSorted(out.HeavyChecks)
	sort.SliceStable(out.DeniedDirectives, func(i, j int) bool {
		return out.DeniedDirectives[i].Field < out.DeniedDirectives[j].Field
	})
	return out
}

func deniedDirective(key, value, reason string) model.SyncFieldDiff {
	return model.SyncFieldDiff{
		Field:  strings.TrimPrefix(key, "sync."),
		Status: "denied",
		To:     strings.TrimSpace(value),
		Reason: reason,
	}
}

func isProtectedDirective(key string) bool {
	for _, p := range []string{
		"sync.extensions.mcp",
		"sync.extensions.plugins",
		"sync.extensions.skills",
		"sync.extensions.workflows",
	} {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func parseBool(raw string) (bool, bool) {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
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

func boolPtr(v bool) *bool { return &v }

func boolPtrValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
