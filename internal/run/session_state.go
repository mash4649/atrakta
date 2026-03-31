package run

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
	"github.com/mash4649/atrakta/v0/internal/validation"
)

const (
	sessionStateSchemaVersion          = "session-state.v1"
	sessionStateSchemaVersionLegacy    = "session-state.v0"
	sessionProgressSchemaVersion       = "session-progress.v1"
	sessionProgressSchemaVersionLegacy = "session-progress.v0"
	sessionTaskGraphSchemaVersion      = "session-task-graph.v1"
)

type SessionState struct {
	SchemaVersion     string `json:"schema_version"`
	Command           string `json:"command"`
	CanonicalState    string `json:"canonical_state"`
	Status            string `json:"status"`
	InterfaceID       string `json:"interface_id"`
	InterfaceSource   string `json:"interface_source"`
	PortabilityStatus string `json:"portability_status,omitempty"`
	PortabilityReason string `json:"portability_reason,omitempty"`
	ApplyRequested    bool   `json:"apply_requested"`
	Approved          bool   `json:"approved"`
	PlannedCount      int    `json:"planned_count"`
	AppliedCount      int    `json:"applied_count"`
}

type SessionProgress struct {
	SchemaVersion     string `json:"schema_version"`
	Command           string `json:"command"`
	Status            string `json:"status"`
	NextAllowedAction string `json:"next_allowed_action,omitempty"`
	PlannedCount      int    `json:"planned_count"`
	AppliedCount      int    `json:"applied_count"`
}

type SessionTaskGraph struct {
	SchemaVersion string            `json:"schema_version"`
	Command       string            `json:"command"`
	Nodes         []SessionTaskNode `json:"nodes"`
}

type SessionTaskNode struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"`
	Status string `json:"status"`
}

func SessionStatePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "state.json")
}

func SessionProgressPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "progress.json")
}

func SessionTaskGraphPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".atrakta", "task-graph.json")
}

func SaveSessionState(projectRoot string, state SessionState) error {
	state.SchemaVersion = sessionStateSchemaVersion
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := validation.ValidateSessionStateRaw(raw); err != nil {
		return err
	}
	return writeSessionBytes(projectRoot, SessionStatePath(projectRoot), raw)
}

func SaveSessionProgress(projectRoot string, progress SessionProgress) error {
	progress.SchemaVersion = sessionProgressSchemaVersion
	raw, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		return err
	}
	if err := validation.ValidateSessionProgressRaw(raw); err != nil {
		return err
	}
	return writeSessionBytes(projectRoot, SessionProgressPath(projectRoot), raw)
}

func SaveSessionTaskGraph(projectRoot string, graph SessionTaskGraph) error {
	if strings.TrimSpace(graph.SchemaVersion) == "" {
		graph.SchemaVersion = sessionTaskGraphSchemaVersion
	}
	if existing, err := LoadSessionTaskGraph(projectRoot); err == nil {
		graph = mergeSessionTaskGraph(existing, graph)
	}
	return writeSessionJSON(projectRoot, SessionTaskGraphPath(projectRoot), graph)
}

func LoadSessionTaskGraph(projectRoot string) (SessionTaskGraph, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return SessionTaskGraph{}, err
	}
	raw, err := os.ReadFile(SessionTaskGraphPath(root))
	if err != nil {
		return SessionTaskGraph{}, err
	}
	var graph SessionTaskGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		return SessionTaskGraph{}, fmt.Errorf("decode session task graph: %w", err)
	}
	return graph, nil
}

func LoadSessionState(projectRoot string) (SessionState, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return SessionState{}, err
	}
	raw, err := os.ReadFile(SessionStatePath(root))
	if err != nil {
		return SessionState{}, err
	}
	if err := validation.ValidateSessionStateRaw(raw); err != nil {
		return SessionState{}, err
	}
	var state SessionState
	if err := json.Unmarshal(raw, &state); err != nil {
		return SessionState{}, err
	}
	return state, nil
}

func LoadSessionProgress(projectRoot string) (SessionProgress, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return SessionProgress{}, err
	}
	raw, err := os.ReadFile(SessionProgressPath(root))
	if err != nil {
		return SessionProgress{}, err
	}
	if err := validation.ValidateSessionProgressRaw(raw); err != nil {
		return SessionProgress{}, err
	}
	var progress SessionProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return SessionProgress{}, err
	}
	return progress, nil
}

func writeSessionJSON(projectRoot, path string, payload any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeSessionBytes(projectRoot, path, raw)
}

func writeSessionBytes(projectRoot, path string, raw []byte) error {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return err
	}
	normalized := path
	if !filepath.IsAbs(normalized) {
		normalized = filepath.Join(root, normalized)
	}
	if err := os.MkdirAll(filepath.Dir(normalized), 0o755); err != nil {
		return err
	}
	return os.WriteFile(normalized, append(raw, '\n'), 0o644)
}

func mergeSessionTaskGraph(existing, next SessionTaskGraph) SessionTaskGraph {
	if strings.TrimSpace(next.SchemaVersion) == "" {
		next.SchemaVersion = existing.SchemaVersion
	}
	if strings.TrimSpace(next.Command) == "" {
		next.Command = existing.Command
	}
	if len(existing.Nodes) == 0 {
		return next
	}
	if len(next.Nodes) == 0 {
		next.Nodes = append([]SessionTaskNode(nil), existing.Nodes...)
		return next
	}
	existingByKey := make(map[string]SessionTaskNode, len(existing.Nodes))
	for _, node := range existing.Nodes {
		existingByKey[taskNodeKey(node)] = node
	}
	merged := make([]SessionTaskNode, 0, len(next.Nodes))
	for _, node := range next.Nodes {
		key := taskNodeKey(node)
		if prev, ok := existingByKey[key]; ok {
			node.Status = mergeTaskNodeStatus(prev.Status, node.Status)
		}
		merged = append(merged, node)
		delete(existingByKey, key)
	}
	next.Nodes = merged
	return next
}

func taskNodeKey(node SessionTaskNode) string {
	if key := strings.TrimSpace(node.Target); key != "" {
		return key
	}
	return strings.TrimSpace(node.ID)
}

func mergeTaskNodeStatus(existingStatus, nextStatus string) string {
	if nextStatus == "" {
		return existingStatus
	}
	if existingStatus == "" {
		return nextStatus
	}
	if taskNodeStatusRank(nextStatus) >= taskNodeStatusRank(existingStatus) {
		return nextStatus
	}
	return existingStatus
}

func taskNodeStatusRank(status string) int {
	switch strings.TrimSpace(status) {
	case "planned":
		return 1
	case "applied", "done", "completed", "skipped", "blocked":
		return 2
	default:
		if strings.TrimSpace(status) != "" {
			return 2
		}
		return 0
	}
}
