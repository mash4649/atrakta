package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/util"
)

type Contract struct {
	V           int          `json:"v"`
	ProjectID   string       `json:"project_id"`
	Interfaces  Interfaces   `json:"interfaces"`
	Boundary    Boundary     `json:"boundary"`
	Tools       Tools        `json:"tools"`
	TokenBudget TokenBudget  `json:"token_budget"`
	Hints       *Hints       `json:"hints,omitempty"`
	Quality     *Quality     `json:"quality,omitempty"`
	Autonomy    *Autonomy    `json:"autonomy,omitempty"`
	Projections *Projections `json:"projections,omitempty"`
	Routing     *Routing     `json:"routing,omitempty"`
	Context     *Context     `json:"context,omitempty"`
	Security    *Security    `json:"security,omitempty"`
	EditSafety  *EditSafety  `json:"edit_safety,omitempty"`
	Policies    *Policies    `json:"policies,omitempty"`
	Parity      *Parity      `json:"parity,omitempty"`
	Extensions  *Extensions  `json:"extensions,omitempty"`
}

type Interfaces struct {
	Supported          []string `json:"supported"`
	Fallback           string   `json:"fallback"`
	CoreSet            []string `json:"core_set"`
	PruneUnusedDefault bool     `json:"prune_unused_default"`
}

type Boundary struct {
	Include     []string `json:"include"`
	Exclude     []string `json:"exclude"`
	ManagedRoot string   `json:"managed_root"`
}

type Tools struct {
	Allow               []string `json:"allow"`
	Deny                []string `json:"deny"`
	ApprovalRequiredFor []string `json:"approval_required_for"`
}

type TokenBudget struct {
	Soft int `json:"soft"`
	Hard int `json:"hard"`
}

type Hints struct {
	Prefer            []string          `json:"prefer,omitempty"`
	Avoid             []string          `json:"avoid,omitempty"`
	Anchors           map[string]string `json:"anchors,omitempty"`
	DisableInterfaces []string          `json:"disable_interfaces,omitempty"`
}

type Quality struct {
	QuickChecks []string `json:"quick_checks,omitempty"`
	HeavyChecks []string `json:"heavy_checks,omitempty"`
	EnableHeavy bool     `json:"enable_heavy,omitempty"`
}

type Autonomy struct {
	Subworkers *SubworkerPolicy `json:"subworkers,omitempty"`
	Git        *GitPolicy       `json:"git,omitempty"`
}

type SubworkerPolicy struct {
	Mode               string `json:"mode,omitempty"`
	MaxWorkers         int    `json:"max_workers,omitempty"`
	TimeoutMs          int    `json:"timeout_ms,omitempty"`
	RetryLimit         int    `json:"retry_limit,omitempty"`
	MaxDigestChars     int    `json:"max_digest_chars,omitempty"`
	MaxOutputChars     int    `json:"max_output_chars,omitempty"`
	AutoMinProjections int    `json:"auto_min_projections,omitempty"`
	AutoMinScopes      int    `json:"auto_min_scopes,omitempty"`
	BranchMode         string `json:"branch_mode,omitempty"`
	BranchAutoMinTasks int    `json:"branch_auto_min_tasks,omitempty"`
	BranchPrefix       string `json:"branch_prefix,omitempty"`
}

type GitPolicy struct {
	Mode string `json:"mode,omitempty"`
}

type Projections struct {
	OptionalTemplates map[string][]string `json:"optional_templates,omitempty"`
	MaxPerInterface   int                 `json:"max_per_interface,omitempty"`
}

type Routing struct {
	Categories map[string]RoutingRule `json:"categories,omitempty"`
	Default    RoutingRule            `json:"default"`
}

type RoutingRule struct {
	Worker  string `json:"worker"`
	Quality string `json:"quality"`
}

type Context struct {
	Resolution          string   `json:"resolution,omitempty"`
	Projection          string   `json:"projection,omitempty"`
	MaxImportDepth      int      `json:"max_import_depth,omitempty"`
	Conventions         []string `json:"conventions,omitempty"`
	ConventionsReadOnly *bool    `json:"conventions_read_only,omitempty"`
	RepoMapTokens       int      `json:"repo_map_tokens,omitempty"`
	RepoMapRefreshSec   int      `json:"repo_map_refresh_seconds,omitempty"`
}

type Security struct {
	Profile string `json:"profile,omitempty"`
}

type EditSafety struct {
	Mode      string            `json:"mode,omitempty"`
	Anchors   *EditSafetyAnchor `json:"anchors,omitempty"`
	Languages map[string]string `json:"languages,omitempty"`
}

type EditSafetyAnchor struct {
	Normalization string `json:"normalization,omitempty"`
	WindowLines   int    `json:"window_lines,omitempty"`
}

type Policies struct {
	PromptMin *PromptMinRef `json:"prompt_min,omitempty"`
}

type PromptMinRef struct {
	Ref      string `json:"ref"`
	Required bool   `json:"required,omitempty"`
	Apply    string `json:"apply,omitempty"`
}

