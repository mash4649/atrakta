package ide

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/util"
)

const (
	managedDetail = "managed-by-atrakta"
	managedLabel  = "Atrakta: Auto Start"
)

type Status struct {
	Path          string
	FileExists    bool
	Installed     bool
	ManagedTaskN  int
	OtherTaskN    int
	ParseErrorMsg string
}

func Install(repoRoot string) (bool, string, error) {
	path := tasksPath(repoRoot)
	doc, err := loadOrNew(path)
	if err != nil {
		return false, path, err
	}
	changed := false
	if version, _ := doc["version"].(string); strings.TrimSpace(version) == "" {
		doc["version"] = "2.0.0"
		changed = true
	}
	tasks, err := extractTasks(doc)
	if err != nil {
		return false, path, err
	}
	managedFound, managedEquivalent := scanManagedTasks(tasks)
	if managedFound == 1 && managedEquivalent && !changed {
		return false, path, nil
	}
	out := make([]any, 0, len(tasks)+1)
	for _, raw := range tasks {
		task, ok := raw.(map[string]any)
		if ok && isManagedTask(task) {
			changed = true
			continue
		}
		out = append(out, raw)
	}
	out = append(out, canonicalManagedTask())
	doc["tasks"] = out
	if err := writeDoc(path, doc); err != nil {
		return false, path, err
	}
	return true, path, nil
}

func Uninstall(repoRoot string) (bool, string, error) {
	path := tasksPath(repoRoot)
	doc, err := loadOrNew(path)
	if err != nil {
		return false, path, err
	}
	tasks, err := extractTasks(doc)
	if err != nil {
		return false, path, err
	}
	changed := false
	out := make([]any, 0, len(tasks))
	for _, raw := range tasks {
		task, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		if isManagedTask(task) {
			changed = true
			continue
		}
		out = append(out, raw)
	}
	if !changed {
		return false, path, nil
	}
	doc["tasks"] = out
	if err := writeDoc(path, doc); err != nil {
		return false, path, err
	}
	return true, path, nil
}

func Check(repoRoot string) Status {
	path := tasksPath(repoRoot)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Status{Path: path}
		}
		return Status{Path: path, ParseErrorMsg: err.Error()}
	}
	st := Status{Path: path, FileExists: true}
	doc := map[string]any{}
	if err := json.Unmarshal(b, &doc); err != nil {
		st.ParseErrorMsg = err.Error()
		return st
	}
	tasks, err := extractTasks(doc)
	if err != nil {
		st.ParseErrorMsg = err.Error()
		return st
	}
	for _, raw := range tasks {
		task, ok := raw.(map[string]any)
		if ok && isManagedTask(task) {
			st.ManagedTaskN++
		} else {
			st.OtherTaskN++
		}
	}
	st.Installed = st.ManagedTaskN > 0
	return st
}

func tasksPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".vscode", "tasks.json")
}

func loadOrNew(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{
				"version": "2.0.0",
				"tasks":   []any{},
			}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return doc, nil
}

func extractTasks(doc map[string]any) ([]any, error) {
	raw, ok := doc["tasks"]
	if !ok || raw == nil {
		return []any{}, nil
	}
	tasks, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("tasks must be array")
	}
	return tasks, nil
}

func scanManagedTasks(tasks []any) (count int, equivalent bool) {
	for _, raw := range tasks {
		task, ok := raw.(map[string]any)
		if !ok || !isManagedTask(task) {
			continue
		}
		count++
		if isEquivalentManagedTask(task) {
			equivalent = true
		}
	}
	return count, equivalent
}

func writeDoc(path string, doc map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	b = append(b, '\n')
	if err := util.AtomicWriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func isManagedTask(task map[string]any) bool {
	if d, _ := task["detail"].(string); strings.TrimSpace(d) == managedDetail {
		return true
	}
	if l, _ := task["label"].(string); strings.TrimSpace(l) == managedLabel {
		return true
	}
	return false
}

func isEquivalentManagedTask(task map[string]any) bool {
	if !isManagedTask(task) {
		return false
	}
	if t, _ := task["type"].(string); strings.TrimSpace(t) != "shell" {
		return false
	}
	if c, _ := task["command"].(string); strings.TrimSpace(c) != "atrakta" {
		return false
	}
	args, ok := task["args"].([]any)
	if !ok || len(args) != 1 {
		return false
	}
	a0, _ := args[0].(string)
	if strings.TrimSpace(a0) != "start" {
		return false
	}
	runOptions, ok := task["runOptions"].(map[string]any)
	if !ok {
		return false
	}
	runOn, _ := runOptions["runOn"].(string)
	return strings.TrimSpace(runOn) == "folderOpen"
}

func canonicalManagedTask() map[string]any {
	return map[string]any{
		"label":   managedLabel,
		"type":    "shell",
		"command": "atrakta",
		"args":    []any{"start"},
		"runOptions": map[string]any{
			"runOn": "folderOpen",
		},
		"problemMatcher": []any{},
		"presentation": map[string]any{
			"reveal": "never",
			"panel":  "dedicated",
		},
		"detail": managedDetail,
	}
}
