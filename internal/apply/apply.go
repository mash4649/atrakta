package apply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"atrakta/internal/contract"
	"atrakta/internal/editsafety"
	"atrakta/internal/model"
	"atrakta/internal/platform"
	"atrakta/internal/projection"
	"atrakta/internal/proof"
	"atrakta/internal/state"
	"atrakta/internal/taskgraph"
	"atrakta/internal/util"
)

const (
	parallelModeOff  = "off"
	parallelModeAuto = "auto"
	parallelModeOn   = "on"
)

type Input struct {
	RepoRoot     string
	Contract     contract.Contract
	ContractHash string
	State        state.State
	Plan         model.PlanResult
	Approved     bool
	DetectReason model.DetectReason
	SourceAGENTS string

	ParallelMode       string
	ParallelMaxWorkers int
}

type sourceResolver struct {
	repoRoot string
	fallback string

	mu    sync.RWMutex
	cache map[string]string
}

type compiledOp struct {
	op             model.Operation
	destructive    bool
	approvalDenied bool
	pathErr        error
	managedErr     error
	linkSatisfied  bool
	adoptSatisfied bool
	adoptKind      string
}

func Run(in Input) model.ApplyResult {
	orderedOps, err := taskgraph.TopoOrder(in.Plan.Ops)
	if err != nil {
		return model.ApplyResult{
			PlanID:    in.Plan.ID,
			FeatureID: in.Plan.FeatureID,
			Result:    "fail",
			Ops: []model.OpResult{{
				Status: "failed",
				Error:  "task graph invalid: " + err.Error(),
			}},
		}
	}
	resolver := newSourceResolver(in.RepoRoot, in.SourceAGENTS)
	compiled := compileOps(in, orderedOps, resolver)
	executable := countExecutableOps(compiled)
	mode := resolveExecutionMode(in, compiled)
	if shouldParallelApply(mode, in, compiled, executable) {
		return runParallel(in, compiled, resolver, resolveParallelWorkers(mode, in, executable))
	}
	return runSequential(in, compiled, resolver)
}

func runSequential(in Input, ops []compiledOp, resolver *sourceResolver) model.ApplyResult {
	result := model.ApplyResult{PlanID: in.Plan.ID, FeatureID: in.Plan.FeatureID, Result: "success", Ops: make([]model.OpResult, 0, len(ops))}
	for _, op := range ops {
		if r, ok := compileTimeNoopResult(op); ok {
			result.Ops = append(result.Ops, r)
			continue
		}
		r, hardFail := runOne(in.RepoRoot, in.Contract.EditSafety, op, resolver)
		if r.Status == "failed" && result.Result == "success" {
			result.Result = "partial"
		}
		if hardFail {
			result.Result = "fail"
			result.Ops = append(result.Ops, r)
			break
		}
		result.Ops = append(result.Ops, r)
	}
	if len(result.Ops) == 0 {
		result.Result = "success"
	}
	return result
}

func runParallel(in Input, ops []compiledOp, resolver *sourceResolver, workers int) model.ApplyResult {
	results := make([]model.OpResult, len(ops))
	jobs := make(chan int)
	pending := make([]int, 0, len(ops))
	if workers <= 0 {
		workers = 1
	}
	for i, op := range ops {
		if r, ok := compileTimeNoopResult(op); ok {
			results[i] = r
			continue
		}
		pending = append(pending, i)
	}
	if len(pending) == 0 {
		return model.ApplyResult{
			PlanID:    in.Plan.ID,
			FeatureID: in.Plan.FeatureID,
			Result:    "success",
			Ops:       results,
		}
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				r, _ := runOne(in.RepoRoot, in.Contract.EditSafety, ops[idx], resolver)
				results[idx] = r
			}
		}()
	}
	for _, idx := range pending {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()

	result := model.ApplyResult{
		PlanID:    in.Plan.ID,
		FeatureID: in.Plan.FeatureID,
		Result:    "success",
		Ops:       make([]model.OpResult, 0, len(results)),
	}
	hasFailure := false
	for _, r := range results {
		if r.Status == "failed" {
			hasFailure = true
		}
		result.Ops = append(result.Ops, r)
	}
	if hasFailure {
		result.Result = "partial"
	}
	if len(result.Ops) == 0 {
		result.Result = "success"
	}
	return result
}

