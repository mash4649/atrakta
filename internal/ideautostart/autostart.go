package ideautostart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const (
	VSCodeTaskLabel = "atrakta start"
)

type FilePlan struct {
	Kind    string
	Path    string
	Exists  bool
	Changed bool
	Content []byte
}

type Plan struct {
	ProjectRoot string
	Files       []FilePlan
}

func BuildPlan(projectRoot string) (Plan, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return Plan{}, err
	}

	tasksPath := filepath.Join(root, ".vscode", "tasks.json")
	tasksContent, tasksExists, tasksChanged, err := buildVSCodeTasks(tasksPath, root)
	if err != nil {
		return Plan{}, err
	}

	cursorPath := filepath.Join(root, ".cursor", "autostart.json")
	cursorContent, cursorExists, cursorChanged, err := buildCursorAutostart(cursorPath, root)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		ProjectRoot: root,
		Files: []FilePlan{
			{
				Kind:    "vscode_tasks",
				Path:    tasksPath,
				Exists:  tasksExists,
				Changed: tasksChanged,
				Content: tasksContent,
			},
			{
				Kind:    "cursor_autostart",
				Path:    cursorPath,
				Exists:  cursorExists,
				Changed: cursorChanged,
				Content: cursorContent,
			},
		},
	}, nil
}

func WritePlan(plan Plan) error {
	for _, file := range plan.Files {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(file.Path, file.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func buildVSCodeTasks(path, root string) ([]byte, bool, bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, false, err
	}

	desired, err := renderVSCodeTasks(existing)
	if err != nil {
		return nil, false, false, err
	}

	exists := err == nil && len(existing) > 0
	return desired, exists, !bytes.Equal(existing, desired), nil
}

func renderVSCodeTasks(existing []byte) ([]byte, error) {
	doc := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return nil, fmt.Errorf("parse .vscode/tasks.json: %w", err)
		}
	}
	if doc == nil {
		doc = map[string]any{}
	}

	if _, ok := doc["version"]; !ok {
		doc["version"] = "2.0.0"
	}

	tasks, err := decodeTaskList(doc["tasks"])
	if err != nil {
		return nil, err
	}
	if !containsTask(tasks, VSCodeTaskLabel) {
		tasks = append(tasks, map[string]any{
			"label":   VSCodeTaskLabel,
			"type":    "shell",
			"command": "atrakta start",
			"presentation": map[string]any{
				"reveal": "always",
				"panel":  "shared",
			},
			"problemMatcher": []any{},
		})
	}
	doc["tasks"] = tasks

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func decodeTaskList(value any) ([]any, error) {
	if value == nil {
		return []any{}, nil
	}
	raw, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf(".vscode/tasks.json tasks must be an array")
	}
	tasks := make([]any, 0, len(raw))
	for _, item := range raw {
		tasks = append(tasks, item)
	}
	return tasks, nil
}

func containsTask(tasks []any, label string) bool {
	for _, item := range tasks {
		task, ok := item.(map[string]any)
		if !ok {
			continue
		}
		got, _ := task["label"].(string)
		if strings.TrimSpace(got) == label {
			return true
		}
	}
	return false
}

func buildCursorAutostart(path, root string) ([]byte, bool, bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, false, err
	}

	desired, err := renderCursorAutostart(existing, root)
	if err != nil {
		return nil, false, false, err
	}

	exists := err == nil && len(existing) > 0
	return desired, exists, !bytes.Equal(existing, desired), nil
}

func renderCursorAutostart(existing []byte, root string) ([]byte, error) {
	doc := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return nil, fmt.Errorf("parse .cursor/autostart.json: %w", err)
		}
	}
	if doc == nil {
		doc = map[string]any{}
	}

	doc["version"] = 1
	doc["source"] = "atrakta"
	doc["command"] = "atrakta start"
	doc["workspace_root"] = root

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
