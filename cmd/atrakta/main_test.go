package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/pipeline"
	runpkg "github.com/mash4649/atrakta/v0/internal/run"
)

func TestWriteArtifact(t *testing.T) {
	dir := t.TempDir()
	payload := map[string]string{"mode": "inspect"}
	if err := writeArtifact(dir, "inspect.bundle.json", payload); err != nil {
		t.Fatalf("writeArtifact: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "inspect.bundle.json"))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal artifact: %v", err)
	}
	if got["mode"] != "inspect" {
		t.Fatalf("artifact mode=%q", got["mode"])
	}
}

func TestRunExportSnapshots(t *testing.T) {
	dir := t.TempDir()
	if err := runExportSnapshots([]string{"--dir", dir}); err != nil {
		t.Fatalf("runExportSnapshots: %v", err)
	}

	required := []string{
		"onboarding.proposal.json",
		"inspect.onboard.bundle.json",
		"preview.onboard.bundle.json",
		"simulate.onboard.bundle.json",
		"inspect.bundle.json",
		"preview.bundle.json",
		"simulate.bundle.json",
		"fixtures.report.json",
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing snapshot %s: %v", name, err)
		}
	}
}

func TestRunVerifyCoverage(t *testing.T) {
	if err := runVerifyCoverage(nil); err != nil {
		t.Fatalf("runVerifyCoverage: %v", err)
	}
}

func TestRunOnboard(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	artifactDir := t.TempDir()
	if err := runOnboard([]string{"--project-root", projectRoot, "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runOnboard: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "onboarding.proposal.json")); err != nil {
		t.Fatalf("missing onboarding artifact: %v", err)
	}
}

func TestApplyOnboardingFailure(t *testing.T) {
	input := pipeline.DefaultInput("inspect")
	bundle := onboarding.ProposalBundle{
		Conflicts: []string{"possible duplicate guidance"},
		InferredFailure: onboarding.FailurePreview{
			FailureClass: "legacy_conflict_failure",
			Scope:        "workspace",
			Triggers:     []string{"instruction_conflict", "policy_ambiguity"},
		},
	}
	got := applyOnboardingFailure(input, bundle)
	if got.FailureClass != "legacy_conflict_failure" {
		t.Fatalf("failure class=%q", got.FailureClass)
	}
	if got.FailureContext.Scope != "workspace" {
		t.Fatalf("scope=%q", got.FailureContext.Scope)
	}
	if len(got.FailureContext.Triggers) != 2 {
		t.Fatalf("trigger length=%d", len(got.FailureContext.Triggers))
	}
	if got.FailureContext.IsDiagnosticsOnly {
		t.Fatalf("expected diagnostics false when conflicts exist")
	}
}

func TestRunModeWithOnboardRoot(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}

	artifactDir := t.TempDir()
	if err := runMode("inspect", []string{"--onboard-root", projectRoot, "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMode with onboard root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "inspect.bundle.json")); err != nil {
		t.Fatalf("missing inspect artifact: %v", err)
	}
}

func TestRunAccept(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	artifactDir := t.TempDir()
	if err := runAccept([]string{"--project-root", projectRoot, "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runAccept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta/state/onboarding-state.json")); err != nil {
		t.Fatalf("missing onboarding state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "accept.result.json")); err != nil {
		t.Fatalf("missing accept artifact: %v", err)
	}
}

func TestRunMutatePhases(t *testing.T) {
	projectRoot := t.TempDir()
	artifactDir := t.TempDir()

	if err := runMutate([]string{"inspect", "--target", ".atrakta/generated/x.json", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMutate inspect: %v", err)
	}
	if err := runMutate([]string{"propose", "--target", ".atrakta/generated/x.json", "--content", "{\"x\":1}", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMutate propose: %v", err)
	}
	if err := runMutate([]string{"apply", "--project-root", projectRoot, "--target", ".atrakta/generated/x.json", "--content", "{\"x\":1}\n", "--allow", "--artifact-dir", artifactDir}); err != nil {
		t.Fatalf("runMutate apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta/generated/x.json")); err != nil {
		t.Fatalf("missing applied mutation target: %v", err)
	}
}

func TestRunAuditAppendAndVerify(t *testing.T) {
	projectRoot := t.TempDir()
	if err := runAudit([]string{"append", "--project-root", projectRoot, "--level", "A2", "--action", "test_append"}); err != nil {
		t.Fatalf("runAudit append: %v", err)
	}
	if err := runAudit([]string{"verify", "--project-root", projectRoot, "--level", "A2"}); err != nil {
		t.Fatalf("runAudit verify: %v", err)
	}
}

func TestRunAliasAndExtensions(t *testing.T) {
	if err := runAlias("doctor", []string{"--execute"}); err != nil {
		t.Fatalf("runAlias doctor: %v", err)
	}

	projectRoot := t.TempDir()
	manifestDir := filepath.Join(projectRoot, "extensions", "manifests")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifests: %v", err)
	}
	manifest := `{"name":"default","items":[{"id":"policy-1","kind":"policy","enabled":true}]}`
	if err := os.WriteFile(filepath.Join(manifestDir, "default.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := runExtensions([]string{"--project-root", projectRoot}); err != nil {
		t.Fatalf("runExtensions: %v", err)
	}
}

func TestRunCommandOnboardingNeedsApprovalNonInteractive(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--non-interactive", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")); err == nil {
		t.Fatalf("onboarding state should not be written without approval")
	}
}

func TestRunCommandOnboardingApproveFlag(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--non-interactive", "--approve", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "onboarding-state.json")); err != nil {
		t.Fatalf("expected onboarding state written: %v", err)
	}
}

func TestRunCommandNeedsInputWhenCanonicalPresentAndInterfaceUnknown(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitNeedsInput {
		t.Fatalf("exit code=%d", code)
	}
}

func TestRunCommandNormalPathWithExplicitInterface(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "audit", "events", "install-events.jsonl")); err != nil {
		t.Fatalf("missing run audit event log: %v", err)
	}
}