func runOne(repoRoot string, safety *contract.EditSafety, c compiledOp, resolver *sourceResolver) (model.OpResult, bool) {
	op := c.op
	r := baseOpResult(op)
	if c.approvalDenied {
		r.Status = "failed"
		r.Error = "approval denied"
		return r, true
	}
	if c.pathErr != nil {
		r.Status = "failed"
		r.Error = c.pathErr.Error()
		return r, true
	}
	if c.managedErr != nil {
		r.Status = "failed"
		r.Error = "managed-only proof failed: " + c.managedErr.Error()
		return r, true
	}
	if c.linkSatisfied {
		r.Kind = "link"
		r.Status = "skipped"
		return r, false
	}
	if op.Op == "adopt" {
		if !c.adoptSatisfied {
			r.Status = "failed"
			r.Error = "adopt precondition failed: path is no longer equivalent"
			return r, false
		}
		r.Kind = c.adoptKind
		if r.Kind == "" {
			r.Kind = "copy"
		}
		r.Status = "skipped"
		return r, false
	}
	kind, status, err := executeOp(repoRoot, op, safety, resolver)
	r.Kind = kind
	r.Status = status
	if err != nil {
		r.Error = err.Error()
		if c.destructive {
			return r, true
		}
	}
	return r, false
}

func shouldParallelApply(mode string, in Input, ops []compiledOp, executable int) bool {
	if mode == parallelModeOff {
		return false
	}
	if executable < 2 {
		return false
	}
	if in.Plan.RequiresApproval {
		return false
	}
	paths := map[string]struct{}{}
	for _, op := range ops {
		if isCompileTimeNoop(op) {
			continue
		}
		if op.destructive {
			return false
		}
		if op.op.RequiresApproval {
			return false
		}
		if _, ok := paths[op.op.Path]; ok {
			return false
		}
		paths[op.op.Path] = struct{}{}
	}
	if mode == parallelModeOn {
		return true
	}
	// auto mode: only when workload is large enough to amortize overhead
	return executable >= 4
}

func resolveParallelWorkers(mode string, in Input, opCount int) int {
	n := in.ParallelMaxWorkers
	if n <= 0 && mode == parallelModeAuto {
		n = runtime.GOMAXPROCS(0)
		if opCount >= 128 {
			n = n * 2
		}
	}
	if n <= 0 {
		n = 4
	}
	if n > 16 {
		n = 16
	}
	if n < 2 {
		n = 2
	}
	if opCount > 0 && n > opCount {
		n = opCount
	}
	return n
}

func resolveExecutionMode(in Input, ops []compiledOp) string {
	raw := strings.TrimSpace(strings.ToLower(in.ParallelMode))
	switch raw {
	case parallelModeOn:
		return parallelModeOn
	case parallelModeAuto:
		return parallelModeAuto
	case parallelModeOff:
		return parallelModeOff
	}
	// Smart default for large, safe workloads when mode is unspecified.
	if len(ops) >= 32 {
		return parallelModeAuto
	}
	return parallelModeOff
}

func countExecutableOps(ops []compiledOp) int {
	n := 0
	for _, op := range ops {
		if isCompileTimeNoop(op) {
			continue
		}
		n++
	}
	return n
}

func isCompileTimeNoop(c compiledOp) bool {
	if c.linkSatisfied {
		return true
	}
	if c.op.Op == "adopt" && c.adoptSatisfied {
		return true
	}
	return false
}

func compileTimeNoopResult(c compiledOp) (model.OpResult, bool) {
	if !isCompileTimeNoop(c) {
		return model.OpResult{}, false
	}
	r := baseOpResult(c.op)
	r.Status = "skipped"
	if c.linkSatisfied {
		r.Kind = "link"
		return r, true
	}
	r.Kind = c.adoptKind
	if r.Kind == "" {
		r.Kind = "copy"
	}
	return r, true
}