type Parity struct {
	V                  int                `json:"v"`
	CanonicalSources   []string           `json:"canonical_sources"`
	InstructionSurface InstructionSurface `json:"instruction_surface"`
	ApprovalSurface    ApprovalSurface    `json:"approval_surface"`
	OutputSurface      OutputSurface      `json:"output_surface"`
	ExecutionSurface   ExecutionSurface   `json:"execution_surface"`
	QualitySurface     QualitySurface     `json:"quality_surface"`
	SafetySurface      SafetySurface      `json:"safety_surface"`
	RoutingSurface     RoutingSurface     `json:"routing_surface"`
	ProjectionSurface  ProjectionSurface  `json:"projection_surface"`
}

type InstructionSurface struct {
	Source          string `json:"source,omitempty"`
	PromptPolicyRef string `json:"prompt_policy_ref,omitempty"`
}

type ApprovalSurface struct {
	ApprovalRequiredForRef string `json:"approval_required_for_ref,omitempty"`
}

type OutputSurface struct {
	PlanFormat  string `json:"plan_format,omitempty"`
	ErrorFormat string `json:"error_format,omitempty"`
}

type ExecutionSurface struct {
	PathPolicy string `json:"path_policy,omitempty"`
	LatestOnly bool   `json:"latest_only,omitempty"`
}

type QualitySurface struct {
	QuickChecksRef string `json:"quick_checks_ref,omitempty"`
	HeavyChecksRef string `json:"heavy_checks_ref,omitempty"`
}

type SafetySurface struct {
	SecurityProfileRef string `json:"security_profile_ref,omitempty"`
}

type RoutingSurface struct {
	CategoriesRef string `json:"categories_ref,omitempty"`
	DefaultRef    string `json:"default_ref,omitempty"`
}

type ProjectionSurface struct {
	Deterministic     bool `json:"deterministic,omitempty"`
	ManagedOnlyRepair bool `json:"managed_only_repair,omitempty"`
}

type Extensions struct {
	V         int              `json:"v"`
	MergeMode string           `json:"merge_mode,omitempty"`
	Agents    *AgentsExtension `json:"agents,omitempty"`
	MCP       []ExtensionEntry `json:"mcp,omitempty"`
	Plugins   []ExtensionEntry `json:"plugins,omitempty"`
	Skills    []ExtensionEntry `json:"skills,omitempty"`
	Workflows []ExtensionEntry `json:"workflows,omitempty"`
	Hooks     *HooksExtension  `json:"hooks,omitempty"`
}

type AgentsExtension struct {
	Mode       string `json:"mode,omitempty"`
	AppendFile string `json:"append_file,omitempty"`
}

