package onboarding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	resolvefailuretier "github.com/mash4649/atrakta/v0/resolvers/failure/resolve-failure-tier"
)

const (
	ModeNewProject       = "new_project"
	ModeBrownfield       = "brownfield_project"
	scopeManagedBlock    = "managed_block"
	scopeManagedInclude  = "managed_include"
	scopeUnmanagedRegion = "unmanaged_user_region"
	scopeProposalOnly    = "proposal_patch_only"
)

// ProposalBundle is the minimum zero-config onboarding output contract.
type ProposalBundle struct {
	DetectedAssets        []string       `json:"detected_assets"`
	DetectedRisks         []string       `json:"detected_risks"`
	InferredMode          string         `json:"inferred_mode"`
	InferredManagedScope  map[string]any `json:"inferred_managed_scope"`
	InferredCapabilities  []string       `json:"inferred_capabilities"`
	InferredGuidance      map[string]any `json:"inferred_guidance_strength"`
	InferredDefaultPolicy map[string]any `json:"inferred_default_policy"`
	InferredFailure       FailurePreview `json:"inferred_failure_routing"`
	Conflicts             []string       `json:"conflicts"`
	SuggestedNextActions  []string       `json:"suggested_next_actions"`
}

// FailurePreview is onboarding-time failure routing preview derived from conflicts.
type FailurePreview struct {
	FailureClass        string   `json:"failure_class"`
	Scope               string   `json:"scope"`
	Triggers            []string `json:"triggers"`
	DefaultTier         string   `json:"default_tier"`
	ResolvedTier        string   `json:"resolved_tier"`
	StrictTransition    string   `json:"strict_transition"`
	RequiresHumanReview bool     `json:"requires_human_review"`
	ExecutionAllowed    bool     `json:"execution_allowed"`
	ProjectionAllowed   bool     `json:"projection_allowed"`
	NextAllowedAction   string   `json:"next_allowed_action"`
}

// DetectProjectRoot finds a project boundary from start path.
// If no marker is found while walking up, it returns the cleaned start directory.
func DetectProjectRoot(start string) (string, error) {
	dir := start
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	dir = filepath.Clean(dir)
	if st, err := os.Stat(dir); err == nil && !st.IsDir() {
		dir = filepath.Dir(dir)
	}
	original := dir

	for {
		if hasProjectMarker(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return original, nil
		}
		dir = parent
	}
}

// DetectMode infers onboarding mode from detected assets.
func DetectMode(projectRoot string) (string, error) {
	assets, err := DetectAssets(projectRoot)
	if err != nil {
		return "", err
	}
	return inferModeFromAssets(assets), nil
}

// DetectAssets discovers the minimum set of guidance/runtime/workflow assets.
func DetectAssets(projectRoot string) ([]string, error) {
	root, err := DetectProjectRoot(projectRoot)
	if err != nil {
		return nil, err
	}

	candidates := []string{
		"AGENTS.md",
		".cursor",
		".cursor/rules",
		".github/workflows",
		"scripts",
		"docs",
		"skills",
		".codex/skills",
		"tests",
		"src",
		"app",
		".vscode",
		".mcp.json",
		"mcp.json",
		"package.json",
		"pyproject.toml",
		"Cargo.toml",
	}

	assets := make([]string, 0, len(candidates))
	for _, rel := range candidates {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err == nil {
			assets = append(assets, rel)
		}
	}
	sort.Strings(assets)
	return assets, nil
}

// InferManagedScope infers a safe initial managed-scope proposal.
func InferManagedScope(detectedAssets []string) map[string]any {
	assetSet := asSet(detectedAssets)
	out := map[string]any{
		".atrakta/canonical/**": scopeManagedBlock,
		".atrakta/generated/**": scopeManagedBlock,
		".atrakta/state/**":     scopeManagedBlock,
		".atrakta/audit/**":     scopeManagedBlock,
		"docs/generated/**":     scopeManagedInclude,
	}

	if hasAny(assetSet, "src") {
		out["src/**"] = scopeUnmanagedRegion
	}
	if hasAny(assetSet, "app") {
		out["app/**"] = scopeUnmanagedRegion
	}
	if hasAny(assetSet, "AGENTS.md") {
		out["AGENTS.md"] = scopeProposalOnly
	}
	if hasAny(assetSet, ".cursor", ".cursor/rules") {
		out[".cursor/**"] = scopeProposalOnly
	}
	return out
}

