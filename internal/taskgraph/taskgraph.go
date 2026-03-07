package taskgraph

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"atrakta/internal/model"
	"atrakta/internal/util"
)

type Task struct {
	ID        string   `json:"id"`
	Index     int      `json:"index"`
	Path      string   `json:"path"`
	Op        string   `json:"op"`
	BlockedBy []string `json:"task_blocked_by,omitempty"`
}

type Graph struct {
	V         int    `json:"v"`
	GraphID   string `json:"graph_id"`
	PlanID    string `json:"plan_id"`
	TaskCount int    `json:"task_count"`
	EdgeCount int    `json:"edge_count"`
	Digest    string `json:"digest"`
	Tasks     []Task `json:"tasks"`
}

func BuildFromOps(planID string, ops []model.Operation) ([]model.Operation, Graph, error) {
	n := len(ops)
	if n == 0 {
		g := Graph{V: 1, GraphID: "", PlanID: planID, TaskCount: 0, EdgeCount: 0, Tasks: nil}
		g.Digest = digest(g)
		g.GraphID = g.Digest
		return ops, g, nil
	}

	work := make([]model.Operation, n)
	copy(work, ops)
	seenTaskID := map[string]struct{}{}
	for i := range work {
		if strings.TrimSpace(work[i].TaskID) == "" {
			work[i].TaskID = fmt.Sprintf("task-%04d", i+1)
		}
		if _, ok := seenTaskID[work[i].TaskID]; ok {
			return nil, Graph{}, fmt.Errorf("duplicate task_id: %s", work[i].TaskID)
		}
		seenTaskID[work[i].TaskID] = struct{}{}
		work[i].Path = util.NormalizeRelPath(work[i].Path)
	}

	deps := make([]map[int]struct{}, n)
	for i := 0; i < n; i++ {
		deps[i] = map[int]struct{}{}
		for j := 0; j < i; j++ {
			if conflicts(work[j], work[i]) {
				deps[i][j] = struct{}{}
			}
		}
	}
	reduceTransitiveDeps(deps)

	tasks := make([]Task, 0, n)
	edgeCount := 0
	for i := 0; i < n; i++ {
		blocked := make([]string, 0, len(deps[i]))
		idx := make([]int, 0, len(deps[i]))
		for j := range deps[i] {
			idx = append(idx, j)
		}
		sort.Ints(idx)
		for _, j := range idx {
			blocked = append(blocked, work[j].TaskID)
		}
		work[i].TaskBlockedBy = blocked
		edgeCount += len(blocked)
		tasks = append(tasks, Task{
			ID:        work[i].TaskID,
			Index:     i,
			Path:      work[i].Path,
			Op:        work[i].Op,
			BlockedBy: append([]string(nil), blocked...),
		})
	}

	g := Graph{
		V:         1,
		PlanID:    planID,
		TaskCount: len(tasks),
		EdgeCount: edgeCount,
		Tasks:     tasks,
	}
	g.Digest = digest(g)
	g.GraphID = g.Digest
	return work, g, nil
}

func GraphFromOps(planID string, ops []model.Operation) (Graph, error) {
	needsAnnotation := false
	for _, op := range ops {
		if strings.TrimSpace(op.TaskID) == "" {
			needsAnnotation = true
			break
		}
	}
	if needsAnnotation {
		_, g, err := BuildFromOps(planID, ops)
		return g, err
	}
	if _, err := TopoOrder(ops); err != nil {
		return Graph{}, err
	}
	tasks := make([]Task, 0, len(ops))
	edgeCount := 0
	for i, op := range ops {
		edgeCount += len(op.TaskBlockedBy)
		tasks = append(tasks, Task{
			ID:        op.TaskID,
			Index:     i,
			Path:      util.NormalizeRelPath(op.Path),
			Op:        op.Op,
			BlockedBy: append([]string(nil), op.TaskBlockedBy...),
		})
	}
	g := Graph{
		V:         1,
		PlanID:    planID,
		TaskCount: len(tasks),
		EdgeCount: edgeCount,
		Tasks:     tasks,
	}
	g.Digest = digest(g)
	g.GraphID = g.Digest
	return g, nil
}

func Validate(g Graph) error {
	if g.V != 1 {
		return fmt.Errorf("task graph v must be 1")
	}
	if g.TaskCount != len(g.Tasks) {
		return fmt.Errorf("task_count mismatch")
	}
	seen := map[string]struct{}{}
	for i, t := range g.Tasks {
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("task[%d] missing id", i)
		}
		if _, ok := seen[t.ID]; ok {
			return fmt.Errorf("duplicate task id %q", t.ID)
		}
		seen[t.ID] = struct{}{}
	}
	recomputed := digest(Graph{
		V:         g.V,
		PlanID:    g.PlanID,
		TaskCount: g.TaskCount,
		EdgeCount: g.EdgeCount,
		Tasks:     g.Tasks,
	})
	if recomputed != g.Digest {
		return fmt.Errorf("task graph digest mismatch")
	}
	return nil
}

