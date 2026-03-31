package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	resolveauditrequirements "github.com/mash4649/atrakta/v0/resolvers/audit/resolve-audit-requirements"
	resolveextensionorder "github.com/mash4649/atrakta/v0/resolvers/extension/resolve-extension-order"
	resolvefailuretier "github.com/mash4649/atrakta/v0/resolvers/failure/resolve-failure-tier"
	strictstatemachine "github.com/mash4649/atrakta/v0/resolvers/failure/strict-state-machine"
	resolveguidanceprecedence "github.com/mash4649/atrakta/v0/resolvers/guidance/resolve-guidance-precedence"
	classifylayer "github.com/mash4649/atrakta/v0/resolvers/layer/classify-layer"
	detectlegacydrift "github.com/mash4649/atrakta/v0/resolvers/legacy/detect-legacy-drift"
	resolvelegacystatus "github.com/mash4649/atrakta/v0/resolvers/legacy/resolve-legacy-status"
	checkmutationscope "github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
	resolveoperationcapability "github.com/mash4649/atrakta/v0/resolvers/operations/resolve-operation-capability"
	resolvesurfaceportability "github.com/mash4649/atrakta/v0/resolvers/portability/resolve-surface-portability"
	checkprojectioneligibility "github.com/mash4649/atrakta/v0/resolvers/projection/check-projection-eligibility"
)