// InferCapabilities infers initial inspect/propose capabilities.
func InferCapabilities(detectedAssets []string) []string {
	assetSet := asSet(detectedAssets)
	out := map[string]struct{}{
		"inspect_repo":      {},
		"inspect_drift":     {},
		"inspect_parity":    {},
		"propose_repair":    {},
		"read_config":       {},
		"generate_repo_map": {},
	}
	if hasAny(assetSet, "docs") {
		out["search_docs"] = struct{}{}
	}
	if hasAny(assetSet, "tests", "package.json", "pyproject.toml", "Cargo.toml") {
		out["run_tests"] = struct{}{}
	}
	if hasAny(assetSet, ".github/workflows", "scripts") {
		out["inspect_integration"] = struct{}{}
	}
	caps := make([]string, 0, len(out))
	for k := range out {
		caps = append(caps, k)
	}
	sort.Strings(caps)
	return caps
}

// InferGuidanceStrength infers an initial guidance map by surface.
func InferGuidanceStrength(detectedAssets []string) map[string]any {
	assetSet := asSet(detectedAssets)
	out := map[string]any{
		"canonical_policy": "authoritative_constraint",
	}
	if hasAny(assetSet, ".github/workflows") {
		out["workflow_binding"] = "orchestration_constraint"
	}
	if hasAny(assetSet, "AGENTS.md") {
		out["agents_md"] = "advisory_map"
	}
	if hasAny(assetSet, "docs") {
		out["repo_docs"] = "advisory_map"
	}
	if hasAny(assetSet, ".cursor", ".cursor/rules") {
		out["ide_rules"] = "tool_hint"
	}
	if hasAny(assetSet, ".vscode") {
		out["ide_rules"] = "tool_hint"
	}
	return out
}

// InferDefaultPolicy infers the safe default policy set.
func InferDefaultPolicy(mode string, detectedAssets []string) map[string]any {
	out := map[string]any{
		"read_only":          "allow",
		"local_write":        "proposal_only",
		"destructive":        "deny",
		"external_send":      "deny",
		"unknown_capability": "strict",
		"unmapped_guidance":  "advisory_only",
	}
	if mode == ModeNewProject {
		out["managed_bootstrap_apply"] = "allow"
	} else {
		out["managed_bootstrap_apply"] = "proposal_only"
	}
	return out
}

// BuildOnboardingProposal builds the minimum zero-config onboarding proposal bundle.
func BuildOnboardingProposal(projectRoot string) (ProposalBundle, error) {
	root, err := DetectProjectRoot(projectRoot)
	if err != nil {
		return ProposalBundle{}, err
	}
	assets, err := DetectAssets(root)
	if err != nil {
		return ProposalBundle{}, err
	}
	risks := detectRiskSignals(root, assets)
	mode := inferModeFromAssets(assets)
	conflicts := detectConflicts(assets)
	failurePreview := inferFailureRouting(conflicts, risks)

	return ProposalBundle{
		DetectedAssets:        assets,
		DetectedRisks:         risks,
		InferredMode:          mode,
		InferredManagedScope:  InferManagedScope(assets),
		InferredCapabilities:  InferCapabilities(assets),
		InferredGuidance:      InferGuidanceStrength(assets),
		InferredDefaultPolicy: InferDefaultPolicy(mode, assets),
		InferredFailure:       failurePreview,
		Conflicts:             conflicts,
		SuggestedNextActions:  suggestNextActions(mode, len(conflicts) > 0),
	}, nil
}

// BuildOnboardingProposalFromDetectedAssets builds a proposal without filesystem access.
// It is intended for deterministic fixture/testing paths where detected assets are given.
func BuildOnboardingProposalFromDetectedAssets(detectedAssets []string) ProposalBundle {
	assets := append([]string(nil), detectedAssets...)
	sort.Strings(assets)
	risks := []string{}
	mode := inferModeFromAssets(assets)
	conflicts := detectConflicts(assets)
	failurePreview := inferFailureRouting(conflicts, risks)

	return ProposalBundle{
		DetectedAssets:        assets,
		DetectedRisks:         risks,
		InferredMode:          mode,
		InferredManagedScope:  InferManagedScope(assets),
		InferredCapabilities:  InferCapabilities(assets),
		InferredGuidance:      InferGuidanceStrength(assets),
		InferredDefaultPolicy: InferDefaultPolicy(mode, assets),
		InferredFailure:       failurePreview,
		Conflicts:             conflicts,
		SuggestedNextActions:  suggestNextActions(mode, len(conflicts) > 0),
	}
}