func baseOpResult(op model.Operation) model.OpResult {
	return model.OpResult{
		TaskID:      op.TaskID,
		Path:        op.Path,
		Op:          op.Op,
		Status:      "ok",
		Error:       "",
		Interface:   op.Interface,
		TemplateID:  op.TemplateID,
		Target:      op.Target,
		Fingerprint: op.Fingerprint,
	}
}

func normalizeParallelMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case parallelModeOn:
		return parallelModeOn
	case parallelModeAuto:
		return parallelModeAuto
	default:
		return parallelModeOff
	}
}

func isDestructive(op string) bool {
	switch op {
	case "delete", "unlink":
		return true
	case "write":
		return true
	default:
		return false
	}
}

func expectedFromOp(op model.Operation, resolver *sourceResolver) proof.Expected {
	return proof.Expected{
		Fingerprint: op.Fingerprint,
		Target:      op.Target,
		TemplateID:  op.TemplateID,
		SourceText:  resolver.load(op),
	}
}

func compileOps(in Input, ops []model.Operation, resolver *sourceResolver) []compiledOp {
	conventionsRO := in.Contract.Context == nil || in.Contract.Context.ConventionsReadOnly == nil || *in.Contract.Context.ConventionsReadOnly
	conventions := map[string]struct{}{}
	if conventionsRO {
		for _, p := range contract.ContextConventions(in.Contract) {
			conventions[p] = struct{}{}
		}
	}
	out := make([]compiledOp, 0, len(ops))
	for _, op := range ops {
		c := compiledOp{
			op:             op,
			destructive:    isDestructive(op.Op),
			approvalDenied: op.RequiresApproval && !in.Approved,
		}
		if _, blocked := conventions[util.NormalizeRelPath(op.Path)]; blocked {
			c.pathErr = fmt.Errorf("conventions path is read-only: %s", op.Path)
		}
		if err := platform.ValidateMutationPath(in.RepoRoot, op.Path, in.Contract.Boundary); err != nil && c.pathErr == nil {
			c.pathErr = err
		}
		if c.destructive {
			exp := expectedFromOp(op, resolver)
			if rec, ok := in.State.ManagedPaths[op.Path]; ok {
				if exp.Fingerprint == "" {
					exp.Fingerprint = rec.Fingerprint
				}
				if exp.TemplateID == "" {
					exp.TemplateID = rec.TemplateID
				}
				if exp.Target == "" {
					exp.Target = rec.Target
				}
			}
			c.managedErr = proof.IsManagedDestructiveAllowed(in.RepoRoot, op.Path, in.State.ManagedPaths, exp)
		}
		if op.Op == "link" && c.pathErr == nil && !c.approvalDenied {
			c.linkSatisfied = isLinkSatisfied(in.RepoRoot, op)
		}
		if op.Op == "adopt" && c.pathErr == nil && !c.approvalDenied {
			c.adoptKind, c.adoptSatisfied = probeAdoptSatisfaction(in.RepoRoot, op, resolver)
		}
		out = append(out, c)
	}
	return out
}

