package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/contract"
	"atrakta/internal/util"
)

const DefaultPromptMinRef = ".atrakta/policies/prompt-min.json"

type PromptMin struct {
	ID                string `json:"id"`
	Apply             string `json:"apply"`
	RequireGoalPrefix bool   `json:"require_goal_prefix"`
	GoalLabel         string `json:"goal_label"`
	Required          bool   `json:"required"`
}

func DefaultPromptMin() PromptMin {
	return PromptMin{
		ID:                "prompt-min@1",
		Apply:             "conditional",
		RequireGoalPrefix: true,
		GoalLabel:         "Goal",
		Required:          false,
	}
}

func LoadPromptMin(repoRoot string, ref contract.PromptMinRef) (PromptMin, error) {
	path, err := resolvePolicyPath(repoRoot, ref.Ref)
	if err != nil {
		return PromptMin{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return PromptMin{}, err
	}
	var out PromptMin
	if err := json.Unmarshal(b, &out); err != nil {
		return PromptMin{}, fmt.Errorf("parse prompt policy: %w", err)
	}
	out.normalize()
	if out.ID == "" {
		return PromptMin{}, fmt.Errorf("prompt policy id required")
	}
	if out.Apply != "conditional" {
		return PromptMin{}, fmt.Errorf("prompt policy apply must be conditional")
	}
	return out, nil
}

func EnsureDefaultPromptMin(repoRoot, ref string) (bool, error) {
	if util.NormalizeRelPath(ref) != DefaultPromptMinRef {
		return false, nil
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(DefaultPromptMinRef))
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	b, err := json.MarshalIndent(DefaultPromptMin(), "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func ShouldApplyPromptMin(taskCategory string, machineOnly bool, p PromptMin) bool {
	if machineOnly {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(taskCategory)) {
	case "json", "patch", "export":
		return false
	}
	p.normalize()
	return p.Apply == "conditional"
}

func ApplyGoalPrefix(summary, details string, p PromptMin) (string, string, bool) {
	p.normalize()
	if !p.RequireGoalPrefix {
		return summary, details, false
	}
	prefix := p.GoalLabel + ": "
	if strings.TrimSpace(summary) != "" {
		if strings.HasPrefix(summary, prefix) {
			return summary, details, false
		}
		return prefix + summary, details, true
	}
	if strings.TrimSpace(details) != "" {
		if strings.HasPrefix(details, prefix) {
			return summary, details, false
		}
		return summary, prefix + details, true
	}
	return summary, details, false
}

func resolvePolicyPath(repoRoot, ref string) (string, error) {
	n := util.NormalizeRelPath(ref)
	if n == "" || strings.HasPrefix(n, "../") {
		return "", fmt.Errorf("policy ref must be repo-relative")
	}
	return filepath.Join(repoRoot, filepath.FromSlash(n)), nil
}

func (p *PromptMin) normalize() {
	p.ID = strings.TrimSpace(p.ID)
	p.Apply = strings.TrimSpace(strings.ToLower(p.Apply))
	if p.Apply == "" {
		p.Apply = "conditional"
	}
	p.GoalLabel = strings.TrimSpace(p.GoalLabel)
	if p.GoalLabel == "" {
		p.GoalLabel = "Goal"
	}
}
