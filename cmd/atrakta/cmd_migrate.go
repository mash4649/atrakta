package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

type migrateCheckResult struct {
	ProjectRoot string               `json:"project_root"`
	Command     string               `json:"command"`
	OK          bool                 `json:"ok"`
	Checks      []migrateCheckRecord `json:"checks"`
	Guidance    []string             `json:"guidance,omitempty"`
	Message     string               `json:"message"`
}

type migrateCheckRecord struct {
	Name          string   `json:"name"`
	Status        string   `json:"status"`
	SchemaVersion string   `json:"schema_version,omitempty"`
	Detail        string   `json:"detail"`
	Guidance      []string `json:"guidance,omitempty"`
}

func runMigrate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: atrakta migrate check [flags]")
	}
	sub := strings.TrimSpace(args[0])
	switch sub {
	case "check":
		return runMigrateCheck(args[1:])
	default:
		return fmt.Errorf("unsupported migrate subcommand %q", sub)
	}
}

func runMigrateCheck(args []string) error {
	fs := flag.NewFlagSet("migrate check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var projectRoot string
	var jsonOut bool
	var artifactDir string

	fs.StringVar(&projectRoot, "project-root", "", "project root")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable output")
	fs.StringVar(&artifactDir, "artifact-dir", "", "directory for JSON artifacts")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}

	records := []migrateCheckRecord{}
	records = append(records, checkSessionSchema(filepath.Join(root, ".atrakta", "state.json"), "state.json", "session-state.v1", []string{"session-state.v0"}, []string{
		"backup `.atrakta/state.json` and rerun `atrakta start` to regenerate the v1 session state",
	}))
	records = append(records, checkSessionSchema(filepath.Join(root, ".atrakta", "progress.json"), "progress.json", "session-progress.v1", []string{"session-progress.v0"}, []string{
		"backup `.atrakta/progress.json` and rerun `atrakta start` / `atrakta resume` to regenerate the v1 progress file",
	}))
	records = append(records, checkSessionSchema(filepath.Join(root, ".atrakta", "task-graph.json"), "task-graph.json", "session-task-graph.v1", []string{"session-task-graph.v0"}, []string{
		"backup `.atrakta/task-graph.json` and rerun `atrakta start` / `atrakta resume` to regenerate the v1 task graph",
	}))
	records = append(records, checkRunEventsSchema(filepath.Join(root, ".atrakta", "audit", "events", "run-events.jsonl"), []string{
		"keep the legacy `/.atrakta/events.jsonl` stream read-only",
		"regenerate `.atrakta/audit/events/run-events.jsonl` by rerunning `atrakta start`",
	}))

	ok := true
	for _, r := range records {
		if r.Status != "compatible" {
			ok = false
		}
	}

	guidance := collectMigrateGuidance(records)
	result := migrateCheckResult{
		ProjectRoot: root,
		Command:     "migrate check",
		OK:          ok,
		Checks:      records,
		Guidance:    guidance,
	}
	if ok {
		result.Message = "migrate check passed"
	} else {
		result.Message = "migrate check found migration work"
	}
	return emitMigrateCheckResult(result, jsonOut, artifactDir)
}

func checkSessionSchema(path, name, expected string, legacy []string, guidance []string) migrateCheckRecord {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return migrateCheckRecord{Name: "session." + name, Status: "unknown", Detail: "file not found"}
		}
		return migrateCheckRecord{Name: "session." + name, Status: "unknown", Detail: err.Error()}
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return migrateCheckRecord{Name: "session." + name, Status: "unknown", Detail: "invalid json"}
	}
	got, _ := payload["schema_version"].(string)
	if strings.TrimSpace(got) == "" {
		return migrateCheckRecord{Name: "session." + name, Status: "unknown", Detail: "missing schema_version"}
	}
	if got == expected {
		return migrateCheckRecord{Name: "session." + name, Status: "compatible", SchemaVersion: got, Detail: got}
	}
	if hasString(legacy, got) {
		return migrateCheckRecord{Name: "session." + name, Status: "needs_migration", SchemaVersion: got, Detail: fmt.Sprintf("expected %s, got %s", expected, got), Guidance: guidance}
	}
	return migrateCheckRecord{Name: "session." + name, Status: "unknown", SchemaVersion: got, Detail: fmt.Sprintf("unrecognized schema_version %s", got)}
}

func checkRunEventsSchema(path string, guidance []string) migrateCheckRecord {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: "run-events.jsonl not found"}
		}
		return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: err.Error()}
	}
	defer f.Close()

	type versionSeen struct {
		value int
	}
	var seen *versionSeen
	line := 0
	empty := true
	buf := make([]byte, 0, 64*1024)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		empty = false
		var row map[string]any
		if err := json.Unmarshal([]byte(text), &row); err != nil {
			return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: fmt.Sprintf("invalid json at line %d", line)}
		}
		version, ok := row["schema_version"]
		if !ok {
			return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: fmt.Sprintf("missing schema_version at line %d", line)}
		}
		intVersion, ok := asInt(version)
		if !ok {
			return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: fmt.Sprintf("invalid schema_version at line %d", line)}
		}
		if seen == nil {
			seen = &versionSeen{value: intVersion}
		} else if seen.value != intVersion {
			return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: fmt.Sprintf("mixed schema_version values at line %d", line)}
		}
	}
	if err := scanner.Err(); err != nil {
		return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: err.Error()}
	}
	if empty || seen == nil {
		return migrateCheckRecord{Name: "events.run-events", Status: "unknown", Detail: "run-events.jsonl is empty"}
	}
	if seen.value == 1 {
		return migrateCheckRecord{Name: "events.run-events", Status: "compatible", SchemaVersion: "1", Detail: "schema_version=1"}
	}
	if seen.value == 2 {
		return migrateCheckRecord{Name: "events.run-events", Status: "needs_migration", SchemaVersion: "2", Detail: "schema_version=2", Guidance: guidance}
	}
	return migrateCheckRecord{Name: "events.run-events", Status: "unknown", SchemaVersion: fmt.Sprint(seen.value), Detail: fmt.Sprintf("unrecognized schema_version %d", seen.value)}
}

func emitMigrateCheckResult(result migrateCheckResult, jsonOut bool, artifactDir string) error {
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			return err
		}
	} else {
		fmt.Printf("project_root: %s\n", result.ProjectRoot)
		fmt.Printf("command: %s\n", result.Command)
		fmt.Printf("ok: %t\n", result.OK)
		fmt.Printf("checks: %d\n", len(result.Checks))
		fmt.Printf("message: %s\n", result.Message)
		for _, check := range result.Checks {
			fmt.Printf(" - %s: %s (%s)\n", check.Name, check.Status, check.Detail)
			for _, g := range check.Guidance {
				fmt.Printf("   guidance: %s\n", g)
			}
		}
	}
	if artifactDir != "" {
		if err := writeArtifact(artifactDir, "migrate.check.result.json", result); err != nil {
			return err
		}
	}
	return nil
}

func collectMigrateGuidance(records []migrateCheckRecord) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, record := range records {
		for _, guidance := range record.Guidance {
			guidance = strings.TrimSpace(guidance)
			if guidance == "" {
				continue
			}
			if _, ok := seen[guidance]; ok {
				continue
			}
			seen[guidance] = struct{}{}
			out = append(out, guidance)
		}
	}
	return out
}

func hasString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case float32:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}