func executeOp(repoRoot string, op model.Operation, safety *contract.EditSafety, resolver *sourceResolver) (kind string, status string, err error) {
	abs := filepath.Join(repoRoot, filepath.FromSlash(op.Path))
	switch op.Op {
	case "link":
		srcAbs := filepath.Join(repoRoot, filepath.FromSlash(op.Target))
		if _, err := os.Lstat(abs); err == nil {
			if err := os.Remove(abs); err != nil {
				return "", "failed", fmt.Errorf("replace existing path: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", "failed", fmt.Errorf("lstat existing path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return "", "failed", fmt.Errorf("mkdir parent: %w", err)
		}
		relTarget, _ := filepath.Rel(filepath.Dir(abs), srcAbs)
		if err := os.Symlink(relTarget, abs); err == nil {
			return "link", "ok", nil
		}
		sourceText := resolver.load(op)
		if err := editsafety.ValidateCandidate(op.Path, sourceText, safety); err != nil {
			return "", "failed", err
		}
		content := projection.ManagedContentForPath(op.Path, op.TemplateID, op.Fingerprint, sourceText)
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			return "", "failed", fmt.Errorf("fallback copy write: %w", err)
		}
		return "copy", "ok", nil
	case "copy", "write":
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return "", "failed", fmt.Errorf("mkdir parent: %w", err)
		}
		sourceText := resolver.load(op)
		if err := editsafety.ValidateCandidate(op.Path, sourceText, safety); err != nil {
			return "", "failed", err
		}
		content := projection.ManagedContentForPath(op.Path, op.TemplateID, op.Fingerprint, sourceText)
		if b, err := os.ReadFile(abs); err == nil {
			if util.NormalizeContentLF(string(b)) == content {
				return "copy", "skipped", nil
			}
		}
		if fi, err := os.Lstat(abs); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return "", "failed", fmt.Errorf("refuse writing through symlink")
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			return "", "failed", fmt.Errorf("write file: %w", err)
		}
		return "copy", "ok", nil
	case "delete", "unlink":
		err := os.Remove(abs)
		if err == nil {
			return "", "ok", nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return "", "skipped", nil
		}
		if strings.Contains(err.Error(), "directory not empty") {
			return "", "failed", fmt.Errorf("refuse deleting non-empty directory")
		}
		return "", "failed", fmt.Errorf("delete: %w", err)
	default:
		return "", "failed", fmt.Errorf("unsupported op %q", op.Op)
	}
}

func newSourceResolver(repoRoot, fallback string) *sourceResolver {
	return &sourceResolver{
		repoRoot: repoRoot,
		fallback: fallback,
		cache:    map[string]string{},
	}
}

func (r *sourceResolver) load(op model.Operation) string {
	if synthetic, ok := projection.SyntheticTemplateContent(op.TemplateID); ok {
		return synthetic
	}
	source := op.Source
	if source == "" {
		if strings.HasSuffix(op.TemplateID, ":agents-md@1") {
			return r.fallback
		}
		return r.fallback
	}

	source = util.NormalizeRelPath(source)
	r.mu.RLock()
	if cached, ok := r.cache[source]; ok {
		r.mu.RUnlock()
		return cached
	}
	r.mu.RUnlock()

	b, err := os.ReadFile(filepath.Join(r.repoRoot, filepath.FromSlash(source)))
	text := ""
	if err != nil {
		text = r.fallback
	} else {
		text = string(b)
	}

	r.mu.Lock()
	r.cache[source] = text
	r.mu.Unlock()
	return text
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isLinkSatisfied(repoRoot string, op model.Operation) bool {
	abs := filepath.Join(repoRoot, filepath.FromSlash(op.Path))
	fi, err := os.Lstat(abs)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	got, err := os.Readlink(abs)
	if err != nil {
		return false
	}
	gotAbs := got
	if !filepath.IsAbs(gotAbs) {
		gotAbs = filepath.Join(filepath.Dir(abs), gotAbs)
	}
	srcAbs := filepath.Join(repoRoot, filepath.FromSlash(op.Target))
	return filepath.Clean(gotAbs) == filepath.Clean(srcAbs)
}

func probeAdoptSatisfaction(repoRoot string, op model.Operation, resolver *sourceResolver) (kind string, ok bool) {
	if isLinkSatisfied(repoRoot, op) {
		return "link", true
	}
	abs := filepath.Join(repoRoot, filepath.FromSlash(op.Path))
	fi, err := os.Lstat(abs)
	if err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return "", false
	}
	sourceText := resolver.load(op)
	content := projection.ManagedContentForPath(op.Path, op.TemplateID, op.Fingerprint, sourceText)
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", false
	}
	got := util.NormalizeContentLF(string(b))
	if got == content {
		return "copy", true
	}
	if util.NormalizeRelPath(op.Path) == util.NormalizeRelPath(op.Source) && got == util.NormalizeContentLF(sourceText) {
		return "copy", true
	}
	return "", false
}
