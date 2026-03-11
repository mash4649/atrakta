package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/brownfield"
	"atrakta/internal/contract"
	"atrakta/internal/events"
	"atrakta/internal/projection"
	"atrakta/internal/registry"
	"atrakta/internal/util"
)

type IntegrationFinding struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	Interface  string `json:"interface,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	RepairHint string `json:"repair_hint,omitempty"`
}

type IntegrationReport struct {
	Outcome           string               `json:"outcome"`
	Reason            string               `json:"reason"`
	CheckedInterfaces []string             `json:"checked_interfaces,omitempty"`
	Detection         brownfield.Detection `json:"detection"`
	BlockingIssues    []IntegrationFinding `json:"blocking_issues"`
	Warnings          []IntegrationFinding `json:"warnings"`
	SuggestedCommands []string             `json:"suggested_commands,omitempty"`
}

func RunIntegration(repoRoot string) (IntegrationReport, error) {
	report := IntegrationReport{
		Outcome:        "PASS",
		Reason:         "integration healthy",
		BlockingIssues: []IntegrationFinding{},
		Warnings:       []IntegrationFinding{},
	}

	c, cb, err := contract.LoadOrInit(repoRoot)
	if err != nil {
		addIntegrationBlocking(&report, IntegrationFinding{
			Code:       "contract_invalid",
			Severity:   "blocking",
			Message:    "failed to load or validate contract",
			Actual:     err.Error(),
			RepairHint: "fix .atrakta/contract.json",
		})
		report.Reason = "integration blocked: contract invalid"
		report.SuggestedCommands = suggestIntegrationCommands(report)
		recordIntegrationEvent(repoRoot, report)
		return report, err
	}

	detection, err := brownfield.Detect(repoRoot)
	if err != nil {
		addIntegrationBlocking(&report, IntegrationFinding{
			Code:       "detection_failed",
			Severity:   "blocking",
			Message:    "brownfield detection failed",
			Actual:     err.Error(),
			RepairHint: "check repository and filesystem permissions",
		})
		report.Reason = "integration blocked: detection failed"
		report.SuggestedCommands = suggestIntegrationCommands(report)
		recordIntegrationEvent(repoRoot, report)
		return report, err
	}
	report.Detection = detection

	targets := integrationTargets(c)
	report.CheckedInterfaces = append(report.CheckedInterfaces, targets...)
	sourceAGENTS := readSourceAGENTS(repoRoot)
	reg := registry.ApplyOverrides(registry.Default(), c)
	desired, err := projection.RequiredForTargets(repoRoot, c, reg, targets, contract.ContractHash(cb), sourceAGENTS)
	if err != nil {
		addIntegrationBlocking(&report, IntegrationFinding{
			Code:       "projection_plan_failed",
			Severity:   "blocking",
			Message:    "failed to plan integration projections",
			Actual:     err.Error(),
			RepairHint: "fix contract projection settings and rerun doctor --integration",
		})
	}
	if len(desired) > 0 {
		conflicts, ferr := brownfield.FindConflicts(repoRoot, desired, true)
		if ferr != nil {
			addIntegrationBlocking(&report, IntegrationFinding{
				Code:       "conflict_detection_failed",
				Severity:   "blocking",
				Message:    "overwrite-risk detection failed",
				Actual:     ferr.Error(),
				RepairHint: "check repository filesystem permissions",
			})
		} else {
			for _, c := range conflicts {
				addIntegrationBlocking(&report, IntegrationFinding{
					Code:       "overwrite_risk",
					Severity:   "blocking",
					Message:    c.Reason,
					Path:       c.Path,
					Interface:  c.Interface,
					RepairHint: "run atrakta init --mode brownfield --no-overwrite to generate proposal patch",
				})
			}
		}
	}

	validateAppendIncludeSurface(repoRoot, c, &report)
	validateUnsupportedExtensions(c, &report)
	report.SuggestedCommands = suggestIntegrationCommands(report)
	if len(report.BlockingIssues) > 0 {
		report.Outcome = "BLOCKED"
		report.Reason = fmt.Sprintf("integration blocked: %d blocking issue(s), %d warning(s)", len(report.BlockingIssues), len(report.Warnings))
	} else if len(report.Warnings) > 0 {
		report.Outcome = "WARN"
		report.Reason = fmt.Sprintf("integration warnings: %d", len(report.Warnings))
	}
	recordIntegrationEvent(repoRoot, report)
	return report, nil
}

func (r IntegrationReport) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

func validateAppendIncludeSurface(repoRoot string, c contract.Contract, report *IntegrationReport) {
	mode := "append"
	appendFile := ".atrakta/AGENTS.append.md"
	if c.Extensions != nil && c.Extensions.Agents != nil {
		if strings.TrimSpace(c.Extensions.Agents.Mode) != "" {
			mode = strings.TrimSpace(strings.ToLower(c.Extensions.Agents.Mode))
		}
		if strings.TrimSpace(c.Extensions.Agents.AppendFile) != "" {
			appendFile = strings.TrimSpace(c.Extensions.Agents.AppendFile)
		}
	}
	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	agentsBody, _ := os.ReadFile(agentsPath)
	startCount := strings.Count(string(agentsBody), "<!-- ATRAKTA_MANAGED:START -->")
	endCount := strings.Count(string(agentsBody), "<!-- ATRAKTA_MANAGED:END -->")

	switch mode {
	case "append":
		if startCount != endCount {
			addIntegrationBlocking(report, IntegrationFinding{
				Code:       "append_failure",
				Severity:   "blocking",
				Message:    "managed append block markers are unbalanced",
				Path:       "AGENTS.md",
				Expected:   "balanced ATRAKTA_MANAGED markers",
				Actual:     fmt.Sprintf("start=%d end=%d", startCount, endCount),
				RepairHint: "run atrakta projection repair --all",
			})
		}
	case "include":
		ok, err := exists(filepath.Join(repoRoot, filepath.FromSlash(appendFile)))
		if err != nil {
			addIntegrationBlocking(report, IntegrationFinding{
				Code:       "include_check_failed",
				Severity:   "blocking",
				Message:    "failed to check include append file",
				Path:       appendFile,
				Actual:     err.Error(),
				RepairHint: "check repository filesystem permissions",
			})
			return
		}
		if !ok {
			addIntegrationWarning(report, IntegrationFinding{
				Code:       "include_missing",
				Severity:   "warning",
				Message:    "include mode append file is missing",
				Path:       appendFile,
				RepairHint: "run atrakta projection render --all",
			})
		}
	default:
		addIntegrationWarning(report, IntegrationFinding{
			Code:       "agents_mode_unknown",
			Severity:   "warning",
			Message:    "unknown agents mode in extensions.agents.mode",
			Actual:     mode,
			RepairHint: "set extensions.agents.mode to append|include|generate",
		})
	}
}

func validateUnsupportedExtensions(c contract.Contract, report *IntegrationReport) {
	if c.Extensions == nil {
		return
	}
	appendWarn := func(path string) {
		addIntegrationWarning(report, IntegrationFinding{
			Code:       "unsupported_extension_projection",
			Severity:   "warning",
			Message:    "extension is enabled but no native projection is available yet",
			Path:       path,
			RepairHint: "use fallback/link markdown projection or disable the extension until renderer is available",
		})
	}
	for _, e := range c.Extensions.MCP {
		if extensionEnabled(e.Enabled) {
			appendWarn("extensions.mcp." + e.ID)
		}
	}
	for _, e := range c.Extensions.Plugins {
		if extensionEnabled(e.Enabled) {
			appendWarn("extensions.plugins." + e.ID)
		}
	}
	for _, e := range c.Extensions.Skills {
		if extensionEnabled(e.Enabled) {
			appendWarn("extensions.skills." + e.ID)
		}
	}
	for _, e := range c.Extensions.Workflows {
		if extensionEnabled(e.Enabled) {
			appendWarn("extensions.workflows." + e.ID)
		}
	}
	if c.Extensions.Hooks != nil {
		if c.Extensions.Hooks.Shell != nil {
			if boolEnabled(c.Extensions.Hooks.Shell.OnCD) {
				appendWarn("extensions.hooks.shell.on_cd")
			}
			if boolEnabled(c.Extensions.Hooks.Shell.OnExec) {
				appendWarn("extensions.hooks.shell.on_exec")
			}
		}
		if c.Extensions.Hooks.Git != nil {
			if boolEnabled(c.Extensions.Hooks.Git.PreCommit) {
				appendWarn("extensions.hooks.git.pre_commit")
			}
			if boolEnabled(c.Extensions.Hooks.Git.PrePush) {
				appendWarn("extensions.hooks.git.pre_push")
			}
		}
		if c.Extensions.Hooks.IDE != nil {
			if boolEnabled(c.Extensions.Hooks.IDE.OnOpen) {
				appendWarn("extensions.hooks.ide.on_open")
			}
		}
		if c.Extensions.Hooks.Workflow != nil {
			if boolEnabled(c.Extensions.Hooks.Workflow.BeforeStart) {
				appendWarn("extensions.hooks.workflow.before_start")
			}
			if boolEnabled(c.Extensions.Hooks.Workflow.AfterApply) {
				appendWarn("extensions.hooks.workflow.after_apply")
			}
		}
	}
}

func integrationTargets(c contract.Contract) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	sup := contract.SupportedSet(c)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := sup[id]; !ok {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range c.Interfaces.CoreSet {
		add(id)
	}
	if len(out) == 0 {
		add("cursor")
	}
	sort.Strings(out)
	return out
}

func readSourceAGENTS(repoRoot string) string {
	b, err := os.ReadFile(filepath.Join(repoRoot, "AGENTS.md"))
	if err != nil {
		return ""
	}
	return util.NormalizeContentLF(string(b))
}

func extensionEnabled(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

func boolEnabled(v *bool) bool {
	return v != nil && *v
}

func suggestIntegrationCommands(report IntegrationReport) []string {
	set := map[string]struct{}{}
	add := func(cmd string) {
		if strings.TrimSpace(cmd) == "" {
			return
		}
		set[cmd] = struct{}{}
	}
	if len(report.BlockingIssues) > 0 || len(report.Warnings) > 0 {
		add("atrakta projection status --json")
	}
	for _, f := range report.BlockingIssues {
		switch f.Code {
		case "overwrite_risk":
			add("atrakta init --mode brownfield --no-overwrite")
			add("ls .atrakta/proposals/*.patch")
		case "append_failure":
			add("atrakta projection repair --all")
		}
	}
	for _, f := range report.Warnings {
		switch f.Code {
		case "include_missing":
			add("atrakta projection render --all")
		case "unsupported_extension_projection":
			add("atrakta doctor --parity --json")
		}
	}
	out := make([]string, 0, len(set))
	for cmd := range set {
		out = append(out, cmd)
	}
	sort.Strings(out)
	return out
}

func recordIntegrationEvent(repoRoot string, report IntegrationReport) {
	eventType := events.EventIntegrationChecked
	if report.Outcome == "BLOCKED" {
		eventType = events.EventIntegrationBlocked
	}
	_, _ = events.Append(repoRoot, eventType, "doctor", map[string]any{
		"outcome":            report.Outcome,
		"reason":             report.Reason,
		"blocking_count":     len(report.BlockingIssues),
		"warning_count":      len(report.Warnings),
		"checked_interfaces": report.CheckedInterfaces,
	})
}

func addIntegrationBlocking(report *IntegrationReport, f IntegrationFinding) {
	f.Severity = "blocking"
	report.BlockingIssues = append(report.BlockingIssues, f)
}

func addIntegrationWarning(report *IntegrationReport, f IntegrationFinding) {
	f.Severity = "warning"
	report.Warnings = append(report.Warnings, f)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