func TopoOrder(ops []model.Operation) ([]model.Operation, error) {
	taskToIdx := map[string]int{}
	hasTaskID := false
	hasDeps := false
	for i, op := range ops {
		if strings.TrimSpace(op.TaskID) == "" {
			continue
		}
		hasTaskID = true
		if _, ok := taskToIdx[op.TaskID]; ok {
			return nil, fmt.Errorf("duplicate task_id: %s", op.TaskID)
		}
		taskToIdx[op.TaskID] = i
		if len(op.TaskBlockedBy) > 0 {
			hasDeps = true
		}
	}
	if !hasTaskID {
		return ops, nil
	}
	if !hasDeps {
		return ops, nil
	}

	indeg := make([]int, len(ops))
	outs := make([][]int, len(ops))
	for i, op := range ops {
		for _, depID := range op.TaskBlockedBy {
			j, ok := taskToIdx[depID]
			if !ok {
				return nil, fmt.Errorf("unknown dependency %q for task %q", depID, op.TaskID)
			}
			if j == i {
				return nil, fmt.Errorf("self dependency for task %q", op.TaskID)
			}
			indeg[i]++
			outs[j] = append(outs[j], i)
		}
	}

	ready := make([]int, 0, len(ops))
	for i := range ops {
		if indeg[i] == 0 {
			ready = append(ready, i)
		}
	}
	sort.Ints(ready)

	orderedIdx := make([]int, 0, len(ops))
	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]
		orderedIdx = append(orderedIdx, i)
		for _, to := range outs[i] {
			indeg[to]--
			if indeg[to] == 0 {
				ready = insertSorted(ready, to)
			}
		}
	}
	if len(orderedIdx) != len(ops) {
		return nil, fmt.Errorf("task graph cycle detected")
	}

	ordered := make([]model.Operation, 0, len(ops))
	for _, i := range orderedIdx {
		ordered = append(ordered, ops[i])
	}
	return ordered, nil
}

func conflicts(a, b model.Operation) bool {
	pa := util.NormalizeRelPath(a.Path)
	pb := util.NormalizeRelPath(b.Path)
	if pa == "" || pb == "" {
		return false
	}
	if pa == pb {
		return true
	}
	da := isDestructive(a.Op)
	db := isDestructive(b.Op)
	if !(da || db) {
		return false
	}
	if isAncestorPath(pa, pb) || isAncestorPath(pb, pa) {
		return true
	}
	if path.Dir(pa) == path.Dir(pb) {
		return true
	}
	return false
}

func isDestructive(op string) bool {
	switch strings.TrimSpace(strings.ToLower(op)) {
	case "delete", "unlink", "write":
		return true
	default:
		return false
	}
}

func isAncestorPath(parent, child string) bool {
	if parent == child {
		return true
	}
	return strings.HasPrefix(child+"/", parent+"/")
}

func reduceTransitiveDeps(incoming []map[int]struct{}) {
	outgoing := make([]map[int]struct{}, len(incoming))
	for i := range outgoing {
		outgoing[i] = map[int]struct{}{}
	}
	for to, deps := range incoming {
		for from := range deps {
			outgoing[from][to] = struct{}{}
		}
	}

	for target := range incoming {
		if len(incoming[target]) <= 1 {
			continue
		}
		deps := make([]int, 0, len(incoming[target]))
		for d := range incoming[target] {
			deps = append(deps, d)
		}
		sort.Ints(deps)
		for _, d := range deps {
			if _, ok := incoming[target][d]; !ok {
				continue
			}
			for _, other := range deps {
				if d == other {
					continue
				}
				if reachable(other, d, outgoing, target) {
					delete(incoming[target], d)
					break
				}
			}
		}
	}
}

func reachable(start, goal int, outgoing []map[int]struct{}, maxNode int) bool {
	if start == goal {
		return true
	}
	seen := map[int]struct{}{start: {}}
	stack := []int{start}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for next := range outgoing[n] {
			if next >= maxNode {
				continue
			}
			if next == goal {
				return true
			}
			if _, ok := seen[next]; ok {
				continue
			}
			seen[next] = struct{}{}
			stack = append(stack, next)
		}
	}
	return false
}

func insertSorted(in []int, v int) []int {
	i := sort.SearchInts(in, v)
	in = append(in, 0)
	copy(in[i+1:], in[i:])
	in[i] = v
	return in
}

func digest(g Graph) string {
	payload := map[string]any{
		"v":          g.V,
		"plan_id":    g.PlanID,
		"task_count": g.TaskCount,
		"edge_count": g.EdgeCount,
		"tasks":      g.Tasks,
	}
	b, err := util.MarshalCanonical(payload)
	if err != nil {
		return ""
	}
	return util.SHA256Tagged(b)
}