func hasProjectMarker(dir string) bool {
	markers := []string{
		".git",
		"go.mod",
		"AGENTS.md",
		"package.json",
		"pyproject.toml",
		"Cargo.toml",
	}
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

func inferModeFromAssets(assets []string) string {
	assetSet := asSet(assets)
	score := 0
	if hasAny(assetSet, "AGENTS.md") {
		score += 2
	}
	if hasAny(assetSet, ".cursor", ".cursor/rules") {
		score += 2
	}
	if hasAny(assetSet, ".github/workflows") {
		score += 2
	}
	if hasAny(assetSet, "scripts") {
		score++
	}
	if hasAny(assetSet, "docs") {
		score++
	}
	if hasAny(assetSet, "package.json", "pyproject.toml", "Cargo.toml") {
		score++
	}
	if score >= 3 {
		return ModeBrownfield
	}
	return ModeNewProject
}

func detectConflicts(assets []string) []string {
	assetSet := asSet(assets)
	conflicts := make([]string, 0)
	if hasAny(assetSet, "AGENTS.md") && hasAny(assetSet, ".cursor", ".cursor/rules") {
		conflicts = append(conflicts, "possible duplicate guidance: agents_md (e.g., AGENTS.md) and ide_rules (e.g., .cursor/rules)")
	}
	languages := 0
	if hasAny(assetSet, "package.json") {
		languages++
	}
	if hasAny(assetSet, "pyproject.toml") {
		languages++
	}
	if hasAny(assetSet, "Cargo.toml") {
		languages++
	}
	if languages >= 2 {
		conflicts = append(conflicts, "multiple runtime ecosystems detected; enforce explicit capability bindings")
	}
	sort.Strings(conflicts)
	return conflicts
}

func suggestNextActions(mode string, hasConflicts bool) []string {
	next := make([]string, 0, 4)
	if hasConflicts {
		next = append(next, "review conflicts")
	}
	if mode == ModeBrownfield {
		next = append(next, "inspect details", "accept defaults", "export proposal only")
		return next
	}
	next = append(next, "accept defaults", "initialize canonical store", "run inspect baseline")
	return next
}

func asSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func hasAny(set map[string]struct{}, items ...string) bool {
	for _, item := range items {
		if _, ok := set[item]; ok {
			return true
		}
	}
	return false
}

func inferFailureRouting(conflicts, risks []string) FailurePreview {
	failureClass, triggers := mapConflictsToFailure(conflicts, risks)
	isDiagnosticsOnly := len(conflicts) == 0 && len(risks) == 0
	ctx := resolvefailuretier.Context{
		Scope:             "workspace",
		Triggers:          triggers,
		IsDiagnosticsOnly: isDiagnosticsOnly,
	}
	out := resolvefailuretier.ResolveFailureTier(failureClass, ctx)
	decision, ok := out.Decision.(resolvefailuretier.FailureDecision)
	if !ok {
		return FailurePreview{
			FailureClass:      failureClass,
			Scope:             "workspace",
			Triggers:          triggers,
			DefaultTier:       resolvefailuretier.TierWarnOnly,
			ResolvedTier:      resolvefailuretier.TierWarnOnly,
			StrictTransition:  "none",
			NextAllowedAction: "inspect",
		}
	}
	return FailurePreview{
		FailureClass:        decision.FailureClass,
		Scope:               decision.Scope,
		Triggers:            triggers,
		DefaultTier:         decision.DefaultTier,
		ResolvedTier:        decision.ResolvedTier,
		StrictTransition:    decision.StrictTransition,
		RequiresHumanReview: decision.RequiresHumanReview,
		ExecutionAllowed:    decision.ExecutionAllowed,
		ProjectionAllowed:   decision.ProjectionAllowed,
		NextAllowedAction:   out.NextAllowedAction,
	}
}

func mapConflictsToFailure(conflicts, risks []string) (string, []string) {
	if len(conflicts) == 0 && len(risks) == 0 {
		return "projection_failure", []string{}
	}

	triggers := make([]string, 0, 2)
	hasGuidanceConflict := false
	hasCapabilityConflict := false
	hasPolicyRisk := false

	for _, c := range conflicts {
		lc := strings.ToLower(c)
		if strings.Contains(lc, "duplicate guidance") {
			hasGuidanceConflict = true
			triggers = append(triggers, "instruction_conflict", "policy_ambiguity")
		}
		if strings.Contains(lc, "runtime ecosystems") {
			hasCapabilityConflict = true
			triggers = append(triggers, "unresolved_capability")
		}
	}
	for _, r := range risks {
		lr := strings.ToLower(r)
		if strings.Contains(lr, "destructive_script_candidate") {
			hasPolicyRisk = true
			triggers = append(triggers, "policy_ambiguity")
		}
		if strings.Contains(lr, "external_send_candidate") {
			hasCapabilityConflict = true
			triggers = append(triggers, "unresolved_capability")
		}
	}

	if len(triggers) == 0 {
		triggers = append(triggers, "policy_ambiguity")
	}
	triggers = dedupeAndSort(triggers)

	if hasGuidanceConflict {
		return "legacy_conflict_failure", triggers
	}
	if hasPolicyRisk {
		return "policy_failure", triggers
	}
	if hasCapabilityConflict {
		return "capability_resolution_failure", triggers
	}
	return "legacy_conflict_failure", triggers
}

func dedupeAndSort(items []string) []string {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func detectRiskSignals(root string, assets []string) []string {
	riskSet := map[string]struct{}{}
	assetSet := asSet(assets)

	if hasAny(assetSet, "package.json") {
		if b, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
			if detectPackageJSONRiskSignals(b, riskSet) {
				riskSet["script_surface_present"] = struct{}{}
			}
		}
	}
	if hasAny(assetSet, ".github/workflows") {
		detectTextRiskInDir(filepath.Join(root, ".github", "workflows"), riskSet)
	}
	if hasAny(assetSet, "scripts") {
		detectTextRiskInDir(filepath.Join(root, "scripts"), riskSet)
	}

	out := make([]string, 0, len(riskSet))
	for risk := range riskSet {
		out = append(out, risk)
	}
	sort.Strings(out)
	return out
}

func detectPackageJSONRiskSignals(content []byte, riskSet map[string]struct{}) bool {
	type pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	var p pkg
	if err := json.Unmarshal(content, &p); err != nil {
		return false
	}
	if len(p.Scripts) == 0 {
		return false
	}
	hit := false
	for _, script := range p.Scripts {
		hit = true
		lc := strings.ToLower(script)
		if strings.Contains(lc, "rm -rf") || strings.Contains(lc, "rmdir /s /q") {
			riskSet["destructive_script_candidate"] = struct{}{}
		}
		if containsNetworkSendSignal(lc) {
			riskSet["external_send_candidate"] = struct{}{}
		}
	}
	return hit
}

func detectTextRiskInDir(dir string, riskSet map[string]struct{}) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".yml" && ext != ".yaml" && ext != ".sh" && ext != ".ps1" && ext != ".txt" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lc := strings.ToLower(string(b))
		if strings.Contains(lc, "rm -rf") || strings.Contains(lc, "rmdir /s /q") {
			riskSet["destructive_script_candidate"] = struct{}{}
		}
		if containsNetworkSendSignal(lc) {
			riskSet["external_send_candidate"] = struct{}{}
		}
		if strings.Contains(lc, "secrets.") && strings.Contains(lc, "echo") {
			riskSet["secrets_exposure_candidate"] = struct{}{}
		}
		return nil
	})
}

func containsNetworkSendSignal(content string) bool {
	return (strings.Contains(content, "curl ") || strings.Contains(content, "wget ") || strings.Contains(content, "http://") || strings.Contains(content, "https://")) &&
		(strings.Contains(content, "api") || strings.Contains(content, "upload") || strings.Contains(content, "post") || strings.Contains(content, "send"))
}

// MustMode validates the supplied mode and returns a normalized value.
func MustMode(mode string) (string, error) {
	switch mode {
	case ModeNewProject, ModeBrownfield:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported onboarding mode %q", mode)
	}
}
