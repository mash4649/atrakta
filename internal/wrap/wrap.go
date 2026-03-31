package wrap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mash4649/atrakta/v0/internal/onboarding"
)

type Binding struct {
	ID                    string   `json:"id"`
	Kind                  string   `json:"kind,omitempty"`
	Surfaces              []string `json:"surfaces,omitempty"`
	ProjectionTargets     []string `json:"projection_targets,omitempty"`
	IngestSources         []string `json:"ingest_sources,omitempty"`
	ApprovalChannel       string   `json:"approval_channel,omitempty"`
	PortabilityMode       string   `json:"portability_mode,omitempty"`
	CanMutateCoreContract bool     `json:"can_mutate_core_contract,omitempty"`

	InstallPath    string   `json:"install_path"`
	ScriptTemplate string   `json:"script_template"`
	Capabilities   []string `json:"capabilities"`
}

type InstallPlan struct {
	ToolID       string
	ProjectRoot  string
	RepoRoot     string
	BindingPath  string
	InstallPath  string
	Script       string
	Capabilities []string
}

type UninstallPlan struct {
	ToolID      string
	ProjectRoot string
	RepoRoot    string
	BindingPath string
	InstallPath string
}

const managedScriptMarker = "# managed by atrakta wrap install"

func BuildInstallPlan(projectRoot, toolID string) (InstallPlan, error) {
	if strings.TrimSpace(toolID) == "" {
		return InstallPlan{}, fmt.Errorf("tool id required")
	}
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return InstallPlan{}, err
	}
	binding, bindingPath, repoRoot, err := LoadBinding(toolID)
	if err != nil {
		return InstallPlan{}, err
	}
	installPath, err := resolveInstallPath(root, binding.InstallPath)
	if err != nil {
		return InstallPlan{}, err
	}
	script, err := renderScript(binding, root, installPath)
	if err != nil {
		return InstallPlan{}, err
	}
	return InstallPlan{
		ToolID:       binding.ID,
		ProjectRoot:  root,
		RepoRoot:     repoRoot,
		BindingPath:  bindingPath,
		InstallPath:  installPath,
		Script:       script,
		Capabilities: append([]string(nil), binding.Capabilities...),
	}, nil
}

func BuildUninstallPlan(projectRoot, toolID string) (UninstallPlan, error) {
	if strings.TrimSpace(toolID) == "" {
		return UninstallPlan{}, fmt.Errorf("tool id required")
	}
	root, err := onboarding.DetectProjectRoot(projectRoot)
	if err != nil {
		return UninstallPlan{}, err
	}
	binding, bindingPath, repoRoot, err := LoadBinding(toolID)
	if err != nil {
		return UninstallPlan{}, err
	}
	installPath, err := resolveInstallPath(root, binding.InstallPath)
	if err != nil {
		return UninstallPlan{}, err
	}
	raw, err := os.ReadFile(installPath)
	if err != nil {
		if os.IsNotExist(err) {
			return UninstallPlan{}, fmt.Errorf("wrap uninstall target for %q not found at %s", toolID, installPath)
		}
		return UninstallPlan{}, err
	}
	if !isManagedInstallScript(raw, binding.ID) {
		return UninstallPlan{}, fmt.Errorf("wrap uninstall target %s is not managed by atrakta", installPath)
	}
	return UninstallPlan{
		ToolID:      binding.ID,
		ProjectRoot: root,
		RepoRoot:    repoRoot,
		BindingPath: bindingPath,
		InstallPath: installPath,
	}, nil
}

func LoadBinding(toolID string) (Binding, string, string, error) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		return Binding{}, "", "", err
	}
	path := filepath.Join(repoRoot, "adapters", "bindings", toolID, "binding.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Binding{}, "", "", fmt.Errorf("wrap binding for %q not found", toolID)
		}
		return Binding{}, "", "", err
	}

	var binding Binding
	if err := json.Unmarshal(raw, &binding); err != nil {
		return Binding{}, "", "", fmt.Errorf("parse wrap binding %s: %w", path, err)
	}
	if strings.TrimSpace(binding.ID) == "" {
		binding.ID = toolID
	}
	if strings.TrimSpace(binding.InstallPath) == "" {
		return Binding{}, "", "", fmt.Errorf("wrap binding %q missing install_path", toolID)
	}
	if strings.TrimSpace(binding.ScriptTemplate) == "" {
		return Binding{}, "", "", fmt.Errorf("wrap binding %q missing script_template", toolID)
	}
	if len(binding.Capabilities) == 0 {
		return Binding{}, "", "", fmt.Errorf("wrap binding %q missing capabilities", toolID)
	}
	return binding, path, repoRoot, nil
}

func WriteInstallPlan(plan InstallPlan) error {
	dir := filepath.Dir(plan.InstallPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(plan.InstallPath, []byte(plan.Script), 0o755)
}

func RemoveUninstallPlan(plan UninstallPlan) error {
	return os.Remove(plan.InstallPath)
}

func RunWrapped(toolID string, args []string) (int, error) {
	toolID = strings.TrimSpace(toolID)
	if toolID == "" {
		return 1, fmt.Errorf("tool id required")
	}

	root, err := onboarding.DetectProjectRoot("")
	if err != nil {
		return 1, err
	}

	binding, _, _, err := LoadBinding(toolID)
	if err != nil {
		return 1, err
	}

	installPath, err := resolveInstallPath(root, binding.InstallPath)
	if err != nil {
		return 1, err
	}

	raw, err := os.ReadFile(installPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, fmt.Errorf("wrap run target for %q not found at %s; run `atrakta wrap install %s` first", toolID, installPath, toolID)
		}
		return 1, err
	}
	if !isManagedInstallScript(raw, binding.ID) {
		return 1, fmt.Errorf("wrap run target %s is not managed by atrakta", installPath)
	}

	cmd := exec.Command(installPath, args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code == 0 {
				code = 1
			}
			return code, fmt.Errorf("wrap run exited with code %d", code)
		}
		return 1, err
	}

	return 0, nil
}

func resolveRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found")
}

func resolveInstallPath(projectRoot, configuredPath string) (string, error) {
	configuredPath = strings.TrimSpace(configuredPath)
	if configuredPath == "" {
		return "", fmt.Errorf("install_path required")
	}
	if filepath.IsAbs(configuredPath) {
		return filepath.Clean(configuredPath), nil
	}
	return filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(configuredPath))), nil
}

func renderScript(binding Binding, projectRoot, installPath string) (string, error) {
	capabilities := strings.Join(binding.Capabilities, ",")
	replacements := map[string]string{
		"{{tool_id}}":          binding.ID,
		"{{project_root}}":     projectRoot,
		"{{install_path}}":     installPath,
		"{{capabilities}}":     capabilities,
		"{{capabilities_csv}}": capabilities,
	}
	script := binding.ScriptTemplate
	for key, value := range replacements {
		script = strings.ReplaceAll(script, key, value)
	}
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}
	return script, nil
}

func isManagedInstallScript(raw []byte, toolID string) bool {
	script := string(raw)
	if !strings.Contains(script, managedScriptMarker) {
		return false
	}
	if !strings.Contains(script, "# tool_id: "+toolID) {
		return false
	}
	return true
}
