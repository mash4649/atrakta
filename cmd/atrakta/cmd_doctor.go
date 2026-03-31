package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/audit"
	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/projection"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
	resolveoperationcapability "github.com/mash4649/atrakta/v0/resolvers/operations/resolve-operation-capability"
)

type doctorCheck struct {
	CheckID     string `json:"check_id"`
	Status      string `json:"status"`
	Detail      string `json:"detail"`
	Remediation string `json:"remediation,omitempty"`
}

type doctorReport struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Checks  []doctorCheck `json:"checks"`
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var execute bool
	var jsonOut bool
	var artifactDir string

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&execute, "execute", false, "execute mapped pipeline mode")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}

	capOut := resolveoperationcapability.ResolveOperationCapability(resolveoperationcapability.Input{
		CommandOrAlias: "doctor",
	})
	diagnostics := buildDoctorReport(root)

	response := map[string]any{
		"alias":        "doctor",
		"project_root": root,
		"capability":   capOut,
		"doctor":       diagnostics,
		"checks":       diagnostics.Checks,
		"status":       diagnostics.Status,
		"message":      diagnostics.Message,
		"next_action":  nextDoctorAction(diagnostics),
	}

	if execute {
		mode := "inspect"
		decision, _ := capOut.Decision.(resolveoperationcapability.CapabilityDecision)
		switch decision.EffectiveActionClass {
		case resolveoperationcapability.ActionPropose:
			mode = "preview"
		case resolveoperationcapability.ActionApply:
			mode = "simulate"
		}
		input, err := buildDefaultInput(mode)
		if err != nil {
			return err
		}
		bundle, err := executeBundle(mode, input)
		if err != nil {
			return err
		}
		response["mode"] = mode
		response["bundle"] = bundle
	}

	_ = jsonOut
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(response); err != nil {
		return err
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "doctor.result.json", response); err != nil {
			return err
		}
	}
	return nil
}

func buildDoctorReport(projectRoot string) doctorReport {
	checks := []doctorCheck{
		checkDoctorStateIntegrity(projectRoot),
		checkDoctorProjectionParity(projectRoot),
		checkDoctorEventChain(projectRoot),
		checkDoctorSecurityModel(projectRoot),
	}

	status := "ok"
	for _, check := range checks {
		switch check.Status {
		case "needs_attention":
			status = "needs_attention"
		case "unknown":
			if status == "ok" {
				status = "unknown"
			}
		}
	}

	message := "doctor checks passed"
	switch status {
	case "needs_attention":
		message = "doctor found issues"
	case "unknown":
		message = "doctor found unknown state"
	}

	return doctorReport{
		Status:  status,
		Message: message,
		Checks:  checks,
	}
}

func checkDoctorStateIntegrity(projectRoot string) doctorCheck {
	taskGraphPath := filepath.Join(projectRoot, ".atrakta", "task-graph.json")

	state, err := runpkg.LoadSessionState(projectRoot)
	if err != nil {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("state.json unreadable: %v", err),
			Remediation: "run `atrakta start` to regenerate session state",
		}
	}
	var graph runpkg.SessionTaskGraph
	if err := readJSONFile(taskGraphPath, &graph); err != nil {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("task-graph.json unreadable: %v", err),
			Remediation: "run `atrakta start` to regenerate session task graph",
		}
	}

	expectedState := "session-state.v1"
	expectedGraph := "session-task-graph.v1"
	if strings.TrimSpace(state.SchemaVersion) != expectedState || strings.TrimSpace(graph.SchemaVersion) != expectedGraph {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("schema_version mismatch: state=%q task_graph=%q", state.SchemaVersion, graph.SchemaVersion),
			Remediation: "rerun `atrakta start` to rewrite state artifacts",
		}
	}
	if strings.TrimSpace(state.Command) == "" || strings.TrimSpace(graph.Command) == "" || state.Command != graph.Command {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("command mismatch: state=%q task_graph=%q", state.Command, graph.Command),
			Remediation: "rerun `atrakta start` or `atrakta resume` to realign session state",
		}
	}

	canonical, err := runpkg.DetectCanonicalState(projectRoot)
	if err != nil {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "unknown",
			Detail:      fmt.Sprintf("canonical state detection failed: %v", err),
			Remediation: "inspect the workspace and rerun doctor",
		}
	}
	if canonical == runpkg.StatePartialState || canonical == runpkg.StateCorruptState {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("canonical state=%s", canonical),
			Remediation: "repair canonical workspace state before rerunning doctor",
		}
	}
	if strings.TrimSpace(state.CanonicalState) != canonical {
		return doctorCheck{
			CheckID:     "state-integrity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("canonical_state mismatch: state=%q detected=%q", state.CanonicalState, canonical),
			Remediation: "rerun `atrakta start` to refresh session state",
		}
	}

	return doctorCheck{
		CheckID: "state-integrity",
		Status:  "ok",
		Detail:  fmt.Sprintf("state schema=%s task_graph schema=%s canonical=%s", state.SchemaVersion, graph.SchemaVersion, canonical),
	}
}