// CaseResult is one fixture execution result.
type CaseResult struct {
	Fixture string `json:"fixture"`
	Case    int    `json:"case"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// Report is fixture-run summary.
type Report struct {
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
	Results []CaseResult `json:"results"`
}

// RunAll executes all known fixtures under root.
func RunAll(root string) (Report, error) {
	files, err := findFixtures(root)
	if err != nil {
		return Report{}, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	report := Report{Results: make([]CaseResult, 0)}
	for _, f := range files {
		results, err := runFixture(f)
		if err != nil {
			report.Results = append(report.Results, CaseResult{Fixture: normalizeFixturePath(rootAbs, f), Case: 0, Passed: false, Message: err.Error()})
			report.Failed++
			continue
		}
		for _, r := range results {
			r.Fixture = normalizeFixturePath(rootAbs, r.Fixture)
			report.Results = append(report.Results, r)
			if r.Passed {
				report.Passed++
			} else {
				report.Failed++
			}
		}
	}
	return report, nil
}

func findFixtures(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" && filepath.Base(path) != ".gitkeep" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func runFixture(path string) ([]CaseResult, error) {
	base := filepath.Base(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	switch base {
	case "classify-layer.fixture.json":
		var payload struct {
			Cases []struct {
				Input struct {
					Kind string `json:"kind"`
				} `json:"input"`
				ExpectedLayer string `json:"expected_layer"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		out := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			got := classifylayer.ClassifyLayer(classifylayer.Item{Kind: c.Input.Kind})
			pass := got.Decision == c.ExpectedLayer
			msg := ""
			if !pass {
				msg = fmt.Sprintf("got=%v want=%v", got.Decision, c.ExpectedLayer)
			}
			out = append(out, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return out, nil
	case "guidance-precedence.fixture.json":
		var payload struct {
			Input         []resolveguidanceprecedence.GuidanceItem `json:"input"`
			ExpectedOrder []string                                 `json:"expected_order"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		got := resolveguidanceprecedence.ResolveGuidancePrecedence(payload.Input)
		decision := got.Decision.(resolveguidanceprecedence.GuidanceDecision)
		pass := equalStringSlices(decision.OrderedIDs, payload.ExpectedOrder)
		msg := ""
		if !pass {
			msg = fmt.Sprintf("got=%v want=%v", decision.OrderedIDs, payload.ExpectedOrder)
		}
		return []CaseResult{{Fixture: path, Case: 1, Passed: pass, Message: msg}}, nil
	case "projection-eligibility.fixture.json":
		var payload struct {
			Input    []checkprojectioneligibility.Source `json:"input"`
			Expected []struct {
				Eligibility string `json:"eligibility"`
			} `json:"expected"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		results := make([]CaseResult, 0, len(payload.Input))
		for i, in := range payload.Input {
			got := checkprojectioneligibility.CheckProjectionEligibility(in).Decision.(checkprojectioneligibility.ProjectionDecision)
			want := payload.Expected[i].Eligibility
			pass := got.Eligibility == want
			msg := ""
			if !pass {
				msg = fmt.Sprintf("got=%s want=%s", got.Eligibility, want)
			}
			results = append(results, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return results, nil
	case "surface-portability.fixture.json":
		var payload struct {
			Cases []struct {
				Input struct {
					InterfaceID         string                                        `json:"interface_id"`
					RequestedTargets    []string                                      `json:"requested_targets"`
					AvailableSources    []string                                      `json:"available_sources"`
					BindingCapabilities resolvesurfaceportability.BindingCapabilities `json:"binding_capabilities"`
					DegradePolicy       string                                        `json:"degrade_policy"`
				} `json:"input"`
				Expected struct {
					Status            string   `json:"status"`
					Supported         []string `json:"supported,omitempty"`
					Degraded          []string `json:"degraded,omitempty"`
					Unsupported       []string `json:"unsupported,omitempty"`
					NextAllowedAction string   `json:"next_allowed_action,omitempty"`
				} `json:"expected"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		results := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			out := resolvesurfaceportability.ResolveSurfacePortability(resolvesurfaceportability.Input{
				InterfaceID:         c.Input.InterfaceID,
				RequestedTargets:    c.Input.RequestedTargets,
				AvailableSources:    c.Input.AvailableSources,
				BindingCapabilities: c.Input.BindingCapabilities,
				DegradePolicy:       c.Input.DegradePolicy,
			})
			got := out.Decision.(resolvesurfaceportability.PortabilityDecision)
			pass := got.PortabilityStatus == c.Expected.Status &&
				equalStringSlices(got.SupportedTargets, c.Expected.Supported) &&
				equalStringSlices(got.DegradedTargets, c.Expected.Degraded) &&
				equalStringSlices(got.UnsupportedTargets, c.Expected.Unsupported)
			msg := ""
			if !pass {
				msg = fmt.Sprintf("got status=%s supported=%v degraded=%v unsupported=%v", got.PortabilityStatus, got.SupportedTargets, got.DegradedTargets, got.UnsupportedTargets)
			}
			if c.Expected.NextAllowedAction != "" && out.NextAllowedAction != c.Expected.NextAllowedAction {
				pass = false
				msg = fmt.Sprintf("next_action got=%s want=%s", out.NextAllowedAction, c.Expected.NextAllowedAction)
			}
			results = append(results, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return results, nil
	case "failure-routing.fixture.json":
		var payload struct {
			Input struct {
				FailureClass string                     `json:"failure_class"`
				Context      resolvefailuretier.Context `json:"context"`
			} `json:"input"`
			Expected struct {
				ResolvedTier     string `json:"resolved_tier"`
				StrictTransition string `json:"strict_transition"`
			} `json:"expected"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		got := resolvefailuretier.ResolveFailureTier(payload.Input.FailureClass, payload.Input.Context).Decision.(resolvefailuretier.FailureDecision)
		pass := got.ResolvedTier == payload.Expected.ResolvedTier && got.StrictTransition == payload.Expected.StrictTransition
		msg := ""
		if !pass {
			msg = fmt.Sprintf("got=(%s,%s) want=(%s,%s)", got.ResolvedTier, got.StrictTransition, payload.Expected.ResolvedTier, payload.Expected.StrictTransition)
		}
		return []CaseResult{{Fixture: path, Case: 1, Passed: pass, Message: msg}}, nil
	case "strict-state-machine.fixture.json":
		var payload struct {
			Cases []struct {
				Input                     strictstatemachine.StateInput `json:"input"`
				ExpectedNextState         string                        `json:"expected_next_state"`
				ExpectedNextAllowedAction string                        `json:"expected_next_allowed_action,omitempty"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		results := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			out := strictstatemachine.Transition(c.Input)
			got := out.Decision.(strictstatemachine.StateDecision)
			pass := got.NextState == c.ExpectedNextState
			msg := ""
			if !pass {
				msg = fmt.Sprintf("next_state got=%s want=%s", got.NextState, c.ExpectedNextState)
			}
			if c.ExpectedNextAllowedAction != "" && out.NextAllowedAction != c.ExpectedNextAllowedAction {
				pass = false
				msg = fmt.Sprintf("next_action got=%s want=%s", out.NextAllowedAction, c.ExpectedNextAllowedAction)
			}
			results = append(results, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return results, nil
	case "mutation-scope.fixture.json":
		var payload struct {
			Cases []struct {
				Input         checkmutationscope.Target `json:"input"`
				ExpectedScope string                    `json:"expected_scope"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		res := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			got := checkmutationscope.CheckMutationScope(c.Input).Decision.(checkmutationscope.MutationDecision)
			pass := got.Scope == c.ExpectedScope
			msg := ""
			if !pass {
				msg = fmt.Sprintf("got=%s want=%s", got.Scope, c.ExpectedScope)
			}
			res = append(res, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return res, nil
	case "operation-capability.fixture.json":
		var payload struct {
			Cases []struct {
				Input                        resolveoperationcapability.Input `json:"input"`
				ExpectedCapability           string                           `json:"expected_capability,omitempty"`
				ExpectedEffectiveActionClass string                           `json:"expected_effective_action_class,omitempty"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		res := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			got := resolveoperationcapability.ResolveOperationCapability(c.Input).Decision.(resolveoperationcapability.CapabilityDecision)
			pass := true
			msg := ""
			if c.ExpectedCapability != "" && got.CanonicalCapability != c.ExpectedCapability {
				pass = false
				msg = fmt.Sprintf("capability got=%s want=%s", got.CanonicalCapability, c.ExpectedCapability)
			}
			if c.ExpectedEffectiveActionClass != "" && got.EffectiveActionClass != c.ExpectedEffectiveActionClass {
				pass = false
				msg = fmt.Sprintf("effective got=%s want=%s", got.EffectiveActionClass, c.ExpectedEffectiveActionClass)
			}
			res = append(res, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return res, nil
	case "legacy-status.fixture.json":
		var payload struct {
			Cases []struct {
				Input          resolvelegacystatus.Asset `json:"input"`
				ExpectedStatus string                    `json:"expected_status"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		res := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			got := resolvelegacystatus.ResolveLegacyStatus(c.Input).Decision.(resolvelegacystatus.LegacyDecision)
			pass := got.Status == c.ExpectedStatus
			msg := ""
			if !pass {
				msg = fmt.Sprintf("got=%s want=%s", got.Status, c.ExpectedStatus)
			}
			res = append(res, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return res, nil
	case "legacy-drift.fixture.json":
		var payload struct {
			Cases []struct {
				Input              detectlegacydrift.Input `json:"input"`
				ExpectedSeverity   string                  `json:"expected_severity"`
				ExpectedNextAction string                  `json:"expected_next_allowed_action"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		res := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			out := detectlegacydrift.DetectLegacyDrift(c.Input)
			got := out.Decision.(detectlegacydrift.DriftDecision)
			pass := got.Severity == c.ExpectedSeverity && out.NextAllowedAction == c.ExpectedNextAction
			msg := ""
			if !pass {
				msg = fmt.Sprintf("severity/next got=(%s,%s) want=(%s,%s)", got.Severity, out.NextAllowedAction, c.ExpectedSeverity, c.ExpectedNextAction)
			}
			res = append(res, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return res, nil
	case "extension-order.fixture.json":
		var payload struct {
			Input         []resolveextensionorder.Item `json:"input"`
			ExpectedOrder []string                     `json:"expected_order"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		got := resolveextensionorder.ResolveExtensionOrder(payload.Input).Decision.(resolveextensionorder.ExtensionDecision)
		ordered := make([]string, 0, len(got.Ordered))
		for _, it := range got.Ordered {
			ordered = append(ordered, it.ID)
		}
		pass := equalStringSlices(ordered, payload.ExpectedOrder)
		msg := ""
		if !pass {
			msg = fmt.Sprintf("got=%v want=%v", ordered, payload.ExpectedOrder)
		}
		return []CaseResult{{Fixture: path, Case: 1, Passed: pass, Message: msg}}, nil
	case "audit-requirements.fixture.json":
		var payload struct {
			Cases []struct {
				Input                          resolveauditrequirements.Input `json:"input"`
				ExpectedRequiredIntegrityLevel string                         `json:"expected_required_integrity_level,omitempty"`
				ExpectedNextAllowedAction      string                         `json:"expected_next_allowed_action,omitempty"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		res := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			out := resolveauditrequirements.ResolveAuditRequirements(c.Input)
			got := out.Decision.(resolveauditrequirements.AuditDecision)
			pass := true
			msg := ""
			if c.ExpectedRequiredIntegrityLevel != "" && got.RequiredIntegrityLevel != c.ExpectedRequiredIntegrityLevel {
				pass = false
				msg = fmt.Sprintf("level got=%s want=%s", got.RequiredIntegrityLevel, c.ExpectedRequiredIntegrityLevel)
			}
			if c.ExpectedNextAllowedAction != "" && out.NextAllowedAction != c.ExpectedNextAllowedAction {
				pass = false
				msg = fmt.Sprintf("next got=%s want=%s", out.NextAllowedAction, c.ExpectedNextAllowedAction)
			}
			res = append(res, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return res, nil
	case "onboarding-proposal.fixture.json":
		var payload struct {
			Cases []struct {
				Assets                       []string          `json:"assets"`
				ExpectedDetectedRisks        []string          `json:"expected_detected_risks"`
				ExpectedMode                 string            `json:"expected_mode"`
				ExpectedConflicts            []string          `json:"expected_conflicts"`
				ExpectedFailureRouting       map[string]string `json:"expected_failure_routing"`
				ExpectedNextActions          []string          `json:"expected_next_actions"`
				ExpectedManagedScope         map[string]string `json:"expected_managed_scope"`
				ExpectedGuidanceStrength     map[string]string `json:"expected_guidance_strength"`
				ExpectedCapabilitiesContains []string          `json:"expected_capabilities_contains"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return nil, err
		}
		res := make([]CaseResult, 0, len(payload.Cases))
		for i, c := range payload.Cases {
			pass := true
			msgs := make([]string, 0)

			out := onboarding.BuildOnboardingProposalFromDetectedAssets(c.Assets)
			{
				if c.ExpectedMode != "" && out.InferredMode != c.ExpectedMode {
					pass = false
					msgs = append(msgs, fmt.Sprintf("mode got=%s want=%s", out.InferredMode, c.ExpectedMode))
				}
				if c.ExpectedConflicts != nil && !equalStringSlices(out.Conflicts, c.ExpectedConflicts) {
					pass = false
					msgs = append(msgs, fmt.Sprintf("conflicts got=%v want=%v", out.Conflicts, c.ExpectedConflicts))
				}
				if c.ExpectedDetectedRisks != nil && !equalStringSlices(out.DetectedRisks, c.ExpectedDetectedRisks) {
					pass = false
					msgs = append(msgs, fmt.Sprintf("detected_risks got=%v want=%v", out.DetectedRisks, c.ExpectedDetectedRisks))
				}
				for k, v := range c.ExpectedFailureRouting {
					got, ok := onboardingFailureValue(out.InferredFailure, k)
					if !ok || got != v {
						pass = false
						msgs = append(msgs, fmt.Sprintf("failure_routing[%s] got=%s want=%s", k, got, v))
					}
				}
				if c.ExpectedNextActions != nil && !equalStringSlices(out.SuggestedNextActions, c.ExpectedNextActions) {
					pass = false
					msgs = append(msgs, fmt.Sprintf("next_actions got=%v want=%v", out.SuggestedNextActions, c.ExpectedNextActions))
				}
				for k, v := range c.ExpectedManagedScope {
					got, ok := out.InferredManagedScope[k]
					if !ok || fmt.Sprint(got) != v {
						pass = false
						msgs = append(msgs, fmt.Sprintf("managed_scope[%s] got=%v want=%s", k, got, v))
					}
				}
				for k, v := range c.ExpectedGuidanceStrength {
					got, ok := out.InferredGuidance[k]
					if !ok || fmt.Sprint(got) != v {
						pass = false
						msgs = append(msgs, fmt.Sprintf("guidance_strength[%s] got=%v want=%s", k, got, v))
					}
				}
				for _, wantCap := range c.ExpectedCapabilitiesContains {
					if !contains(out.InferredCapabilities, wantCap) {
						pass = false
						msgs = append(msgs, fmt.Sprintf("capability missing: %s", wantCap))
					}
				}
			}

			msg := ""
			if len(msgs) > 0 {
				msg = strings.Join(msgs, "; ")
			}
			res = append(res, CaseResult{Fixture: path, Case: i + 1, Passed: pass, Message: msg})
		}
		return res, nil
	default:
		// Ignore non-executable examples and unknown fixtures.
		return []CaseResult{}, nil
	}
}

func equalStringSlices(a, b []string) bool {
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

func normalizeFixturePath(rootAbs, fixturePath string) string {
	absPath, err := filepath.Abs(fixturePath)
	if err != nil {
		return filepath.ToSlash(fixturePath)
	}
	rel, err := filepath.Rel(rootAbs, absPath)
	if err != nil {
		return filepath.ToSlash(fixturePath)
	}
	if strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(fixturePath)
	}
	return filepath.ToSlash(filepath.Join("fixtures", rel))
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func materializeDetectedAsset(root, relPath string) error {
	rel := filepath.Clean(filepath.FromSlash(relPath))
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("invalid asset path %q", relPath)
	}
	abs := filepath.Join(root, rel)

	if isFileAsset(relPath) {
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		body := "# fixture\n"
		switch filepath.Base(rel) {
		case "package.json", ".mcp.json", "mcp.json":
			body = "{}\n"
		case "pyproject.toml", "Cargo.toml":
			body = "[tool]\n"
		}
		return os.WriteFile(abs, []byte(body), 0o644)
	}
	return os.MkdirAll(abs, 0o755)
}

func isFileAsset(relPath string) bool {
	switch relPath {
	case "AGENTS.md", ".mcp.json", "mcp.json", "package.json", "pyproject.toml", "Cargo.toml":
		return true
	case ".cursor", ".cursor/rules", ".github/workflows", "scripts", "docs", "tests", "src", "app", ".vscode":
		return false
	}
	ext := filepath.Ext(relPath)
	return ext != ""
}

func onboardingFailureValue(v onboarding.FailurePreview, key string) (string, bool) {
	switch key {
	case "failure_class":
		return v.FailureClass, true
	case "scope":
		return v.Scope, true
	case "default_tier":
		return v.DefaultTier, true
	case "resolved_tier":
		return v.ResolvedTier, true
	case "strict_transition":
		return v.StrictTransition, true
	case "next_allowed_action":
		return v.NextAllowedAction, true
	default:
		return "", false
	}
}
