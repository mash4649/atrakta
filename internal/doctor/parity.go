package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/bootstrap"
	agentsctx "atrakta/internal/context"
	"atrakta/internal/contract"
	"atrakta/internal/manifest"
	"atrakta/internal/model"
	"atrakta/internal/proof"
	"atrakta/internal/state"
	"atrakta/internal/util"
)

type ParityFinding struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"` // blocking|warning
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	Interface  string `json:"interface,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	RepairHint string `json:"repair_hint,omitempty"`
}

type ParityReport struct {
	Outcome           string          `json:"outcome"` // PASS|WARN|BLOCKED
	Reason            string          `json:"reason"`
	CheckedAt         string          `json:"checked_at"`
	BlockingIssues    []ParityFinding `json:"blocking_issues"`
	Warnings          []ParityFinding `json:"warnings"`
	SuggestedCommands []string        `json:"suggested_commands,omitempty"`
}

func RunParity(repoRoot string) (ParityReport, error) {
	now := util.NowUTC()
	report := ParityReport{
		Outcome:        "PASS",
		Reason:         "parity healthy",
		CheckedAt:      now,
		BlockingIssues: []ParityFinding{},
		Warnings:       []ParityFinding{},
	}

	c, _, err := contract.LoadOrInit(repoRoot)
	if err != nil {
		report.Outcome = "BLOCKED"
		report.Reason = "contract validation failed"
		report.BlockingIssues = append(report.BlockingIssues, ParityFinding{
			Code:       "contract_invalid",
			Severity:   "blocking",
			Message:    "failed to load/validate contract",
			Actual:     err.Error(),
			RepairHint: "fix .atrakta/contract.json and rerun doctor --parity",
		})
		report.SuggestedCommands = append(report.SuggestedCommands, "atrakta doctor")
		return report, err
	}
	c = contract.CanonicalizeBoundary(c)

	_, _, _ = bootstrap.EnsureRootAGENTS(repoRoot)
	sourceAGENTS, _, ctxErr := agentsctx.Resolve(agentsctx.ResolveInput{
		RepoRoot: repoRoot,
		StartDir: repoRoot,
		Config:   c.Context,
	})
	if ctxErr != nil {
		addParityBlocking(&report, ParityFinding{
			Code:       "context_resolve_failed",
			Severity:   "blocking",
			Message:    "failed to resolve AGENTS context",
			Actual:     ctxErr.Error(),
			RepairHint: "fix AGENTS import chain and rerun doctor --parity",
		})
	}

	st, _, stErr := state.LoadOrEmpty(repoRoot, "")
	if stErr != nil {
		addParityBlocking(&report, ParityFinding{
			Code:       "state_invalid",
			Severity:   "blocking",
			Message:    "failed to parse state.json",
			Actual:     stErr.Error(),
			RepairHint: "run atrakta doctor to rebuild state",
		})
	}
	ms, msErr := manifest.ReadStatus(repoRoot)
	if msErr != nil {
		addParityBlocking(&report, ParityFinding{
			Code:       "manifest_invalid",
			Severity:   "blocking",
			Message:    "failed to parse manifest files",
			Actual:     msErr.Error(),
			RepairHint: "run atrakta projection repair --all",
		})
	} else {
		if !ms.ProjectionExists {
			addParityBlocking(&report, ParityFinding{
				Code:       "manifest_missing",
				Severity:   "blocking",
				Message:    "projection manifest is missing",
				Path:       ms.ProjectionPath,
				RepairHint: "run atrakta projection render --all",
			})
		}
		if !ms.ExtensionExists {
			addParityWarning(&report, ParityFinding{
				Code:       "extension_manifest_missing",
				Severity:   "warning",
				Message:    "extension manifest is missing",
				Path:       ms.ExtensionPath,
				RepairHint: "run atrakta projection render --all",
			})
		}
		validateProjectionManifestFiles(repoRoot, ms, &report)
		validateExtensionManifestFiles(repoRoot, ms, &report)
		validateManifestHashes(st, ms, &report)
	}

	validateManagedParity(repoRoot, st, sourceAGENTS, &report)
	validateApprovalSurface(c, &report)
	validateOutputSurface(c, &report)
	validateNonInteractiveSurface(&report)

	report.SuggestedCommands = suggestParityCommands(report)
	switch {
	case len(report.BlockingIssues) > 0:
		report.Outcome = "BLOCKED"
		report.Reason = fmt.Sprintf("parity drift detected: %d blocking issue(s), %d warning(s)", len(report.BlockingIssues), len(report.Warnings))
	case len(report.Warnings) > 0:
		report.Outcome = "WARN"
		report.Reason = fmt.Sprintf("parity warning(s): %d", len(report.Warnings))
	default:
		report.Outcome = "PASS"
		report.Reason = "parity healthy"
	}
	return report, nil
}

func validateProjectionManifestFiles(repoRoot string, ms manifest.Status, report *ParityReport) {
	for _, e := range ms.Projection.Entries {
		if strings.TrimSpace(e.Interface) == "" {
			addParityWarning(report, ParityFinding{
				Code:       "manifest_entry_invalid",
				Severity:   "warning",
				Message:    "projection entry has empty interface",
				RepairHint: "run atrakta projection repair --all",
			})
		}
		if len(e.Files) == 0 {
			addParityWarning(report, ParityFinding{
				Code:       "manifest_entry_invalid",
				Severity:   "warning",
				Message:    "projection entry has no files",
				Interface:  e.Interface,
				RepairHint: "run atrakta projection repair --all",
			})
			continue
		}
		for _, p := range e.Files {
			abs := filepath.Join(repoRoot, filepath.FromSlash(p))
			if _, err := os.Lstat(abs); err != nil {
				addParityBlocking(report, ParityFinding{
					Code:       "projection_missing",
					Severity:   "blocking",
					Message:    "projection file recorded in manifest is missing",
					Path:       p,
					Interface:  e.Interface,
					RepairHint: "run atrakta projection repair --all",
				})
			}
		}
		if e.Status != "" && e.Status != "ok" && e.Status != "skipped" {
			addParityWarning(report, ParityFinding{
				Code:       "projection_status_warning",
				Severity:   "warning",
				Message:    "projection manifest entry status is not ok/skipped",
				Interface:  e.Interface,
				Actual:     e.Status,
				RepairHint: "run atrakta projection repair --all",
			})
		}
	}
}

func validateExtensionManifestFiles(repoRoot string, ms manifest.Status, report *ParityReport) {
	for _, e := range ms.Extension.Entries {
		for _, p := range e.Files {
			abs := filepath.Join(repoRoot, filepath.FromSlash(p))
			if _, err := os.Lstat(abs); err != nil {
				addParityWarning(report, ParityFinding{
					Code:       "extension_projection_drift",
					Severity:   "warning",
					Message:    "extension manifest file is missing",
					Path:       p,
					Interface:  e.Kind,
					RepairHint: "run atrakta projection render --all",
				})
			}
		}
	}
}

func validateManifestHashes(st state.State, ms manifest.Status, report *ParityReport) {
	if st.Projection == nil {
		if len(ms.Projection.Entries) > 0 {
			addParityWarning(report, ParityFinding{
				Code:       "projection_state_missing",
				Severity:   "warning",
				Message:    "state.projection is missing while manifest has projection entries",
				RepairHint: "run atrakta projection repair --all",
			})
		}
		return
	}
	computed, err := projectionManifestHash(ms.Projection)
	if err != nil {
		addParityBlocking(report, ParityFinding{
			Code:       "render_hash_compute_failed",
			Severity:   "blocking",
			Message:    "failed to compute projection manifest hash",
			Actual:     err.Error(),
			RepairHint: "run atrakta projection repair --all",
		})
		return
	}
	if st.Projection.RenderHash != "" && st.Projection.RenderHash != computed {
		addParityBlocking(report, ParityFinding{
			Code:       "render_hash_mismatch",
			Severity:   "blocking",
			Message:    "state projection render_hash does not match manifest",
			Expected:   computed,
			Actual:     st.Projection.RenderHash,
			RepairHint: "run atrakta projection repair --all",
		})
	}
	sourceHashes := map[string]struct{}{}
	for _, e := range ms.Projection.Entries {
		if strings.TrimSpace(e.SourceHash) == "" {
			continue
		}
		sourceHashes[e.SourceHash] = struct{}{}
	}
	if len(sourceHashes) > 1 {
		addParityWarning(report, ParityFinding{
			Code:       "source_hash_inconsistent",
			Severity:   "warning",
			Message:    "projection manifest contains multiple source_hash values",
			RepairHint: "run atrakta projection render --all",
		})
	}
	if st.Projection.SourceHash != "" && len(sourceHashes) == 1 {
		var only string
		for h := range sourceHashes {
			only = h
		}
		if only != st.Projection.SourceHash {
			addParityWarning(report, ParityFinding{
				Code:       "source_hash_mismatch",
				Severity:   "warning",
				Message:    "state projection source_hash differs from manifest",
				Expected:   only,
				Actual:     st.Projection.SourceHash,
				RepairHint: "run atrakta projection render --all",
			})
		}
	}
}

func validateManagedParity(repoRoot string, st state.State, sourceAGENTS string, report *ParityReport) {
	for p, rec := range st.ManagedPaths {
		abs := filepath.Join(repoRoot, filepath.FromSlash(p))
		if _, err := os.Lstat(abs); err != nil {
			addParityBlocking(report, ParityFinding{
				Code:       "projection_missing",
				Severity:   "blocking",
				Message:    "managed projection file is missing",
				Path:       p,
				Interface:  rec.Interface,
				RepairHint: "run atrakta projection repair --interface " + rec.Interface,
			})
			continue
		}
		exp := proof.Expected{
			Fingerprint: rec.Fingerprint,
			Target:      rec.Target,
			TemplateID:  rec.TemplateID,
			SourceText:  sourceAGENTS,
		}
		if exp.Target == "" {
			exp.Target = "AGENTS.md"
		}
		if err := proof.Revalidate(repoRoot, p, rec, exp); err != nil {
			addParityBlocking(report, ParityFinding{
				Code:       "managed_block_corruption",
				Severity:   "blocking",
				Message:    "managed projection artifact failed revalidation",
				Path:       p,
				Interface:  rec.Interface,
				Actual:     err.Error(),
				RepairHint: "run atrakta projection repair --interface " + rec.Interface,
			})
		}
	}
}

func validateApprovalSurface(c contract.Contract, report *ParityReport) {
	if c.Parity == nil {
		addParityBlocking(report, ParityFinding{
			Code:       "approval_surface_mismatch",
			Severity:   "blocking",
			Message:    "parity config is missing",
			RepairHint: "restore parity section in .atrakta/contract.json",
		})
		return
	}
	if c.Parity.ApprovalSurface.ApprovalRequiredForRef != "tools.approval_required_for" {
		addParityBlocking(report, ParityFinding{
			Code:       "approval_surface_mismatch",
			Severity:   "blocking",
			Message:    "approval surface reference is not aligned with tools.approval_required_for",
			Expected:   "tools.approval_required_for",
			Actual:     c.Parity.ApprovalSurface.ApprovalRequiredForRef,
			RepairHint: "run atrakta doctor --sync-proposal",
		})
	}
	if len(c.Tools.ApprovalRequiredFor) == 0 {
		addParityBlocking(report, ParityFinding{
			Code:       "approval_surface_mismatch",
			Severity:   "blocking",
			Message:    "tools.approval_required_for is empty",
			RepairHint: "run atrakta doctor --sync-proposal",
		})
	}
}

func validateOutputSurface(c contract.Contract, report *ParityReport) {
	if c.Parity == nil {
		return
	}
	planFormat := strings.TrimSpace(strings.ToLower(c.Parity.OutputSurface.PlanFormat))
	statusJSON := os.Getenv("ATRAKTA_STATUS_JSON") == "1"
	if planFormat == "json" && !statusJSON {
		addParityWarning(report, ParityFinding{
			Code:       "output_surface_mismatch",
			Severity:   "warning",
			Message:    "parity output surface expects json but ATRAKTA_STATUS_JSON is not enabled",
			Expected:   "ATRAKTA_STATUS_JSON=1",
			Actual:     "ATRAKTA_STATUS_JSON=0",
			RepairHint: "enable ATRAKTA_STATUS_JSON=1 for parity-aligned json output",
		})
	}
	if planFormat == "markdown" && statusJSON {
		addParityWarning(report, ParityFinding{
			Code:       "output_surface_mismatch",
			Severity:   "warning",
			Message:    "parity output surface expects markdown but json output mode is enabled",
			Expected:   "ATRAKTA_STATUS_JSON=0",
			Actual:     "ATRAKTA_STATUS_JSON=1",
			RepairHint: "disable ATRAKTA_STATUS_JSON for markdown output parity",
		})
	}
}

func validateNonInteractiveSurface(report *ParityReport) {
	if os.Getenv("ATRAKTA_NONINTERACTIVE") != "1" {
		return
	}
	st, err := os.Stdin.Stat()
	if err != nil {
		return
	}
	if (st.Mode() & os.ModeCharDevice) != 0 {
		addParityWarning(report, ParityFinding{
			Code:       "noninteractive_mismatch",
			Severity:   "warning",
			Message:    "ATRAKTA_NONINTERACTIVE=1 is set but stdin is interactive",
			Expected:   "stdin non-interactive",
			Actual:     "stdin interactive",
			RepairHint: "unset ATRAKTA_NONINTERACTIVE or run in non-interactive context",
		})
	}
}

func projectionManifestHash(pm model.ProjectionManifest) (string, error) {
	b, err := util.MarshalCanonical(pm)
	if err != nil {
		return "", fmt.Errorf("canonical manifest hash: %w", err)
	}
	return util.SHA256Tagged(b), nil
}

func suggestParityCommands(report ParityReport) []string {
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
		case "manifest_missing", "projection_missing", "render_hash_mismatch", "managed_block_corruption", "manifest_invalid":
			add("atrakta projection repair --all")
		case "approval_surface_mismatch":
			add("atrakta doctor --sync-proposal")
		}
	}
	for _, f := range report.Warnings {
		switch f.Code {
		case "output_surface_mismatch":
			add("ATRAKTA_STATUS_JSON=1 atrakta doctor --parity --json")
		case "noninteractive_mismatch":
			add("unset ATRAKTA_NONINTERACTIVE")
		case "extension_manifest_missing":
			add("atrakta projection render --all")
		}
	}
	out := make([]string, 0, len(set))
	for cmd := range set {
		out = append(out, cmd)
	}
	sort.Strings(out)
	return out
}

func addParityBlocking(report *ParityReport, f ParityFinding) {
	f.Severity = "blocking"
	report.BlockingIssues = append(report.BlockingIssues, f)
}

func addParityWarning(report *ParityReport, f ParityFinding) {
	f.Severity = "warning"
	report.Warnings = append(report.Warnings, f)
}

func (r ParityReport) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}
