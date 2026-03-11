package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/model"
	"atrakta/internal/projection"
	"atrakta/internal/state"
	"atrakta/internal/taskgraph"
	"atrakta/internal/util"
)

type Input struct {
	RepoRoot    string
	Contract    contract.Contract
	Detect      model.DetectResult
	State       state.State
	FeatureID   string
	Projections []projection.Desired
}

func Build(in Input) (model.PlanResult, error) {
	desired := normalizeDesired(in.Projections)
	desiredPaths := make([]string, 0, len(desired))
	for p := range desired {
		desiredPaths = append(desiredPaths, p)
	}
	sort.Strings(desiredPaths)

	ops := []model.Operation{}
	sourceCache := map[string]string{}
	for _, rel := range desiredPaths {
		d := desired[rel]
		rec, managed := in.State.ManagedPaths[rel]
		_, statErr := os.Lstat(join(in.RepoRoot, rel))
		exists := statErr == nil
		managedStable := managed && rec.Fingerprint == d.Fingerprint && rec.TemplateID == d.TemplateID
		// Trust managed-state fast path only when the projection artifact is still present.
		if managedStable && exists {
			continue
		}
		equivalentExisting := exists && isEquivalentExistingProjection(in.RepoRoot, rel, d, sourceCache)
		if equivalentExisting {
			ops = append(ops, model.Operation{
				Op:               "adopt",
				Path:             rel,
				RequiresApproval: false,
				Source:           d.Source,
				Target:           d.Target,
				Fingerprint:      d.Fingerprint,
				Interface:        d.Interface,
				TemplateID:       d.TemplateID,
				Reason:           "adopt_existing_equivalent",
			})
			continue
		}
		if exists && !managed {
			ops = append(ops, model.Operation{
				Op:               "write",
				Path:             rel,
				RequiresApproval: true,
				Source:           d.Source,
				Target:           d.Target,
				Fingerprint:      d.Fingerprint,
				Interface:        d.Interface,
				TemplateID:       d.TemplateID,
				Reason:           "overwrite_non_managed",
			})
			continue
		}
		ops = append(ops, model.Operation{
			Op:               desiredOp(d),
			Path:             rel,
			RequiresApproval: false,
			Source:           d.Source,
			Target:           d.Target,
			Fingerprint:      d.Fingerprint,
			Interface:        d.Interface,
			TemplateID:       d.TemplateID,
			Reason:           "ensure_required_template",
		})
	}

	if in.Detect.PruneAllowed {
		requireApproval := contract.HasApprovalRequirement(in.Contract, "destructive_prune")
		for path, rec := range in.State.ManagedPaths {
			if _, ok := desired[path]; ok {
				continue
			}
			ops = append(ops, model.Operation{
				Op:               "delete",
				Path:             path,
				RequiresApproval: requireApproval,
				Interface:        rec.Interface,
				TemplateID:       rec.TemplateID,
				Reason:           "prune_unused",
			})
		}
	}

	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		pi := opPriority(ops[i].Op)
		pj := opPriority(ops[j].Op)
		if pi != pj {
			return pi < pj
		}
		return ops[i].Interface < ops[j].Interface
	})

	requiresApproval := false
	for _, op := range ops {
		if op.RequiresApproval {
			requiresApproval = true
			break
		}
	}

	planID := util.NewEventID()
	ops, graph, err := taskgraph.BuildFromOps(planID, ops)
	if err != nil {
		return model.PlanResult{}, err
	}
	summary := fmt.Sprintf("plan: %d ops", len(ops))
	details := fmt.Sprintf("target_set=%v prune_allowed=%v", in.Detect.TargetSet, in.Detect.PruneAllowed)
	return model.PlanResult{
		ID:                 planID,
		TaskGraphID:        graph.GraphID,
		TaskCount:          graph.TaskCount,
		TaskEdgeCount:      graph.EdgeCount,
		FeatureID:          in.FeatureID,
		Ops:                ops,
		RequiredPermission: requiredPermissionForOps(ops),
		Summary:            summary,
		Details:            details,
		RequiresApproval:   requiresApproval,
		ApprovalContext: map[string]any{
			"ops":    ops,
			"reason": "operations require approval",
		},
	}, nil
}

func desiredOp(d projection.Desired) string {
	if strings.TrimSpace(d.Target) == "" {
		return "copy"
	}
	return "link"
}

func normalizeDesired(in []projection.Desired) map[string]projection.Desired {
	out := make(map[string]projection.Desired, len(in))
	for _, d := range in {
		rel := util.NormalizeRelPath(d.Path)
		if rel == "" {
			continue
		}
		d.Path = rel
		// Last write wins for deterministic override behavior.
		out[rel] = d
	}
	return out
}

func opPriority(op string) int {
	switch op {
	case "adopt", "link", "copy", "write":
		return 0
	case "delete", "unlink":
		return 1
	default:
		return 2
	}
}

func requiredPermissionForOps(ops []model.Operation) model.Permission {
	if len(ops) == 0 {
		return model.PermissionReadOnly
	}
	for _, op := range ops {
		switch op.Op {
		case "adopt", "link", "copy", "write", "delete", "unlink":
			return model.PermissionWorkspaceWrite
		}
	}
	return model.PermissionReadOnly
}

func join(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

func isEquivalentExistingProjection(repoRoot, rel string, d projection.Desired, sourceCache map[string]string) bool {
	if isEquivalentSymlink(repoRoot, rel, d.Target) {
		return true
	}
	abs := join(repoRoot, rel)
	fi, err := os.Lstat(abs)
	if err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		return false
	}
	sourceText, ok := loadSourceText(repoRoot, d, sourceCache)
	if !ok {
		return false
	}
	expected := projection.ManagedContentForPath(rel, d.TemplateID, d.Fingerprint, sourceText)
	b, err := os.ReadFile(abs)
	if err != nil {
		return false
	}
	return util.NormalizeContentLF(string(b)) == expected
}

func loadSourceText(repoRoot string, d projection.Desired, sourceCache map[string]string) (string, bool) {
	if synthetic, ok := projection.SyntheticTemplateContent(d.TemplateID); ok {
		return synthetic, true
	}
	source := util.NormalizeRelPath(d.Source)
	if source == "" {
		return "", false
	}
	if cached, ok := sourceCache[source]; ok {
		return cached, true
	}
	b, err := os.ReadFile(join(repoRoot, source))
	if err != nil {
		return "", false
	}
	text := string(b)
	sourceCache[source] = text
	return text, true
}

func isEquivalentSymlink(repoRoot, rel, target string) bool {
	if strings.TrimSpace(target) == "" {
		return false
	}
	abs := join(repoRoot, rel)
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
	targetAbs := join(repoRoot, target)
	return filepath.Clean(gotAbs) == filepath.Clean(targetAbs)
}