func TestRunCommandApplyNeedsApproval(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--apply", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitNeedsApproval {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "audit", "events", "install-events.jsonl")); err != nil {
		t.Fatalf("missing run audit event log: %v", err)
	}
}

func TestRunCommandApplyWithApprove(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--apply", "--approve", "--json"})
	if err != nil {
		t.Fatalf("runCommand: %v", err)
	}
	if code != exitOK {
		t.Fatalf("exit code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "generated", "repo-map.generated.json")); err != nil {
		t.Fatalf("expected generated projection written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "generated", "capabilities.generated.json")); err != nil {
		t.Fatalf("expected capabilities projection written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "generated", "guidance.generated.json")); err != nil {
		t.Fatalf("expected guidance projection written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")); err != nil {
		t.Fatalf("expected run state written: %v", err)
	}
}

func TestRunCommandApplyBlockedByDegradedPortability(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "cursor", "--apply", "--approve", "--json"})
		if err != nil {
			t.Fatalf("runCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal run output: %v", err)
	}
	if out["portability_status"] != "degraded" {
		t.Fatalf("portability_status=%v", out["portability_status"])
	}
	if out["next_allowed_action"] != "propose" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")); err == nil {
		t.Fatalf("run-state should not be written when portability is degraded")
	}
}

func TestRunCommandApplyBlockedByUnsupportedPortability(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{\"status\":\"accepted\"}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	raw := captureStdout(t, func() {
		code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "mcp", "--apply", "--approve", "--json"})
		if err != nil {
			t.Fatalf("runCommand: %v", err)
		}
		if code != exitOK {
			t.Fatalf("exit code=%d", code)
		}
	})

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal run output: %v", err)
	}
	if out["portability_status"] != "unsupported" {
		t.Fatalf("portability_status=%v", out["portability_status"])
	}
	if out["next_allowed_action"] != "propose" {
		t.Fatalf("next_allowed_action=%v", out["next_allowed_action"])
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".atrakta", "state", "run-state.json")); err == nil {
		t.Fatalf("run-state should not be written when portability is unsupported")
	}
}

func TestRunCommandInvalidCanonicalIndex(t *testing.T) {
	projectRoot := t.TempDir()
	policyDir := filepath.Join(projectRoot, ".atrakta", "canonical", "policies", "registry")
	stateDir := filepath.Join(projectRoot, ".atrakta", "state")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "index.json"), []byte("{not-json}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "onboarding-state.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write onboarding state: %v", err)
	}

	code, err := runCommand([]string{"--project-root", projectRoot, "--interface", "generic-cli", "--json"})
	if err == nil {
		t.Fatal("expected canonical parse error")
	}
	if code != exitRuntimeError {
		t.Fatalf("exit code=%d", code)
	}
}

func TestBuildRunInspectInputFromDetectedAssets(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".cursor", "rules"), 0o755); err != nil {
		t.Fatalf("mkdir .cursor/rules: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, ".github", "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}

	input, err := buildRunInspectInput(projectRoot, runpkg.InterfaceResolution{InterfaceID: "generic-cli", Source: "flag"}, true, true)
	if err != nil {
		t.Fatalf("buildRunInspectInput: %v", err)
	}
	if input.FailureClass != "approval_failure" {
		t.Fatalf("failure class=%q", input.FailureClass)
	}
	if input.MutationTarget.Path != ".atrakta/generated/repo-map.generated.json" {
		t.Fatalf("mutation target path=%q", input.MutationTarget.Path)
	}
	if len(input.GuidanceItems) < 4 {
		t.Fatalf("guidance item count too small: %d", len(input.GuidanceItems))
	}
	if input.PortabilityInput.InterfaceID != "generic-cli" {
		t.Fatalf("portability interface=%q", input.PortabilityInput.InterfaceID)
	}
}

func TestBuildRunApplyPlans(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	plans, err := buildRunApplyPlans(projectRoot, runpkg.InterfaceResolution{InterfaceID: "generic-cli", Source: "flag"}, map[string]any{"policy_entry_count": 1})
	if err != nil {
		t.Fatalf("buildRunApplyPlans: %v", err)
	}
	if len(plans) != 3 {
		t.Fatalf("plan count=%d", len(plans))
	}
	if plans[0].Target.Path != ".atrakta/generated/repo-map.generated.json" {
		t.Fatalf("first plan target=%q", plans[0].Target.Path)
	}
}

func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return out
}