type ExtensionEntry struct {
	ID      string `json:"id"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type HooksExtension struct {
	Shell    *ShellHooks    `json:"shell,omitempty"`
	Git      *GitHooks      `json:"git,omitempty"`
	IDE      *IDEHooks      `json:"ide,omitempty"`
	Workflow *WorkflowHooks `json:"workflow,omitempty"`
}

type ShellHooks struct {
	OnCD   *bool `json:"on_cd,omitempty"`
	OnExec *bool `json:"on_exec,omitempty"`
}

type GitHooks struct {
	PreCommit *bool `json:"pre_commit,omitempty"`
	PrePush   *bool `json:"pre_push,omitempty"`
}

type IDEHooks struct {
	OnOpen *bool `json:"on_open,omitempty"`
}

type WorkflowHooks struct {
	BeforeStart *bool `json:"before_start,omitempty"`
	AfterApply  *bool `json:"after_apply,omitempty"`
}

func LoadOrInit(repoRoot string) (Contract, []byte, error) {
	path := filepath.Join(repoRoot, ".atrakta", "contract.json")
	b, err := os.ReadFile(path)
	if err == nil {
		var c Contract
		if err := json.Unmarshal(b, &c); err != nil {
			return Contract{}, nil, fmt.Errorf("parse contract: %w", err)
		}
		if err := Validate(c); err != nil {
			return Contract{}, nil, err
		}
		return c, b, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Contract{}, nil, fmt.Errorf("read contract: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Contract{}, nil, fmt.Errorf("mkdir .atrakta: %w", err)
	}
	c := Default(repoRoot)
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Contract{}, nil, fmt.Errorf("marshal default contract: %w", err)
	}
	out = append(out, '\n')
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "contract.json.lock")
	if err := util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(path, out, 0o644)
	}); err != nil {
		return Contract{}, nil, fmt.Errorf("write default contract: %w", err)
	}
	return c, out, nil
}

func Save(repoRoot string, c Contract) ([]byte, error) {
	if err := Validate(c); err != nil {
		return nil, err
	}
	path := filepath.Join(repoRoot, ".atrakta", "contract.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir .atrakta: %w", err)
	}
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal contract: %w", err)
	}
	out = append(out, '\n')
	lockPath := filepath.Join(repoRoot, ".atrakta", ".locks", "contract.json.lock")
	if err := util.WithFileLock(lockPath, util.DefaultFileLockOptions(), func() error {
		return util.AtomicWriteFile(path, out, 0o644)
	}); err != nil {
		return nil, fmt.Errorf("write contract: %w", err)
	}
	return out, nil
}

func Default(repoRoot string) Contract {
	projectID := filepath.Base(repoRoot)
	if projectID == "." || projectID == string(filepath.Separator) || projectID == "" {
		projectID = "atrakta-project"
	}
	return Contract{
		V:         1,
		ProjectID: projectID,
		Interfaces: Interfaces{
			Supported: []string{
				"vscode", "cursor", "windsurf", "trae", "antigravity",
				"aider", "codex_cli", "gemini_cli", "claude_code", "opencode", "github_copilot",
			},
			Fallback:           "core",
			CoreSet:            []string{"cursor"},
			PruneUnusedDefault: true,
		},
		Boundary: Boundary{
			Include:     []string{""},
			Exclude:     []string{".atrakta/"},
			ManagedRoot: ".atrakta/",
		},
		Tools: Tools{
			Allow:               []string{"create", "edit", "run"},
			Deny:                []string{},
			ApprovalRequiredFor: []string{"boundary_expand", "external_side_effect", "destructive_prune"},
		},
		TokenBudget: TokenBudget{Soft: 8000, Hard: 16000},
		Quality: &Quality{
			QuickChecks: []string{"verify_loop"},
			HeavyChecks: []string{"verify_perf_gate"},
			EnableHeavy: true,
		},
		Projections: &Projections{MaxPerInterface: 3},
		Routing: &Routing{
			Categories: map[string]RoutingRule{
				"sync":   {Worker: "sync_safe", Quality: "quick"},
				"edit":   {Worker: "code_edit_fast", Quality: "quick"},
				"verify": {Worker: "ci_strict", Quality: "heavy"},
			},
			Default: RoutingRule{Worker: "general", Quality: "quick"},
		},
		Context: &Context{
			Resolution:     "nearest_with_import",
			Projection:     "interface_only",
			MaxImportDepth: 6,
		},
		Security: &Security{
			Profile: "workspace_write",
		},
		EditSafety: &EditSafety{
			Mode: "anchor+optional_ast",
			Anchors: &EditSafetyAnchor{
				Normalization: "ws+eol+unicode_nfc",
				WindowLines:   20,
			},
			Languages: map[string]string{
				"go":   "ast",
				"json": "parse",
			},
		},
		Policies: &Policies{
			PromptMin: &PromptMinRef{
				Ref:      ".atrakta/policies/prompt-min.json",
				Required: false,
				Apply:    "conditional",
			},
		},
		Parity: &Parity{
			V:                1,
			CanonicalSources: []string{".atrakta/contract.json", ".atrakta/state.json", ".atrakta/events.jsonl"},
			InstructionSurface: InstructionSurface{
				Source:          "contract_and_policy",
				PromptPolicyRef: "policies.prompt_min.ref",
			},
			ApprovalSurface: ApprovalSurface{
				ApprovalRequiredForRef: "tools.approval_required_for",
			},
			OutputSurface: OutputSurface{
				PlanFormat:  "markdown",
				ErrorFormat: "structured",
			},
			ExecutionSurface: ExecutionSurface{
				PathPolicy: "fast_first_strict_on_demand",
				LatestOnly: true,
			},
			QualitySurface: QualitySurface{
				QuickChecksRef: "quality.quick_checks",
				HeavyChecksRef: "quality.heavy_checks",
			},
			SafetySurface: SafetySurface{
				SecurityProfileRef: "security.profile",
			},
			RoutingSurface: RoutingSurface{
				CategoriesRef: "routing.categories",
				DefaultRef:    "routing.default",
			},
			ProjectionSurface: ProjectionSurface{
				Deterministic:     true,
				ManagedOnlyRepair: true,
			},
		},
		Extensions: &Extensions{
			V:         1,
			MergeMode: "append-first",
			Agents: &AgentsExtension{
				Mode:       "append",
				AppendFile: ".atrakta/AGENTS.append.md",
			},
			Hooks: &HooksExtension{},
		},
	}
}

func Validate(c Contract) error {
	if c.V != 1 {
		return fmt.Errorf("contract.v must be 1")
	}
	if c.ProjectID == "" {
		return fmt.Errorf("contract.project_id required")
	}
	if len(c.Interfaces.Supported) == 0 {
		return fmt.Errorf("interfaces.supported required")
	}
	if len(c.Interfaces.CoreSet) == 0 {
		return fmt.Errorf("interfaces.core_set required")
	}
	sup := map[string]struct{}{}
	for _, s := range c.Interfaces.Supported {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("interfaces.supported contains empty")
		}
		sup[s] = struct{}{}
	}
	for _, cset := range c.Interfaces.CoreSet {
		if _, ok := sup[cset]; !ok {
			return fmt.Errorf("interfaces.core_set contains unsupported id %q", cset)
		}
	}
	if c.Interfaces.Fallback != "core" && c.Interfaces.Fallback != "all" {
		if _, ok := sup[c.Interfaces.Fallback]; !ok {
			return fmt.Errorf("interfaces.fallback %q not in supported", c.Interfaces.Fallback)
		}
	}
	if len(c.Boundary.Include) == 0 {
		return fmt.Errorf("boundary.include required")
	}
	if c.Boundary.ManagedRoot == "" {
		return fmt.Errorf("boundary.managed_root required")
	}
	if !strings.HasSuffix(c.Boundary.ManagedRoot, "/") {
		c.Boundary.ManagedRoot += "/"
	}
	if c.TokenBudget.Soft <= 0 || c.TokenBudget.Hard <= 0 || c.TokenBudget.Soft > c.TokenBudget.Hard {
		return fmt.Errorf("invalid token_budget soft/hard")
	}
	if c.Hints != nil {
		for _, id := range c.Hints.DisableInterfaces {
			if _, ok := sup[id]; !ok {
				return fmt.Errorf("hints.disable_interfaces contains unsupported id %q", id)
			}
		}
		for _, id := range c.Hints.Prefer {
			if _, ok := sup[id]; !ok {
				return fmt.Errorf("hints.prefer contains unsupported id %q", id)
			}
		}
		for _, id := range c.Hints.Avoid {
			if _, ok := sup[id]; !ok {
				return fmt.Errorf("hints.avoid contains unsupported id %q", id)
			}
		}
		for id := range c.Hints.Anchors {
			if _, ok := sup[id]; !ok {
				return fmt.Errorf("hints.anchors contains unsupported id %q", id)
			}
		}
	}
	if c.Projections != nil {
		if c.Projections.MaxPerInterface < 0 {
			return fmt.Errorf("projections.max_per_interface cannot be negative")
		}
		for id, templates := range c.Projections.OptionalTemplates {
			if _, ok := sup[id]; !ok {
				return fmt.Errorf("projections.optional_templates contains unsupported id %q", id)
			}
			for _, t := range templates {
				switch t {
				case "contract-json", "atrakta-link":
				default:
					return fmt.Errorf("unsupported optional template %q for interface %q", t, id)
				}
			}
		}
	}
	if c.Autonomy != nil && c.Autonomy.Subworkers != nil {
		sw := c.Autonomy.Subworkers
		mode := strings.TrimSpace(strings.ToLower(sw.Mode))
		if mode != "" && mode != "off" && mode != "auto" && mode != "on" {
			return fmt.Errorf("autonomy.subworkers.mode must be off|auto|on")
		}
		if sw.MaxWorkers < 0 {
			return fmt.Errorf("autonomy.subworkers.max_workers cannot be negative")
		}
		if sw.TimeoutMs < 0 {
			return fmt.Errorf("autonomy.subworkers.timeout_ms cannot be negative")
		}
		if sw.RetryLimit < 0 {
			return fmt.Errorf("autonomy.subworkers.retry_limit cannot be negative")
		}
		if sw.MaxDigestChars < 0 {
			return fmt.Errorf("autonomy.subworkers.max_digest_chars cannot be negative")
		}
		if sw.MaxOutputChars < 0 {
			return fmt.Errorf("autonomy.subworkers.max_output_chars cannot be negative")
		}
		if sw.AutoMinProjections < 0 {
			return fmt.Errorf("autonomy.subworkers.auto_min_projections cannot be negative")
		}
		if sw.AutoMinScopes < 0 {
			return fmt.Errorf("autonomy.subworkers.auto_min_scopes cannot be negative")
		}
		branchMode := strings.TrimSpace(strings.ToLower(sw.BranchMode))
		if branchMode != "" && branchMode != "off" && branchMode != "auto" && branchMode != "on" {
			return fmt.Errorf("autonomy.subworkers.branch_mode must be off|auto|on")
		}
		if sw.BranchAutoMinTasks < 0 {
			return fmt.Errorf("autonomy.subworkers.branch_auto_min_tasks cannot be negative")
		}
	}
	if c.Autonomy != nil && c.Autonomy.Git != nil {
		mode := strings.TrimSpace(strings.ToLower(c.Autonomy.Git.Mode))
		if mode != "" && mode != "off" && mode != "auto" && mode != "on" {
			return fmt.Errorf("autonomy.git.mode must be off|auto|on")
		}
	}
	if c.Routing != nil {
		if err := validateRoutingRule(c.Routing.Default, "routing.default"); err != nil {
			return err
		}
		for category, rule := range c.Routing.Categories {
			if strings.TrimSpace(category) == "" {
				return fmt.Errorf("routing.categories contains empty key")
			}
			if err := validateRoutingRule(rule, fmt.Sprintf("routing.categories[%q]", category)); err != nil {
				return err
			}
		}
	}
	if c.Context != nil {
		resolution := strings.TrimSpace(strings.ToLower(c.Context.Resolution))
		if resolution != "" && resolution != "nearest_with_import" {
			return fmt.Errorf("context.resolution must be nearest_with_import")
		}
		projection := strings.TrimSpace(strings.ToLower(c.Context.Projection))
		if projection != "" && projection != "interface_only" {
			return fmt.Errorf("context.projection must be interface_only")
		}
		if c.Context.MaxImportDepth < 0 {
			return fmt.Errorf("context.max_import_depth cannot be negative")
		}
		if c.Context.ConventionsReadOnly != nil && !*c.Context.ConventionsReadOnly {
			return fmt.Errorf("context.conventions_read_only must be true")
		}
		for _, p := range c.Context.Conventions {
			if filepath.IsAbs(p) {
				return fmt.Errorf("context.conventions must be repo-relative")
			}
			n := util.NormalizeRelPath(p)
			if n == "" || strings.HasPrefix(n, "../") {
				return fmt.Errorf("context.conventions must be repo-relative")
			}
		}
		if c.Context.RepoMapTokens < 0 {
			return fmt.Errorf("context.repo_map_tokens cannot be negative")
		}
		if c.Context.RepoMapRefreshSec < 0 {
			return fmt.Errorf("context.repo_map_refresh_seconds cannot be negative")
		}
	}
	if c.Security != nil {
		profile := strings.TrimSpace(strings.ToLower(c.Security.Profile))
		if profile != "" && profile != "read_only" && profile != "workspace_write" && profile != "full" {
			return fmt.Errorf("security.profile must be read_only|workspace_write|full")
		}
	}
	if c.EditSafety != nil {
		mode := strings.TrimSpace(strings.ToLower(c.EditSafety.Mode))
		if mode != "" && mode != "anchor+optional_ast" {
			return fmt.Errorf("edit_safety.mode must be anchor+optional_ast")
		}
		if c.EditSafety.Anchors != nil {
			n := strings.TrimSpace(strings.ToLower(c.EditSafety.Anchors.Normalization))
			if n != "" && n != "ws+eol+unicode_nfc" {
				return fmt.Errorf("edit_safety.anchors.normalization must be ws+eol+unicode_nfc")
			}
			if c.EditSafety.Anchors.WindowLines < 0 {
				return fmt.Errorf("edit_safety.anchors.window_lines cannot be negative")
			}
		}
		for lang, policy := range c.EditSafety.Languages {
			if strings.TrimSpace(lang) == "" {
				return fmt.Errorf("edit_safety.languages contains empty key")
			}
			p := strings.TrimSpace(strings.ToLower(policy))
			if p != "" && p != "off" && p != "ast" && p != "parse" {
				return fmt.Errorf("edit_safety.languages[%q] must be off|ast|parse", lang)
			}
		}
	}
	if c.Policies != nil && c.Policies.PromptMin != nil {
		pm := c.Policies.PromptMin
		if strings.TrimSpace(pm.Ref) == "" {
			return fmt.Errorf("policies.prompt_min.ref required")
		}
		if filepath.IsAbs(pm.Ref) {
			return fmt.Errorf("policies.prompt_min.ref must be repo-relative")
		}
		normalized := util.NormalizeRelPath(pm.Ref)
		if normalized == "" || strings.HasPrefix(normalized, "../") {
			return fmt.Errorf("policies.prompt_min.ref must be repo-relative")
		}
		apply := strings.TrimSpace(strings.ToLower(pm.Apply))
		if apply != "" && apply != "conditional" {
			return fmt.Errorf("policies.prompt_min.apply must be conditional")
		}
	}
	if c.Parity != nil {
		if err := validateParity(c); err != nil {
			return err
		}
	}
	if c.Extensions != nil {
		if err := validateExtensions(c.Extensions); err != nil {
			return err
		}
	}
	return nil
}

func ContractHash(b []byte) string {
	return util.SHA256Tagged(b)
}

func SupportedSet(c Contract) map[string]struct{} {
	m := make(map[string]struct{}, len(c.Interfaces.Supported))
	for _, s := range c.Interfaces.Supported {
		m[s] = struct{}{}
	}
	return m
}

func HasApprovalRequirement(c Contract, key string) bool {
	for _, k := range c.Tools.ApprovalRequiredFor {
		if k == key {
			return true
		}
	}
	return false
}

func CanonicalizeBoundary(c Contract) Contract {
	c2 := c
	norm := func(items []string) []string {
		out := make([]string, 0, len(items))
		for _, p := range items {
			n := util.NormalizeRelPath(p)
			if n != "" && !strings.HasSuffix(n, "/") {
				n += "/"
			}
			out = append(out, n)
		}
		sort.Strings(out)
		return out
	}
	c2.Boundary.Include = norm(c.Boundary.Include)
	c2.Boundary.Exclude = norm(c.Boundary.Exclude)
	if mr := util.NormalizeRelPath(c.Boundary.ManagedRoot); mr != "" {
		if !strings.HasSuffix(mr, "/") {
			mr += "/"
		}
		c2.Boundary.ManagedRoot = mr
	}
	if c2.Projections != nil && c2.Projections.MaxPerInterface == 0 {
		c2.Projections.MaxPerInterface = 3
	}
	if c2.Routing != nil {
		c2.Routing.Default.Quality = normalizeQuality(c2.Routing.Default.Quality)
		for category, rule := range c2.Routing.Categories {
			rule.Quality = normalizeQuality(rule.Quality)
			c2.Routing.Categories[category] = rule
		}
	}
	if c2.Security != nil {
		c2.Security.Profile = ResolveSecurityProfile(c2)
	}
	if c2.Context != nil {
		c2.Context.Resolution = strings.TrimSpace(strings.ToLower(c2.Context.Resolution))
		c2.Context.Projection = strings.TrimSpace(strings.ToLower(c2.Context.Projection))
		if c2.Context.Resolution == "" {
			c2.Context.Resolution = "nearest_with_import"
		}
		if c2.Context.Projection == "" {
			c2.Context.Projection = "interface_only"
		}
		if c2.Context.MaxImportDepth == 0 {
			c2.Context.MaxImportDepth = 6
		}
		c2.Context.Conventions = normalizeConventions(c2.Context.Conventions)
		if len(c2.Context.Conventions) == 0 {
			c2.Context.Conventions = []string{"CONVENTIONS.md", "docs/CONVENTIONS.md"}
		}
		if c2.Context.ConventionsReadOnly == nil {
			ro := true
			c2.Context.ConventionsReadOnly = &ro
		}
		if c2.Context.RepoMapTokens == 0 {
			c2.Context.RepoMapTokens = 1200
		}
		if c2.Context.RepoMapRefreshSec == 0 {
			c2.Context.RepoMapRefreshSec = 300
		}
	}
	if c2.EditSafety != nil {
		c2.EditSafety.Mode = strings.TrimSpace(strings.ToLower(c2.EditSafety.Mode))
		if c2.EditSafety.Mode == "" {
			c2.EditSafety.Mode = "anchor+optional_ast"
		}
		if c2.EditSafety.Anchors != nil {
			c2.EditSafety.Anchors.Normalization = strings.TrimSpace(strings.ToLower(c2.EditSafety.Anchors.Normalization))
			if c2.EditSafety.Anchors.Normalization == "" {
				c2.EditSafety.Anchors.Normalization = "ws+eol+unicode_nfc"
			}
			if c2.EditSafety.Anchors.WindowLines == 0 {
				c2.EditSafety.Anchors.WindowLines = 20
			}
		}
		if c2.EditSafety.Languages == nil {
			c2.EditSafety.Languages = map[string]string{}
		}
		if _, ok := c2.EditSafety.Languages["go"]; !ok {
			c2.EditSafety.Languages["go"] = "ast"
		}
		if _, ok := c2.EditSafety.Languages["json"]; !ok {
			c2.EditSafety.Languages["json"] = "parse"
		}
		normLang := make(map[string]string, len(c2.EditSafety.Languages))
		for k, v := range c2.EditSafety.Languages {
			key := strings.TrimSpace(strings.ToLower(k))
			if key == "" {
				continue
			}
			val := strings.TrimSpace(strings.ToLower(v))
			if val == "" {
				val = "off"
			}
			normLang[key] = val
		}
		c2.EditSafety.Languages = normLang
	}
	if c2.Policies != nil && c2.Policies.PromptMin != nil {
		c2.Policies.PromptMin.Ref = util.NormalizeRelPath(c2.Policies.PromptMin.Ref)
		if c2.Policies.PromptMin.Apply == "" {
			c2.Policies.PromptMin.Apply = "conditional"
		}
	}
	if c2.Parity != nil {
		if c2.Parity.V == 0 {
			c2.Parity.V = 1
		}
		if len(c2.Parity.CanonicalSources) == 0 {
			c2.Parity.CanonicalSources = []string{".atrakta/contract.json", ".atrakta/state.json", ".atrakta/events.jsonl"}
		}
		c2.Parity.InstructionSurface.Source = strings.TrimSpace(strings.ToLower(c2.Parity.InstructionSurface.Source))
		if c2.Parity.InstructionSurface.Source == "" {
			c2.Parity.InstructionSurface.Source = "contract_and_policy"
		}
		if c2.Parity.InstructionSurface.PromptPolicyRef == "" {
			c2.Parity.InstructionSurface.PromptPolicyRef = "policies.prompt_min.ref"
		}
		if c2.Parity.ApprovalSurface.ApprovalRequiredForRef == "" {
			c2.Parity.ApprovalSurface.ApprovalRequiredForRef = "tools.approval_required_for"
		}
		c2.Parity.OutputSurface.PlanFormat = strings.TrimSpace(strings.ToLower(c2.Parity.OutputSurface.PlanFormat))
		if c2.Parity.OutputSurface.PlanFormat == "" {
			c2.Parity.OutputSurface.PlanFormat = "markdown"
		}
		c2.Parity.OutputSurface.ErrorFormat = strings.TrimSpace(strings.ToLower(c2.Parity.OutputSurface.ErrorFormat))
		if c2.Parity.OutputSurface.ErrorFormat == "" {
			c2.Parity.OutputSurface.ErrorFormat = "structured"
		}
		c2.Parity.ExecutionSurface.PathPolicy = strings.TrimSpace(strings.ToLower(c2.Parity.ExecutionSurface.PathPolicy))
		if c2.Parity.ExecutionSurface.PathPolicy == "" {
			c2.Parity.ExecutionSurface.PathPolicy = "fast_first_strict_on_demand"
		}
		if !c2.Parity.ExecutionSurface.LatestOnly {
			c2.Parity.ExecutionSurface.LatestOnly = true
		}
		if c2.Parity.QualitySurface.QuickChecksRef == "" {
			c2.Parity.QualitySurface.QuickChecksRef = "quality.quick_checks"
		}
		if c2.Parity.QualitySurface.HeavyChecksRef == "" {
			c2.Parity.QualitySurface.HeavyChecksRef = "quality.heavy_checks"
		}
		if c2.Parity.SafetySurface.SecurityProfileRef == "" {
			c2.Parity.SafetySurface.SecurityProfileRef = "security.profile"
		}
		if c2.Parity.RoutingSurface.CategoriesRef == "" {
			c2.Parity.RoutingSurface.CategoriesRef = "routing.categories"
		}
		if c2.Parity.RoutingSurface.DefaultRef == "" {
			c2.Parity.RoutingSurface.DefaultRef = "routing.default"
		}
		if !c2.Parity.ProjectionSurface.Deterministic {
			c2.Parity.ProjectionSurface.Deterministic = true
		}
		if !c2.Parity.ProjectionSurface.ManagedOnlyRepair {
			c2.Parity.ProjectionSurface.ManagedOnlyRepair = true
		}
	}
	if c2.Extensions != nil {
		if c2.Extensions.V == 0 {
			c2.Extensions.V = 1
		}
		c2.Extensions.MergeMode = strings.TrimSpace(strings.ToLower(c2.Extensions.MergeMode))
		if c2.Extensions.MergeMode == "" {
			c2.Extensions.MergeMode = "append-first"
		}
		if c2.Extensions.Agents == nil {
			c2.Extensions.Agents = &AgentsExtension{}
		}
		c2.Extensions.Agents.Mode = strings.TrimSpace(strings.ToLower(c2.Extensions.Agents.Mode))
		if c2.Extensions.Agents.Mode == "" {
			c2.Extensions.Agents.Mode = "append"
		}
		if c2.Extensions.Agents.AppendFile == "" {
			c2.Extensions.Agents.AppendFile = ".atrakta/AGENTS.append.md"
		} else {
			c2.Extensions.Agents.AppendFile = util.NormalizeRelPath(c2.Extensions.Agents.AppendFile)
		}
	}
	return c2
}

func validateParity(c Contract) error {
	p := c.Parity
	if p.V != 1 {
		return fmt.Errorf("parity.v must be 1")
	}
	if len(p.CanonicalSources) == 0 {
		return fmt.Errorf("parity.canonical_sources required")
	}
	for _, src := range p.CanonicalSources {
		if strings.TrimSpace(src) == "" {
			return fmt.Errorf("parity.canonical_sources contains empty value")
		}
		if filepath.IsAbs(src) {
			return fmt.Errorf("parity.canonical_sources must be repo-relative")
		}
		n := util.NormalizeRelPath(src)
		if n == "" || strings.HasPrefix(n, "../") {
			return fmt.Errorf("parity.canonical_sources must be repo-relative")
		}
	}

	if p.InstructionSurface.PromptPolicyRef != "" && p.InstructionSurface.PromptPolicyRef != "policies.prompt_min.ref" {
		return fmt.Errorf("parity.instruction_surface.prompt_policy_ref must be policies.prompt_min.ref")
	}
	if p.ApprovalSurface.ApprovalRequiredForRef != "" && p.ApprovalSurface.ApprovalRequiredForRef != "tools.approval_required_for" {
		return fmt.Errorf("parity.approval_surface.approval_required_for_ref must be tools.approval_required_for")
	}
	if p.QualitySurface.QuickChecksRef != "" && p.QualitySurface.QuickChecksRef != "quality.quick_checks" {
		return fmt.Errorf("parity.quality_surface.quick_checks_ref must be quality.quick_checks")
	}
	if p.QualitySurface.HeavyChecksRef != "" && p.QualitySurface.HeavyChecksRef != "quality.heavy_checks" {
		return fmt.Errorf("parity.quality_surface.heavy_checks_ref must be quality.heavy_checks")
	}
	if p.SafetySurface.SecurityProfileRef != "" && p.SafetySurface.SecurityProfileRef != "security.profile" {
		return fmt.Errorf("parity.safety_surface.security_profile_ref must be security.profile")
	}
	if p.RoutingSurface.CategoriesRef != "" && p.RoutingSurface.CategoriesRef != "routing.categories" {
		return fmt.Errorf("parity.routing_surface.categories_ref must be routing.categories")
	}
	if p.RoutingSurface.DefaultRef != "" && p.RoutingSurface.DefaultRef != "routing.default" {
		return fmt.Errorf("parity.routing_surface.default_ref must be routing.default")
	}
	if p.ExecutionSurface.PathPolicy != "" && p.ExecutionSurface.PathPolicy != "fast_first_strict_on_demand" {
		return fmt.Errorf("parity.execution_surface.path_policy must be fast_first_strict_on_demand")
	}
	if !p.ExecutionSurface.LatestOnly {
		return fmt.Errorf("parity.execution_surface.latest_only must be true")
	}
	if p.OutputSurface.PlanFormat != "" && p.OutputSurface.PlanFormat != "markdown" && p.OutputSurface.PlanFormat != "json" {
		return fmt.Errorf("parity.output_surface.plan_format must be markdown|json")
	}
	if p.OutputSurface.ErrorFormat != "" && p.OutputSurface.ErrorFormat != "structured" && p.OutputSurface.ErrorFormat != "plain" {
		return fmt.Errorf("parity.output_surface.error_format must be structured|plain")
	}
	if !p.ProjectionSurface.Deterministic {
		return fmt.Errorf("parity.projection_surface.deterministic must be true")
	}
	if !p.ProjectionSurface.ManagedOnlyRepair {
		return fmt.Errorf("parity.projection_surface.managed_only_repair must be true")
	}

	if p.ApprovalSurface.ApprovalRequiredForRef == "tools.approval_required_for" && len(c.Tools.ApprovalRequiredFor) == 0 {
		return fmt.Errorf("tools.approval_required_for required when parity approval surface is enabled")
	}
	if p.QualitySurface.QuickChecksRef == "quality.quick_checks" && c.Quality == nil {
		return fmt.Errorf("quality.quick_checks reference requires quality section")
	}
	if p.QualitySurface.HeavyChecksRef == "quality.heavy_checks" && c.Quality == nil {
		return fmt.Errorf("quality.heavy_checks reference requires quality section")
	}
	if p.RoutingSurface.CategoriesRef == "routing.categories" && c.Routing == nil {
		return fmt.Errorf("routing.categories reference requires routing section")
	}
	if p.SafetySurface.SecurityProfileRef == "security.profile" && c.Security == nil {
		return fmt.Errorf("security.profile reference requires security section")
	}
	if p.InstructionSurface.PromptPolicyRef == "policies.prompt_min.ref" && (c.Policies == nil || c.Policies.PromptMin == nil) {
		return fmt.Errorf("policies.prompt_min required when parity instruction surface references prompt policy")
	}
	return nil
}

func validateExtensions(e *Extensions) error {
	if e.V != 1 {
		return fmt.Errorf("extensions.v must be 1")
	}
	mode := strings.TrimSpace(strings.ToLower(e.MergeMode))
	if mode == "" {
		mode = "append-first"
	}
	switch mode {
	case "append-first", "append", "include", "replace":
	default:
		return fmt.Errorf("extensions.merge_mode must be append-first|append|include|replace")
	}
	if e.Agents != nil {
		agentsMode := strings.TrimSpace(strings.ToLower(e.Agents.Mode))
		if agentsMode == "" {
			agentsMode = "append"
		}
		switch agentsMode {
		case "append", "include", "generate":
		default:
			return fmt.Errorf("extensions.agents.mode must be append|include|generate")
		}
		if e.Agents.AppendFile != "" {
			if filepath.IsAbs(e.Agents.AppendFile) {
				return fmt.Errorf("extensions.agents.append_file must be repo-relative")
			}
			n := util.NormalizeRelPath(e.Agents.AppendFile)
			if n == "" || strings.HasPrefix(n, "../") {
				return fmt.Errorf("extensions.agents.append_file must be repo-relative")
			}
		}
	}
	if err := validateExtensionEntries("extensions.mcp", e.MCP); err != nil {
		return err
	}
	if err := validateExtensionEntries("extensions.plugins", e.Plugins); err != nil {
		return err
	}
	if err := validateExtensionEntries("extensions.skills", e.Skills); err != nil {
		return err
	}
	if err := validateExtensionEntries("extensions.workflows", e.Workflows); err != nil {
		return err
	}
	return nil
}

func validateExtensionEntries(field string, items []ExtensionEntry) error {
	seen := map[string]struct{}{}
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return fmt.Errorf("%s contains empty id", field)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%s contains duplicate id %q", field, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateRoutingRule(rule RoutingRule, field string) error {
	if strings.TrimSpace(rule.Worker) == "" {
		return fmt.Errorf("%s.worker required", field)
	}
	q := strings.TrimSpace(strings.ToLower(rule.Quality))
	if q != "" && q != "quick" && q != "heavy" {
		return fmt.Errorf("%s.quality must be quick|heavy", field)
	}
	return nil
}

func normalizeQuality(q string) string {
	switch strings.TrimSpace(strings.ToLower(q)) {
	case "heavy":
		return "heavy"
	default:
		return "quick"
	}
}

func ResolveSecurityProfile(c Contract) string {
	if c.Security == nil {
		return "workspace_write"
	}
	switch strings.TrimSpace(strings.ToLower(c.Security.Profile)) {
	case "read_only":
		return "read_only"
	case "full":
		return "full"
	default:
		return "workspace_write"
	}
}

func normalizeConventions(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, p := range in {
		n := util.NormalizeRelPath(p)
		if n == "" || strings.HasPrefix(n, "../") {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func ContextConventions(c Contract) []string {
	if c.Context == nil {
		return []string{"CONVENTIONS.md", "docs/CONVENTIONS.md"}
	}
	if len(c.Context.Conventions) == 0 {
		return []string{"CONVENTIONS.md", "docs/CONVENTIONS.md"}
	}
	return normalizeConventions(c.Context.Conventions)
}