func checkDoctorProjectionParity(projectRoot string) doctorCheck {
	res, err := projection.Status(projectRoot, projection.DefaultTarget, "")
	if err != nil {
		return doctorCheck{
			CheckID:     "projection-parity",
			Status:      "unknown",
			Detail:      fmt.Sprintf("projection status failed: %v", err),
			Remediation: "rerun `atrakta projection status` to inspect projection health",
		}
	}

	if res.ProjectionStatus != "up_to_date" {
		return doctorCheck{
			CheckID:     "projection-parity",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("%s drift=%t", res.ProjectionStatus, res.Drift),
			Remediation: "run `atrakta projection repair --project-root <root>`",
		}
	}

	return doctorCheck{
		CheckID: "projection-parity",
		Status:  "ok",
		Detail:  fmt.Sprintf("%s up_to_date", res.TargetPath),
	}
}

func checkDoctorEventChain(projectRoot string) doctorCheck {
	auditRoot := filepath.Join(projectRoot, ".atrakta", "audit")
	runEventsPath := filepath.Join(auditRoot, "events", "run-events.jsonl")
	if _, err := os.Stat(runEventsPath); err != nil {
		if os.IsNotExist(err) {
			return doctorCheck{
				CheckID:     "event-chain",
				Status:      "needs_attention",
				Detail:      "run-events.jsonl missing",
				Remediation: "run `atrakta start` to initialize the runtime event chain",
			}
		}
		return doctorCheck{
			CheckID:     "event-chain",
			Status:      "unknown",
			Detail:      fmt.Sprintf("run-events stat failed: %v", err),
			Remediation: "inspect the audit store and rerun doctor",
		}
	}

	if err := audit.VerifyRunEventsIntegrity(auditRoot, audit.LevelA2); err != nil {
		return doctorCheck{
			CheckID:     "event-chain",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("run-events integrity error: %v", err),
			Remediation: "rerun `atrakta start` or `atrakta resume` to rebuild the event chain",
		}
	}

	return doctorCheck{
		CheckID: "event-chain",
		Status:  "ok",
		Detail:  "run-events integrity verified",
	}
}

func checkDoctorSecurityModel(projectRoot string) doctorCheck {
	contract, err := runpkg.LoadMachineContract(projectRoot)
	if err != nil {
		return doctorCheck{
			CheckID:     "security-model",
			Status:      "needs_attention",
			Detail:      fmt.Sprintf("machine contract invalid: %v", err),
			Remediation: "regenerate `.atrakta/contract.json` and rerun doctor",
		}
	}

	issues := make([]string, 0)
	if boundary, ok := contract["boundary"].(map[string]any); ok {
		if managedRoot, ok := boundary["managed_root"].(string); !ok || strings.TrimSpace(managedRoot) != ".atrakta/" {
			issues = append(issues, "boundary.managed_root must be .atrakta/")
		}
	} else {
		issues = append(issues, "boundary section missing")
	}

	if tools, ok := contract["tools"].(map[string]any); ok {
		allow := stringSlice(tools["allow"])
		if !hasAllStrings(allow, []string{"create", "edit", "run"}) {
			issues = append(issues, "tools.allow must include create, edit, and run")
		}
		for _, tool := range allow {
			if isSecurityToolForbidden(tool) {
				issues = append(issues, "tools.allow includes forbidden capability: "+tool)
			}
		}
	} else {
		issues = append(issues, "tools section missing")
	}

	if security, ok := contract["security"].(map[string]any); ok {
		if s, ok := security["destructive"].(string); !ok || strings.TrimSpace(s) != "deny" {
			issues = append(issues, "security.destructive must be deny")
		}
		if s, ok := security["external_send"].(string); !ok || strings.TrimSpace(s) != "deny" {
			issues = append(issues, "security.external_send must be deny")
		}
		if s, ok := security["approval"].(string); !ok || strings.TrimSpace(s) != "explicit" {
			issues = append(issues, "security.approval must be explicit")
		}
		if s, ok := security["permission_model"].(string); !ok || strings.TrimSpace(s) != "proposal_only" {
			issues = append(issues, "security.permission_model must be proposal_only")
		}
	} else {
		issues = append(issues, "security section missing")
	}

	if len(issues) > 0 {
		return doctorCheck{
			CheckID:     "security-model",
			Status:      "needs_attention",
			Detail:      strings.Join(issues, "; "),
			Remediation: "regenerate `.atrakta/contract.json` with the safe security model and rerun doctor",
		}
	}

	tools := stringSlice(contractMapValue(contract, "tools", "allow"))
	return doctorCheck{
		CheckID: "security-model",
		Status:  "ok",
		Detail:  fmt.Sprintf("managed_root=.atrakta/ allow=%v destructive=deny external_send=deny approval=explicit permission_model=proposal_only", tools),
	}
}

func readJSONFile(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func nextDoctorAction(report doctorReport) string {
	if report.Status == "ok" {
		return "continue"
	}
	return "inspect"
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func hasAllStrings(items []string, required []string) bool {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	for _, item := range required {
		if _, ok := set[item]; !ok {
			return false
		}
	}
	return true
}

func isSecurityToolForbidden(tool string) bool {
	switch strings.TrimSpace(tool) {
	case "delete", "rm", "push", "publish", "curl":
		return true
	default:
		return false
	}
}

func contractMapValue(contract map[string]any, objectKey, fieldKey string) any {
	obj, _ := contract[objectKey].(map[string]any)
	if obj == nil {
		return nil
	}
	return obj[fieldKey]
}
