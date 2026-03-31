package hook

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

const managedHookMarker = "# managed by atrakta hook install"

var supportedHookTypes = []string{"pre-commit", "post-checkout"}

type InstallPlan struct {
	HookType    string
	ProjectRoot string
	RepoRoot    string
	HookPath    string
	Script      string
}

type UninstallPlan struct {
	HookType      string
	ProjectRoot   string
	RepoRoot      string
	HookPath      string
	Exists        bool
	Managed       bool
	CurrentScript string
}

type StatusPlan struct {
	HookType       string
	ProjectRoot    string
	RepoRoot       string
	HookPath       string
	Exists         bool
	Managed        bool
	Drift          bool
	Status         string
	ExpectedScript string
	CurrentScript  string
}

func SupportedHookTypes() []string {
	return append([]string(nil), supportedHookTypes...)
}

func BuildInstallPlan(projectRoot, hookType string) (InstallPlan, error) {
	hookType = normalizeHookType(hookType)
	if hookType == "" {
		return InstallPlan{}, fmt.Errorf("hook type required")
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return InstallPlan{}, err
	}
	hookPath := filepath.Join(root, ".git", "hooks", hookType)
	script := renderScript(hookType, root, hookPath)
	return InstallPlan{
		HookType:    hookType,
		ProjectRoot: root,
		RepoRoot:    root,
		HookPath:    hookPath,
		Script:      script,
	}, nil
}

func LoadExistingHook(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func WriteInstallPlan(plan InstallPlan) error {
	if err := os.MkdirAll(filepath.Dir(plan.HookPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plan.HookPath, []byte(plan.Script), 0o755)
}

func BuildUninstallPlan(projectRoot, hookType string) (UninstallPlan, error) {
	hookType = normalizeHookType(hookType)
	if hookType == "" {
		return UninstallPlan{}, fmt.Errorf("hook type required")
	}

	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return UninstallPlan{}, err
	}
	hookPath := filepath.Join(root, ".git", "hooks", hookType)
	plan := UninstallPlan{
		HookType:    hookType,
		ProjectRoot: root,
		RepoRoot:    root,
		HookPath:    hookPath,
	}

	raw, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return plan, nil
		}
		return UninstallPlan{}, err
	}
	plan.Exists = true
	plan.CurrentScript = string(raw)
	plan.Managed = IsManagedHookScript(raw, hookType)
	return plan, nil
}

func RemoveUninstallPlan(plan UninstallPlan) error {
	if !plan.Exists {
		return nil
	}
	if !plan.Managed {
		return fmt.Errorf("hook uninstall target %s is not managed by atrakta", plan.HookPath)
	}
	return os.Remove(plan.HookPath)
}

func BuildStatusPlans(projectRoot string) ([]StatusPlan, error) {
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return nil, err
	}

	plans := make([]StatusPlan, 0, len(supportedHookTypes))
	for _, hookType := range supportedHookTypes {
		installPlan, err := BuildInstallPlan(root, hookType)
		if err != nil {
			return nil, err
		}

		plan := StatusPlan{
			HookType:       installPlan.HookType,
			ProjectRoot:    installPlan.ProjectRoot,
			RepoRoot:       installPlan.RepoRoot,
			HookPath:       installPlan.HookPath,
			ExpectedScript: installPlan.Script,
			Status:         "missing",
		}

		raw, err := os.ReadFile(installPlan.HookPath)
		if err != nil {
			if os.IsNotExist(err) {
				plans = append(plans, plan)
				continue
			}
			return nil, err
		}

		plan.Exists = true
		plan.CurrentScript = string(raw)
		plan.Managed = IsManagedHookScript(raw, hookType)
		plan.Drift = !bytes.Equal(raw, []byte(installPlan.Script))
		switch {
		case plan.Drift:
			plan.Status = "drift"
		default:
			plan.Status = "up_to_date"
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

func BuildRepairPlans(projectRoot string) ([]StatusPlan, error) {
	plans, err := BuildStatusPlans(projectRoot)
	if err != nil {
		return nil, err
	}

	drifted := make([]StatusPlan, 0, len(plans))
	for _, plan := range plans {
		if plan.Status == "drift" {
			drifted = append(drifted, plan)
		}
	}
	return drifted, nil
}

func RepairStatusPlan(plan StatusPlan) error {
	if strings.TrimSpace(plan.HookType) == "" || strings.TrimSpace(plan.HookPath) == "" || strings.TrimSpace(plan.ExpectedScript) == "" {
		return fmt.Errorf("repair plan incomplete")
	}
	if err := os.MkdirAll(filepath.Dir(plan.HookPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plan.HookPath, []byte(plan.ExpectedScript), 0o755)
}

func IsManagedHookScript(raw []byte, hookType string) bool {
	script := string(raw)
	if !strings.Contains(script, managedHookMarker) {
		return false
	}
	if !strings.Contains(script, "# hook_type: "+normalizeHookType(hookType)) {
		return false
	}
	return true
}

func normalizeHookType(hookType string) string {
	switch strings.TrimSpace(hookType) {
	case "pre-commit", "post-checkout":
		return strings.TrimSpace(hookType)
	default:
		return ""
	}
}

func renderScript(hookType, projectRoot, hookPath string) string {
	script := strings.Join([]string{
		"#!/usr/bin/env sh",
		managedHookMarker,
		"# hook_type: " + hookType,
		"# project_root: " + projectRoot,
		"# hook_path: " + hookPath,
		"exec atrakta start \"$@\"",
		"",
	}, "\n")
	return script
}
