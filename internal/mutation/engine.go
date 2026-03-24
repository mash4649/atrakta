package mutation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/audit"
	checkmutationscope "github.com/mash4649/atrakta/v0/resolvers/mutation/check-mutation-scope"
)

// DecisionEnvelope is the inspect/propose/apply decision contract.
type DecisionEnvelope struct {
	DecisionID        string   `json:"decision_id"`
	Phase             string   `json:"phase"`
	TargetPath        string   `json:"target_path"`
	Scope             string   `json:"scope"`
	Policy            string   `json:"policy"`
	AllowedModes      []string `json:"allowed_modes"`
	RequestedAction   string   `json:"requested_action"`
	Allowed           bool     `json:"allowed"`
	Reason            string   `json:"reason"`
	Evidence          []string `json:"evidence"`
	NextAllowedAction string   `json:"next_allowed_action"`
}

// Proposal is mutation propose output.
type Proposal struct {
	Envelope      DecisionEnvelope `json:"envelope"`
	ProposedPatch string           `json:"proposed_patch"`
}

// Inspect runs scope classification and returns decision envelope.
func Inspect(target checkmutationscope.Target) DecisionEnvelope {
	out := checkmutationscope.CheckMutationScope(target)
	decision := out.Decision.(checkmutationscope.MutationDecision)
	return DecisionEnvelope{
		DecisionID:        decisionID("inspect", target.Path),
		Phase:             "inspect",
		TargetPath:        target.Path,
		Scope:             decision.Scope,
		Policy:            decision.Policy,
		AllowedModes:      append([]string{}, decision.AllowedModes...),
		RequestedAction:   "inspect",
		Allowed:           true,
		Reason:            out.Reason,
		Evidence:          append([]string{}, out.Evidence...),
		NextAllowedAction: out.NextAllowedAction,
	}
}

// Propose returns proposal-first plan. It never writes files.
func Propose(target checkmutationscope.Target, content string) Proposal {
	env := Inspect(target)
	env.DecisionID = decisionID("propose", target.Path)
	env.Phase = "propose"
	env.RequestedAction = "propose"
	env.Allowed = contains(env.AllowedModes, "propose")
	if !env.Allowed {
		env.Reason = "proposal is not allowed for this scope"
		env.NextAllowedAction = "inspect"
	}

	patch := fmt.Sprintf("--- %s\n+++ %s\n@@\n+%s\n", target.Path, target.Path, strings.ReplaceAll(content, "\n", "\n+"))
	return Proposal{
		Envelope:      env,
		ProposedPatch: patch,
	}
}

// Apply performs managed-only mutation with explicit allow switch.
func Apply(projectRoot string, target checkmutationscope.Target, content string, allow bool) (DecisionEnvelope, error) {
	env := Inspect(target)
	env.DecisionID = decisionID("apply", target.Path)
	env.Phase = "apply"
	env.RequestedAction = "apply"
	env.Allowed = false

	if !allow {
		env.Reason = "apply requires explicit allow flag"
		env.NextAllowedAction = "propose"
		return env, errors.New(env.Reason)
	}
	if !contains(env.AllowedModes, "apply") {
		env.Reason = "apply is not allowed for resolved scope"
		env.NextAllowedAction = "propose"
		return env, errors.New(env.Reason)
	}
	if !isManagedScope(env.Scope) {
		env.Reason = "managed-only destructive mutation policy blocks apply"
		env.NextAllowedAction = "propose"
		return env, errors.New(env.Reason)
	}

	root := projectRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return env, err
		}
		root = wd
	}
	absTarget, err := resolveTargetPath(root, target.Path)
	if err != nil {
		env.Reason = err.Error()
		env.NextAllowedAction = "propose"
		return env, err
	}

	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return env, err
	}
	if err := os.WriteFile(absTarget, []byte(content), 0o644); err != nil {
		return env, err
	}

	if _, err := audit.AppendEvent(filepath.Join(root, ".atrakta", "audit"), audit.LevelA2, "apply_mutation", map[string]any{
		"target_path": target.Path,
		"scope":       env.Scope,
		"policy":      env.Policy,
	}); err != nil {
		return env, err
	}
	if err := audit.VerifyIntegrity(filepath.Join(root, ".atrakta", "audit"), audit.LevelA2); err != nil {
		return env, err
	}

	env.Allowed = true
	env.Reason = "mutation applied in managed scope"
	env.NextAllowedAction = "inspect"
	env.Evidence = append(env.Evidence, "write_target="+filepath.ToSlash(absTarget))
	return env, nil
}

func decisionID(phase, path string) string {
	sum := sha256.Sum256([]byte(phase + ":" + path))
	return hex.EncodeToString(sum[:12])
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func isManagedScope(scope string) bool {
	switch scope {
	case checkmutationscope.ScopeManagedBlock, checkmutationscope.ScopeGeneratedProjection, checkmutationscope.ScopeManagedInclude:
		return true
	default:
		return false
	}
}

func resolveTargetPath(projectRoot, target string) (string, error) {
	target = filepath.Clean(target)
	rootAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", err
	}
	var abs string
	if filepath.IsAbs(target) {
		abs = target
	} else {
		abs = filepath.Join(rootAbs, target)
	}
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("target escapes project root")
	}
	return abs, nil
}
