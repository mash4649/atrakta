package subworker

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/util"
)

const (
	modeOff  = "off"
	modeAuto = "auto"
	modeOn   = "on"
)

type Config struct {
	Mode               string `json:"mode"`
	MaxWorkers         int    `json:"max_workers"`
	TimeoutMs          int    `json:"timeout_ms"`
	RetryLimit         int    `json:"retry_limit"`
	MaxDigestChars     int    `json:"max_digest_chars"`
	MaxOutputChars     int    `json:"max_output_chars"`
	AutoMinProjections int    `json:"auto_min_projections"`
	AutoMinScopes      int    `json:"auto_min_scopes"`

	// BranchMode controls optional branch-per-worker plan lanes (Phase C scaffolding).
	// This never performs git merge/apply directly; it only plans mergeable branch lanes.
	BranchMode         string `json:"branch_mode"`
	BranchAutoMinTasks int    `json:"branch_auto_min_tasks"`
	BranchPrefix       string `json:"branch_prefix"`
}

type Decision struct {
	Mode    string         `json:"mode"`
	Enabled bool           `json:"enabled"`
	Reason  string         `json:"reason"`
	Signals map[string]any `json:"signals,omitempty"`
}

type ProposedProjection struct {
	Interface   string `json:"interface"`
	TemplateID  string `json:"template_id"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Fingerprint string `json:"fingerprint"`
}

type Task struct {
	WorkerID    string               `json:"worker_id"`
	Scope       string               `json:"scope"`
	Tools       []string             `json:"tools"`
	Constraints []string             `json:"constraints"`
	Task        string               `json:"task"`
	Digest      string               `json:"digest"`
	SerialGroup int                  `json:"serial_group"`
	Order       int                  `json:"order"`
	Proposals   []ProposedProjection `json:"proposals,omitempty"`
}

type Result struct {
	WorkerID        string `json:"worker_id"`
	Scope           string `json:"scope"`
	Status          string `json:"status"`
	ProposedOpCount int    `json:"proposed_op_count"`
	Summary         string `json:"summary"`
	TokenEstimate   int    `json:"token_estimate"`
	TimeoutMs       int    `json:"timeout_ms"`
	RetryLimit      int    `json:"retry_limit"`
}

type Metrics struct {
	Enabled             bool `json:"enabled"`
	TaskCount           int  `json:"task_count"`
	TotalTokenEstimate  int  `json:"total_token_estimate"`
	MaxDigestCharsUsed  int  `json:"max_digest_chars_used"`
	MaxSummaryCharsUsed int  `json:"max_summary_chars_used"`
}

type BudgetReport struct {
	Applied bool   `json:"applied"`
	Reason  string `json:"reason"`

	SoftLimit int `json:"soft_limit"`
	HardLimit int `json:"hard_limit"`

	BeforeTokenEstimate int `json:"before_token_estimate"`
	AfterTokenEstimate  int `json:"after_token_estimate"`

	BeforeMaxDigestChars int `json:"before_max_digest_chars"`
	AfterMaxDigestChars  int `json:"after_max_digest_chars"`

	BeforeMaxOutputChars int `json:"before_max_output_chars"`
	AfterMaxOutputChars  int `json:"after_max_output_chars"`

	Disabled bool `json:"disabled"`
}

type BranchPlan struct {
	BranchName  string `json:"branch_name"`
	WorkerID    string `json:"worker_id"`
	Scope       string `json:"scope"`
	SerialGroup int    `json:"serial_group"`
	Order       int    `json:"order"`
	Proposals   int    `json:"proposals"`
}

type QueueItem struct {
	Index       int    `json:"index"`
	Path        string `json:"path"`
	WorkerID    string `json:"worker_id"`
	Order       int    `json:"order"`
	SerialGroup int    `json:"serial_group"`
	Interface   string `json:"interface"`
	TemplateID  string `json:"template_id"`
	Source      string `json:"source,omitempty"`
	Target      string `json:"target,omitempty"`
	Fingerprint string `json:"fingerprint"`
}

type IntegrationQueue struct {
	Mode           string      `json:"mode"`
	Reason         string      `json:"reason"`
	UsedFallback   bool        `json:"used_fallback"`
	ItemCount      int         `json:"item_count"`
	NonConflicting bool        `json:"non_conflicting"`
	Digest         string      `json:"digest,omitempty"`
	Items          []QueueItem `json:"items,omitempty"`
}

type MergeReport struct {
	UsedFallback bool `json:"used_fallback"`

	Reason string `json:"reason"`

	BranchMode    string       `json:"branch_mode"`
	BranchEnabled bool         `json:"branch_enabled"`
	BranchReason  string       `json:"branch_reason"`
	Branches      []BranchPlan `json:"branches,omitempty"`
	QueueDigest   string       `json:"queue_digest,omitempty"`
	QueueItems    int          `json:"queue_items,omitempty"`
}

type Plan struct {
	Config   Config   `json:"config"`
	Decision Decision `json:"decision"`
	Tasks    []Task   `json:"tasks,omitempty"`
	Results  []Result `json:"results,omitempty"`
}

func ResolveConfig(c contract.Contract) Config {
	cfg := Config{
		Mode:               modeAuto,
		MaxWorkers:         4,
		TimeoutMs:          12000,
		RetryLimit:         1,
		MaxDigestChars:     240,
		MaxOutputChars:     220,
		AutoMinProjections: 5,
		AutoMinScopes:      2,
		BranchMode:         modeOff,
		BranchAutoMinTasks: 3,
		BranchPrefix:       "codex/sw/",
	}
	if c.Autonomy != nil && c.Autonomy.Subworkers != nil {
		sw := c.Autonomy.Subworkers
		cfg.Mode = normalizeMode(sw.Mode, cfg.Mode)
		cfg.MaxWorkers = pickPositive(sw.MaxWorkers, cfg.MaxWorkers)
		cfg.TimeoutMs = pickPositive(sw.TimeoutMs, cfg.TimeoutMs)
		cfg.RetryLimit = pickNonNegative(sw.RetryLimit, cfg.RetryLimit)
		cfg.MaxDigestChars = pickPositive(sw.MaxDigestChars, cfg.MaxDigestChars)
		cfg.MaxOutputChars = pickPositive(sw.MaxOutputChars, cfg.MaxOutputChars)
		cfg.AutoMinProjections = pickPositive(sw.AutoMinProjections, cfg.AutoMinProjections)
		cfg.AutoMinScopes = pickPositive(sw.AutoMinScopes, cfg.AutoMinScopes)
		cfg.BranchMode = normalizeMode(sw.BranchMode, cfg.BranchMode)
		cfg.BranchAutoMinTasks = pickPositive(sw.BranchAutoMinTasks, cfg.BranchAutoMinTasks)
		if strings.TrimSpace(sw.BranchPrefix) != "" {
			cfg.BranchPrefix = strings.TrimSpace(sw.BranchPrefix)
		}
	}

	cfg.Mode = normalizeMode(os.Getenv("ATRAKTA_SUBWORKER"), cfg.Mode)
	cfg.MaxWorkers = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_MAX_WORKERS")), cfg.MaxWorkers)
	cfg.TimeoutMs = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_TIMEOUT_MS")), cfg.TimeoutMs)
	cfg.RetryLimit = pickNonNegative(parseInt(os.Getenv("ATRAKTA_SUBWORKER_RETRY_LIMIT")), cfg.RetryLimit)
	cfg.MaxDigestChars = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_MAX_DIGEST_CHARS")), cfg.MaxDigestChars)
	cfg.MaxOutputChars = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_MAX_OUTPUT_CHARS")), cfg.MaxOutputChars)
	cfg.AutoMinProjections = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_AUTO_MIN_PROJECTIONS")), cfg.AutoMinProjections)
	cfg.AutoMinScopes = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_AUTO_MIN_SCOPES")), cfg.AutoMinScopes)
	cfg.BranchMode = normalizeMode(os.Getenv("ATRAKTA_SUBWORKER_BRANCH_PARALLEL"), cfg.BranchMode)
	cfg.BranchAutoMinTasks = pickPositive(parseInt(os.Getenv("ATRAKTA_SUBWORKER_BRANCH_AUTO_MIN_TASKS")), cfg.BranchAutoMinTasks)

	return cfg
}

func BuildPhaseA(det model.DetectResult, projections []projection.Desired, cfg Config) Plan {
	grouped := groupByScope(projections)
	scopeCount := len(grouped)
	projectionCount := len(projections)
	decision := decide(det, cfg, projectionCount, scopeCount)
	out := Plan{Config: cfg, Decision: decision}
	if !decision.Enabled || scopeCount == 0 {
		return out
	}

	scopes := sortedScopes(grouped)
	serialGroups := assignSerialGroups(scopes)
	workerCount := minInt(cfg.MaxWorkers, maxInt(1, len(scopes)))
	if workerCount <= 0 {
		workerCount = 1
	}

	out.Tasks = make([]Task, 0, len(scopes))
	out.Results = make([]Result, 0, len(scopes))
	for i, scope := range scopes {
		workerID := fmt.Sprintf("w%02d", (i%workerCount)+1)
		entries := grouped[scope]
		digest := buildDigest(scope, entries, cfg.MaxDigestChars)
		task := Task{
			WorkerID:    workerID,
			Scope:       scope,
			Tools:       []string{"read", "analyze", "propose"},
			Constraints: []string{"no_write", "no_side_effect", "json_output_only", "summary_cap"},
			Task:        "analyze scoped projection intents and return proposal json",
			Digest:      digest,
			SerialGroup: serialGroups[scope],
			Order:       i + 1,
			Proposals:   toProposals(entries),
		}
		out.Tasks = append(out.Tasks, task)
		out.Results = append(out.Results, Result{
			WorkerID:        workerID,
			Scope:           scope,
			Status:          "ok",
			ProposedOpCount: len(entries),
			Summary:         trimString(fmt.Sprintf("scope %s proposed %d operation(s)", scope, len(entries)), cfg.MaxOutputChars),
			TokenEstimate:   estimateTokens(digest, len(entries)),
			TimeoutMs:       cfg.TimeoutMs,
			RetryLimit:      cfg.RetryLimit,
		})
	}
	return out
}

func ValidatePlan(p Plan) error {
	if !p.Decision.Enabled {
		return nil
	}
	for i, t := range p.Tasks {
		if len(t.Digest) > p.Config.MaxDigestChars {
			return fmt.Errorf("task[%d] digest exceeds max_digest_chars", i)
		}
		for j, pr := range t.Proposals {
			if strings.TrimSpace(pr.Path) == "" || strings.TrimSpace(pr.TemplateID) == "" || strings.TrimSpace(pr.Fingerprint) == "" {
				return fmt.Errorf("task[%d] proposal[%d] missing required fields", i, j)
			}
		}
	}
	for i, r := range p.Results {
		if len(r.Summary) > p.Config.MaxOutputChars {
			return fmt.Errorf("result[%d] summary exceeds max_output_chars", i)
		}
	}
	return nil
}

func AggregateMetrics(p Plan) Metrics {
	m := Metrics{Enabled: p.Decision.Enabled, TaskCount: len(p.Tasks)}
	if !p.Decision.Enabled {
		return m
	}
	for _, t := range p.Tasks {
		if n := len(t.Digest); n > m.MaxDigestCharsUsed {
			m.MaxDigestCharsUsed = n
		}
	}
	for _, r := range p.Results {
		m.TotalTokenEstimate += r.TokenEstimate
		if n := len(r.Summary); n > m.MaxSummaryCharsUsed {
			m.MaxSummaryCharsUsed = n
		}
	}
	return m
}

func ApplyBudgetGuard(p Plan, budget contract.TokenBudget) (Plan, BudgetReport) {
	before := AggregateMetrics(p)
	report := BudgetReport{
		Reason:               "no_change",
		SoftLimit:            budget.Soft,
		HardLimit:            budget.Hard,
		BeforeTokenEstimate:  before.TotalTokenEstimate,
		AfterTokenEstimate:   before.TotalTokenEstimate,
		BeforeMaxDigestChars: before.MaxDigestCharsUsed,
		AfterMaxDigestChars:  before.MaxDigestCharsUsed,
		BeforeMaxOutputChars: before.MaxSummaryCharsUsed,
		AfterMaxOutputChars:  before.MaxSummaryCharsUsed,
	}
	if !p.Decision.Enabled {
		report.Reason = "subworker_disabled"
		return p, report
	}

	soft := budget.Soft
	hard := budget.Hard
	if soft <= 0 && hard <= 0 {
		report.Reason = "budget_limits_unset"
		return p, report
	}
	if soft <= 0 {
		soft = hard
	}
	if hard > 0 && soft > hard {
		soft = hard
	}
	report.SoftLimit = soft
	report.HardLimit = hard
	if soft > 0 && before.TotalTokenEstimate <= soft {
		report.Reason = "within_soft_limit"
		return p, report
	}

	out := p
	out.Tasks = append([]Task(nil), p.Tasks...)
	out.Results = append([]Result(nil), p.Results...)

	targetDigest := out.Config.MaxDigestChars
	targetOutput := out.Config.MaxOutputChars
	if targetDigest <= 0 {
		targetDigest = 240
	}
	if targetOutput <= 0 {
		targetOutput = 220
	}

	// Compression stays bounded and deterministic. Hard-limit pressure uses tighter caps.
	digestCap := 140
	outputCap := 140
	if hard > 0 && before.TotalTokenEstimate > hard {
		digestCap = 96
		outputCap = 96
	}
	targetDigest = minInt(targetDigest, digestCap)
	targetOutput = minInt(targetOutput, outputCap)

	for i := range out.Tasks {
		out.Tasks[i].Digest = trimString(out.Tasks[i].Digest, targetDigest)
	}
	for i := range out.Results {
		out.Results[i].Summary = trimString(out.Results[i].Summary, targetOutput)
		if i < len(out.Tasks) {
			out.Results[i].TokenEstimate = estimateTokens(out.Tasks[i].Digest, len(out.Tasks[i].Proposals))
		}
	}
	out.Config.MaxDigestChars = targetDigest
	out.Config.MaxOutputChars = targetOutput

	after := AggregateMetrics(out)
	report.Applied = true
	report.AfterTokenEstimate = after.TotalTokenEstimate
	report.AfterMaxDigestChars = after.MaxDigestCharsUsed
	report.AfterMaxOutputChars = after.MaxSummaryCharsUsed

	if hard > 0 && after.TotalTokenEstimate > hard {
		out.Decision.Enabled = false
		out.Decision.Reason = "budget_hard_limit_exceeded_fallback_single_writer"
		out.Tasks = nil
		out.Results = nil
		report.Disabled = true
		report.Reason = "disabled_due_to_hard_limit"
		final := AggregateMetrics(out)
		report.AfterTokenEstimate = final.TotalTokenEstimate
		report.AfterMaxDigestChars = final.MaxDigestCharsUsed
		report.AfterMaxOutputChars = final.MaxSummaryCharsUsed
		return out, report
	}

	if soft > 0 && after.TotalTokenEstimate <= soft {
		report.Reason = "compressed_to_soft_limit"
	} else {
		report.Reason = "compressed_best_effort"
	}
	return out, report
}

func MergePhaseA(p Plan, fallback []projection.Desired) ([]projection.Desired, MergeReport, error) {
	report := MergeReport{
		UsedFallback: true,
		Reason:       "subworker_disabled",
		BranchMode:   normalizeMode(p.Config.BranchMode, modeOff),
		BranchReason: "branch_mode_off",
	}
	if !p.Decision.Enabled || len(p.Tasks) == 0 {
		return cloneDesired(fallback), report, nil
	}
	if err := ValidatePlan(p); err != nil {
		report.Reason = "validation_failed_fallback_single_writer"
		return cloneDesired(fallback), report, nil
	}

	orderedTasks := append([]Task(nil), p.Tasks...)
	sort.SliceStable(orderedTasks, func(i, j int) bool {
		if orderedTasks[i].Order != orderedTasks[j].Order {
			return orderedTasks[i].Order < orderedTasks[j].Order
		}
		if orderedTasks[i].Scope != orderedTasks[j].Scope {
			return orderedTasks[i].Scope < orderedTasks[j].Scope
		}
		return orderedTasks[i].WorkerID < orderedTasks[j].WorkerID
	})

	type sourceInfo struct {
		WorkerID string
		Order    int
		Proposal ProposedProjection
	}
	seen := map[string]sourceInfo{}
	out := make([]projection.Desired, 0, len(fallback))
	for _, t := range orderedTasks {
		for _, pr := range t.Proposals {
			key := pr.Path + "|" + pr.TemplateID
			if prev, ok := seen[key]; ok {
				if !sameProposal(prev.Proposal, pr) {
					report.Reason = "proposal_conflict_fallback_single_writer"
					return cloneDesired(fallback), report, nil
				}
				continue
			}
			seen[key] = sourceInfo{WorkerID: t.WorkerID, Order: t.Order, Proposal: pr}
			out = append(out, projection.Desired{
				Interface:   pr.Interface,
				TemplateID:  pr.TemplateID,
				Path:        pr.Path,
				Source:      pr.Source,
				Target:      pr.Target,
				Fingerprint: pr.Fingerprint,
			})
		}
	}
	if len(out) == 0 {
		report.Reason = "empty_merge_fallback_single_writer"
		return cloneDesired(fallback), report, nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].TemplateID < out[j].TemplateID
		}
		return out[i].Path < out[j].Path
	})

	report.UsedFallback = false
	report.Reason = "merged_subworker_proposals"
	report.BranchEnabled, report.BranchReason, report.Branches = buildBranchPlan(p, orderedTasks)
	q := BuildSingleWriterQueue(p, out, false)
	report.QueueDigest = q.Digest
	report.QueueItems = q.ItemCount
	return out, report, nil
}

func BuildSingleWriterQueue(p Plan, selected []projection.Desired, usedFallback bool) IntegrationQueue {
	q := IntegrationQueue{
		Mode:           "single_writer",
		UsedFallback:   usedFallback,
		NonConflicting: true,
	}
	if usedFallback {
		q.Reason = "fallback_projection_set"
	} else {
		q.Reason = "merged_projection_set"
	}
	if len(selected) == 0 {
		q.Reason = "empty_projection_set"
		return q
	}
	meta := sourceByProjectionKey(p.Tasks)
	items := make([]QueueItem, 0, len(selected))
	for _, d := range selected {
		src, ok := meta[projectionKey(d.Path, d.TemplateID)]
		if !ok {
			src = queueSource{WorkerID: "orchestrator"}
		}
		items = append(items, QueueItem{
			Path:        d.Path,
			WorkerID:    src.WorkerID,
			Order:       src.Order,
			SerialGroup: src.SerialGroup,
			Interface:   d.Interface,
			TemplateID:  d.TemplateID,
			Source:      d.Source,
			Target:      d.Target,
			Fingerprint: d.Fingerprint,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Path != items[j].Path {
			return items[i].Path < items[j].Path
		}
		if items[i].WorkerID != items[j].WorkerID {
			return items[i].WorkerID < items[j].WorkerID
		}
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		return items[i].TemplateID < items[j].TemplateID
	})
	seenPath := map[string]struct{}{}
	for i := range items {
		items[i].Index = i + 1
		if _, ok := seenPath[items[i].Path]; ok {
			q.NonConflicting = false
		}
		seenPath[items[i].Path] = struct{}{}
	}
	q.Items = items
	q.ItemCount = len(items)
	q.Digest = queueDigest(items)
	return q
}

func ValidateSingleWriterQueue(q IntegrationQueue) error {
	if q.Mode == "" {
		return fmt.Errorf("single writer queue mode is required")
	}
	seenKey := map[string]struct{}{}
	for i, it := range q.Items {
		if strings.TrimSpace(it.Path) == "" || strings.TrimSpace(it.TemplateID) == "" || strings.TrimSpace(it.Fingerprint) == "" {
			return fmt.Errorf("queue item[%d] missing required fields", i)
		}
		if i > 0 && compareQueueItems(q.Items[i-1], it) > 0 {
			return fmt.Errorf("queue order is non-deterministic at item[%d]", i)
		}
		key := projectionKey(it.Path, it.TemplateID)
		if _, exists := seenKey[key]; exists {
			return fmt.Errorf("queue contains duplicate projection key %q", key)
		}
		seenKey[key] = struct{}{}
	}
	if q.ItemCount != len(q.Items) {
		return fmt.Errorf("queue item_count mismatch")
	}
	if q.Digest != "" {
		if expected := queueDigest(q.Items); expected != q.Digest {
			return fmt.Errorf("queue digest mismatch")
		}
	}
	return nil
}

func QueueProjections(q IntegrationQueue) []projection.Desired {
	out := make([]projection.Desired, 0, len(q.Items))
	for _, it := range q.Items {
		out = append(out, projection.Desired{
			Interface:   it.Interface,
			TemplateID:  it.TemplateID,
			Path:        it.Path,
			Source:      it.Source,
			Target:      it.Target,
			Fingerprint: it.Fingerprint,
		})
	}
	return out
}

func decide(det model.DetectResult, cfg Config, projectionCount, scopeCount int) Decision {
	signals := map[string]any{
		"projection_count": projectionCount,
		"scope_count":      scopeCount,
		"target_count":     len(det.TargetSet),
		"detect_reason":    string(det.Reason),
	}
	switch cfg.Mode {
	case modeOff:
		return Decision{Mode: cfg.Mode, Enabled: false, Reason: "mode_off", Signals: signals}
	case modeOn:
		if projectionCount == 0 || scopeCount == 0 {
			return Decision{Mode: cfg.Mode, Enabled: false, Reason: "no_work_items", Signals: signals}
		}
		return Decision{Mode: cfg.Mode, Enabled: true, Reason: "mode_on", Signals: signals}
	default:
		if projectionCount < cfg.AutoMinProjections {
			return Decision{Mode: modeAuto, Enabled: false, Reason: "below_auto_min_projections", Signals: signals}
		}
		if scopeCount < cfg.AutoMinScopes {
			return Decision{Mode: modeAuto, Enabled: false, Reason: "below_auto_min_scopes", Signals: signals}
		}
		isAmbiguous := det.Reason == model.ReasonConflict || det.Reason == model.ReasonMixed || det.Reason == model.ReasonUnknown
		multiTarget := len(det.TargetSet) >= 2
		bigBatch := projectionCount >= cfg.AutoMinProjections*2
		if isAmbiguous || multiTarget || bigBatch {
			return Decision{Mode: modeAuto, Enabled: true, Reason: "auto_parallel_benefit_detected", Signals: signals}
		}
		return Decision{Mode: modeAuto, Enabled: false, Reason: "auto_benefit_not_detected", Signals: signals}
	}
}

func groupByScope(projections []projection.Desired) map[string][]projection.Desired {
	out := map[string][]projection.Desired{}
	for _, d := range projections {
		scope := scopeOfPath(d.Path)
		out[scope] = append(out[scope], d)
	}
	for scope := range out {
		rows := out[scope]
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Path == rows[j].Path {
				return rows[i].TemplateID < rows[j].TemplateID
			}
			return rows[i].Path < rows[j].Path
		})
		out[scope] = rows
	}
	return out
}

func scopeOfPath(p string) string {
	n := util.NormalizeRelPath(p)
	if n == "" {
		return "./"
	}
	dir := path.Dir(n)
	if dir == "." || dir == "" {
		return n
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

func sortedScopes(grouped map[string][]projection.Desired) []string {
	out := make([]string, 0, len(grouped))
	for scope := range grouped {
		out = append(out, scope)
	}
	sort.Strings(out)
	return out
}

func assignSerialGroups(scopes []string) map[string]int {
	out := map[string]int{}
	anchors := []string{}
	for _, scope := range scopes {
		group := 0
		for i, anchor := range anchors {
			if scopeOverlaps(scope, anchor) {
				group = i + 1
				break
			}
		}
		if group == 0 {
			anchors = append(anchors, scope)
			group = len(anchors)
		}
		out[scope] = group
	}
	return out
}

func scopeOverlaps(a, b string) bool {
	if a == b {
		return true
	}
	a = normalizeScope(a)
	b = normalizeScope(b)
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

func normalizeScope(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\\", "/"))
	if s == "" {
		return "./"
	}
	if strings.HasSuffix(s, "/") {
		return s
	}
	return s + "/"
}

func buildDigest(scope string, rows []projection.Desired, maxChars int) string {
	if len(rows) == 0 {
		return trimString("scope="+scope+" ops=0", maxChars)
	}
	paths := make([]string, 0, minInt(3, len(rows)))
	for i := 0; i < len(rows) && i < 3; i++ {
		paths = append(paths, rows[i].Path)
	}
	joined := strings.Join(paths, ",")
	text := fmt.Sprintf("scope=%s ops=%d sample_paths=%s", scope, len(rows), joined)
	return trimString(text, maxChars)
}

func estimateTokens(digest string, opCount int) int {
	base := 40 + len(digest)/4
	if opCount > 0 {
		base += opCount * 12
	}
	return base
}

func trimString(s string, maxChars int) string {
	if maxChars <= 0 {
		return s
	}
	if len(s) <= maxChars {
		return s
	}
	if maxChars <= 3 {
		return s[:maxChars]
	}
	return s[:maxChars-3] + "..."
}

func parseInt(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0
	}
	return n
}

func normalizeMode(raw, fallback string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case modeOff:
		return modeOff
	case modeOn:
		return modeOn
	case modeAuto:
		return modeAuto
	default:
		return fallback
	}
}

func pickPositive(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func pickNonNegative(v, fallback int) int {
	if v >= 0 {
		return v
	}
	return fallback
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func toProposals(rows []projection.Desired) []ProposedProjection {
	out := make([]ProposedProjection, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProposedProjection{
			Interface:   r.Interface,
			TemplateID:  r.TemplateID,
			Path:        r.Path,
			Source:      r.Source,
			Target:      r.Target,
			Fingerprint: r.Fingerprint,
		})
	}
	return out
}

func sameProposal(a, b ProposedProjection) bool {
	return a.Interface == b.Interface &&
		a.TemplateID == b.TemplateID &&
		a.Path == b.Path &&
		a.Source == b.Source &&
		a.Target == b.Target &&
		a.Fingerprint == b.Fingerprint
}

func cloneDesired(in []projection.Desired) []projection.Desired {
	out := make([]projection.Desired, len(in))
	copy(out, in)
	return out
}

func buildBranchPlan(cfg Plan, tasks []Task) (bool, string, []BranchPlan) {
	mode := normalizeMode(cfg.Config.BranchMode, modeOff)
	if mode == modeOff {
		return false, "branch_mode_off", nil
	}
	if len(tasks) < 2 {
		return false, "insufficient_tasks", nil
	}
	serialGroupCounts := map[int]int{}
	for _, t := range tasks {
		serialGroupCounts[t.SerialGroup]++
	}
	for _, cnt := range serialGroupCounts {
		if cnt > 1 {
			return false, "serial_overlap_detected_fallback_single_writer", nil
		}
	}
	if mode == modeAuto && len(tasks) < cfg.Config.BranchAutoMinTasks {
		return false, "below_branch_auto_min_tasks", nil
	}

	branches := make([]BranchPlan, 0, len(tasks))
	prefix := strings.TrimSpace(cfg.Config.BranchPrefix)
	if prefix == "" {
		prefix = "codex/sw/"
	}
	for _, t := range tasks {
		branches = append(branches, BranchPlan{
			BranchName:  prefix + sanitizeBranchSegment(fmt.Sprintf("%02d-%s-%s", t.Order, t.WorkerID, t.Scope)),
			WorkerID:    t.WorkerID,
			Scope:       t.Scope,
			SerialGroup: t.SerialGroup,
			Order:       t.Order,
			Proposals:   len(t.Proposals),
		})
	}
	return true, "branch_parallel_enabled", branches
}

func sanitizeBranchSegment(s string) string {
	x := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(s, "\\", "/")))
	x = strings.ReplaceAll(x, " ", "-")
	x = strings.ReplaceAll(x, "//", "/")
	repl := []string{"~", "^", ":", "?", "*", "[", "\\", "..", "@{", "//"}
	for _, r := range repl {
		x = strings.ReplaceAll(x, r, "-")
	}
	x = strings.Trim(x, "/.-")
	if x == "" {
		return "lane"
	}
	return x
}

type queueSource struct {
	WorkerID    string
	Order       int
	SerialGroup int
}

func sourceByProjectionKey(tasks []Task) map[string]queueSource {
	ordered := append([]Task(nil), tasks...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Order != ordered[j].Order {
			return ordered[i].Order < ordered[j].Order
		}
		if ordered[i].Scope != ordered[j].Scope {
			return ordered[i].Scope < ordered[j].Scope
		}
		return ordered[i].WorkerID < ordered[j].WorkerID
	})
	out := map[string]queueSource{}
	for _, t := range ordered {
		for _, pr := range t.Proposals {
			key := projectionKey(pr.Path, pr.TemplateID)
			if _, exists := out[key]; exists {
				continue
			}
			out[key] = queueSource{
				WorkerID:    t.WorkerID,
				Order:       t.Order,
				SerialGroup: t.SerialGroup,
			}
		}
	}
	return out
}

func projectionKey(path, templateID string) string {
	return path + "|" + templateID
}

func compareQueueItems(a, b QueueItem) int {
	if a.Path != b.Path {
		if a.Path < b.Path {
			return -1
		}
		return 1
	}
	if a.WorkerID != b.WorkerID {
		if a.WorkerID < b.WorkerID {
			return -1
		}
		return 1
	}
	if a.Order != b.Order {
		if a.Order < b.Order {
			return -1
		}
		return 1
	}
	if a.TemplateID != b.TemplateID {
		if a.TemplateID < b.TemplateID {
			return -1
		}
		return 1
	}
	return 0
}

func queueDigest(items []QueueItem) string {
	rows := make([]map[string]any, 0, len(items))
	for _, it := range items {
		rows = append(rows, map[string]any{
			"index":        it.Index,
			"path":         it.Path,
			"worker_id":    it.WorkerID,
			"order":        it.Order,
			"serial_group": it.SerialGroup,
			"interface":    it.Interface,
			"template_id":  it.TemplateID,
			"fingerprint":  it.Fingerprint,
		})
	}
	b, err := util.MarshalCanonical(map[string]any{"items": rows})
	if err != nil {
		return ""
	}
	return util.SHA256Tagged(b)
}
