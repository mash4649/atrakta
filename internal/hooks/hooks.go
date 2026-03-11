package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultShellSurface = "shell.on_cd"
	managedPrefix       = "# >>> ATRAKTA_MANAGED"
	managedSuffix       = "# <<< ATRAKTA_MANAGED"
)

var allSurfaces = []string{
	"shell.on_cd",
	"shell.on_exec",
	"git.pre_commit",
	"git.pre_push",
	"ide.on_open",
	"workflow.before_start",
	"workflow.after_apply",
}

type SurfaceStatus struct {
	Surface   string   `json:"surface"`
	Installed bool     `json:"installed"`
	Paths     []string `json:"paths,omitempty"`
}

type StatusReport struct {
	Surfaces []SurfaceStatus `json:"surfaces"`
}

func Install(selfExe string) error {
	return InstallForRepo("", selfExe, nil)
}

func InstallForRepo(repoRoot, selfExe string, surfaces []string) error {
	selected, err := normalizeSurfaces(surfaces, []string{defaultShellSurface})
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	atraktaHome := filepath.Join(home, ".atrakta")
	if err := os.MkdirAll(atraktaHome, 0o755); err != nil {
		return err
	}

	if hasSurfacePrefix(selected, "shell.") {
		scriptPath := filepath.Join(atraktaHome, "hook.sh")
		script := buildHookScript(selfExe)
		if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
			return fmt.Errorf("write hook script: %w", err)
		}
		for _, rc := range []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".bashrc")} {
			if err := ensureSourceLine(rc, scriptPath); err != nil {
				return err
			}
		}
		for _, s := range selected {
			if !strings.HasPrefix(s, "shell.") {
				continue
			}
			if err := ensureStageScript(filepath.Join(atraktaHome, "hooks.d", s, "20-atrakta.sh"), selfExe, s); err != nil {
				return err
			}
		}
	}

	if contains(selected, "git.pre_commit") {
		if err := installGitSurface(repoRoot, "pre-commit", "git.pre_commit", selfExe); err != nil {
			return err
		}
	}
	if contains(selected, "git.pre_push") {
		if err := installGitSurface(repoRoot, "pre-push", "git.pre_push", selfExe); err != nil {
			return err
		}
	}

	for _, s := range selected {
		if strings.HasPrefix(s, "ide.") || strings.HasPrefix(s, "workflow.") {
			if strings.TrimSpace(repoRoot) == "" {
				return fmt.Errorf("surface %s requires repository root", s)
			}
			p := filepath.Join(repoRoot, ".atrakta", "hooks.d", s, "20-atrakta.sh")
			if err := ensureStageScript(p, selfExe, s); err != nil {
				return err
			}
		}
	}

	fmt.Printf("hooks installed surfaces=%s\n", strings.Join(selected, ","))
	return nil
}

func Uninstall() error {
	return UninstallForRepo("", nil)
}

func UninstallForRepo(repoRoot string, surfaces []string) error {
	selected, err := normalizeSurfaces(surfaces, []string{defaultShellSurface})
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	atraktaHome := filepath.Join(home, ".atrakta")

	for _, s := range selected {
		if strings.HasPrefix(s, "shell.") {
			_ = os.Remove(filepath.Join(atraktaHome, "hooks.d", s, "20-atrakta.sh"))
		}
	}
	if hasSurfacePrefix(selected, "shell.") {
		if !hasAnyShellStage(atraktaHome) {
			scriptPath := filepath.Join(atraktaHome, "hook.sh")
			for _, rc := range []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".bashrc")} {
				if err := removeSourceLine(rc, scriptPath); err != nil {
					return err
				}
			}
			_ = os.Remove(scriptPath)
		}
	}

	if contains(selected, "git.pre_commit") {
		if err := uninstallGitSurface(repoRoot, "pre-commit", "git.pre_commit"); err != nil {
			return err
		}
	}
	if contains(selected, "git.pre_push") {
		if err := uninstallGitSurface(repoRoot, "pre-push", "git.pre_push"); err != nil {
			return err
		}
	}

	for _, s := range selected {
		if strings.HasPrefix(s, "ide.") || strings.HasPrefix(s, "workflow.") {
			if strings.TrimSpace(repoRoot) == "" {
				continue
			}
			_ = os.Remove(filepath.Join(repoRoot, ".atrakta", "hooks.d", s, "20-atrakta.sh"))
		}
	}

	fmt.Printf("hooks uninstalled surfaces=%s\n", strings.Join(selected, ","))
	return nil
}

func RepairForRepo(repoRoot, selfExe string, surfaces []string) error {
	return InstallForRepo(repoRoot, selfExe, surfaces)
}

func Status(repoRoot string, surfaces []string) (StatusReport, error) {
	selected, err := normalizeSurfaces(surfaces, allSurfaces)
	if err != nil {
		return StatusReport{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return StatusReport{}, err
	}
	atraktaHome := filepath.Join(home, ".atrakta")
	scriptPath := filepath.Join(atraktaHome, "hook.sh")
	sourceLine := "[ -f \"" + scriptPath + "\" ] && . \"" + scriptPath + "\""

	out := StatusReport{Surfaces: make([]SurfaceStatus, 0, len(selected))}
	for _, s := range selected {
		st := SurfaceStatus{Surface: s, Installed: false, Paths: []string{}}
		switch {
		case strings.HasPrefix(s, "shell."):
			stage := filepath.Join(atraktaHome, "hooks.d", s, "20-atrakta.sh")
			st.Paths = []string{stage, scriptPath}
			stageOK, _ := exists(stage)
			scriptOK, _ := exists(scriptPath)
			rcOK := rcHasSourceLine(filepath.Join(home, ".zshrc"), sourceLine) || rcHasSourceLine(filepath.Join(home, ".bashrc"), sourceLine)
			st.Installed = stageOK && scriptOK && rcOK
		case strings.HasPrefix(s, "git."):
			hookName := "pre-commit"
			if s == "git.pre_push" {
				hookName = "pre-push"
			}
			path := filepath.Join(repoRoot, ".git", "hooks", hookName)
			st.Paths = []string{path}
			st.Installed = hasManagedBlock(path, s)
		case strings.HasPrefix(s, "ide.") || strings.HasPrefix(s, "workflow."):
			path := filepath.Join(repoRoot, ".atrakta", "hooks.d", s, "20-atrakta.sh")
			st.Paths = []string{path}
			st.Installed, _ = exists(path)
		}
		out.Surfaces = append(out.Surfaces, st)
	}
	return out, nil
}

func buildHookScript(selfExe string) string {
	qExe := strconv.Quote(selfExe)
	return "#!/bin/sh\n" +
		"_atrakta_hook_stage_disabled() {\n" +
		"  case \",${ATRAKTA_HOOK_DISABLE_STAGES:-},\" in\n" +
		"    *\",$1,\"*) return 0 ;;\n" +
		"    *) return 1 ;;\n" +
		"  esac\n" +
		"}\n" +
		"_atrakta_run_stage() {\n" +
		"  stage=\"$1\"\n" +
		"  _atrakta_hook_stage_disabled \"$stage\" && return 0\n" +
		"  dir=\"$HOME/.atrakta/hooks.d/$stage\"\n" +
		"  [ -d \"$dir\" ] || return 0\n" +
		"  for hook in \"$dir\"/*; do\n" +
		"    [ -x \"$hook\" ] || continue\n" +
		"    \"$hook\" \"$PWD\" \"$stage\" >/dev/null 2>&1\n" +
		"    code=\"$?\"\n" +
		"    [ \"$code\" -eq 0 ] && continue\n" +
		"    [ \"${ATRAKTA_HOOK_CONTINUE_ON_ERROR:-1}\" = \"1\" ] && continue\n" +
		"    return \"$code\"\n" +
		"  done\n" +
		"  return 0\n" +
		"}\n" +
		"_atrakta_chpwd_hook() {\n" +
		"  [ \"$ATRAKTA_HOOK_DISABLE\" = \"1\" ] && return\n" +
		"  [ \"$ATRAKTA_HOOK_ACTIVE\" = \"1\" ] && return\n" +
		"  ATRAKTA_HOOK_ACTIVE=1\n" +
		"  export ATRAKTA_HOOK_ACTIVE\n" +
		"  _atrakta_run_stage shell.on_cd || {\n" +
		"    unset ATRAKTA_HOOK_ACTIVE\n" +
		"    return\n" +
		"  }\n" +
		"  ATRAKTA_TRIGGER_SOURCE=hook ATRAKTA_NONINTERACTIVE=1 " + qExe + " start </dev/null >/dev/null 2>&1\n" +
		"  unset ATRAKTA_HOOK_ACTIVE\n" +
		"  return 0\n" +
		"}\n" +
		"_atrakta_preexec_hook() {\n" +
		"  [ \"$ATRAKTA_HOOK_DISABLE\" = \"1\" ] && return\n" +
		"  _atrakta_run_stage shell.on_exec || return\n" +
		"  return 0\n" +
		"}\n" +
		"case \"$SHELL\" in\n" +
		"  *zsh) autoload -Uz add-zsh-hook 2>/dev/null || true; add-zsh-hook chpwd _atrakta_chpwd_hook 2>/dev/null || true; add-zsh-hook preexec _atrakta_preexec_hook 2>/dev/null || true ;;&\n" +
		"  *bash) PROMPT_COMMAND=\"_atrakta_chpwd_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}\" ;;\n" +
		"esac\n"
}

func ensureStageScript(path, selfExe, surface string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := "#!/bin/sh\n" +
		"ATRAKTA_HOOK_SURFACE=\"" + surface + "\"\n" +
		"ATRAKTA_TRIGGER_SOURCE=hook ATRAKTA_NONINTERACTIVE=1 " + strconv.Quote(selfExe) + " start </dev/null >/dev/null 2>&1 || true\n"
	return os.WriteFile(path, []byte(content), 0o755)
}

func installGitSurface(repoRoot, hookName, surface, selfExe string) error {
	if strings.TrimSpace(repoRoot) == "" {
		return fmt.Errorf("surface %s requires repository root", surface)
	}
	hookPath := filepath.Join(repoRoot, ".git", "hooks", hookName)
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		return err
	}
	startMarker := managedPrefix + " " + surface + " START"
	endMarker := managedSuffix + " " + surface + " END"
	body := "ATRAKTA_TRIGGER_SOURCE=hook ATRAKTA_NONINTERACTIVE=1 " + strconv.Quote(selfExe) + " start </dev/null >/dev/null 2>&1 || true"
	return upsertManagedBlock(hookPath, startMarker, endMarker, body)
}

func uninstallGitSurface(repoRoot, hookName, surface string) error {
	if strings.TrimSpace(repoRoot) == "" {
		return nil
	}
	hookPath := filepath.Join(repoRoot, ".git", "hooks", hookName)
	startMarker := managedPrefix + " " + surface + " START"
	endMarker := managedSuffix + " " + surface + " END"
	return removeManagedBlock(hookPath, startMarker, endMarker)
}

func upsertManagedBlock(path, startMarker, endMarker, body string) error {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	text := string(b)
	if os.IsNotExist(err) || strings.TrimSpace(text) == "" {
		text = "#!/bin/sh\n"
	}
	block := startMarker + "\n" + body + "\n" + endMarker + "\n"
	start := strings.Index(text, startMarker)
	end := strings.Index(text, endMarker)
	if start >= 0 && end > start {
		replaceEnd := end + len(endMarker)
		if replaceEnd < len(text) && text[replaceEnd] == '\n' {
			replaceEnd++
		}
		text = text[:start] + block + text[replaceEnd:]
	} else {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n" + block
	}
	return os.WriteFile(path, []byte(text), 0o755)
}

func removeManagedBlock(path, startMarker, endMarker string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	text := string(b)
	start := strings.Index(text, startMarker)
	end := strings.Index(text, endMarker)
	if start >= 0 && end > start {
		removeEnd := end + len(endMarker)
		if removeEnd < len(text) && text[removeEnd] == '\n' {
			removeEnd++
		}
		text = text[:start] + text[removeEnd:]
	} else {
		lines := strings.Split(text, "\n")
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if trimmed == strings.TrimSpace(startMarker) || trimmed == strings.TrimSpace(endMarker) {
				continue
			}
			out = append(out, l)
		}
		text = strings.Join(out, "\n")
	}
	text = strings.TrimRight(text, "\n")
	if text != "" {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text), 0o755)
}

func hasManagedBlock(path, surface string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	startMarker := managedPrefix + " " + surface + " START"
	endMarker := managedSuffix + " " + surface + " END"
	text := string(b)
	return strings.Contains(text, startMarker) && strings.Contains(text, endMarker)
}

func normalizeSurfaces(raw []string, defaults []string) ([]string, error) {
	all := make([]string, 0)
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			t := strings.TrimSpace(strings.ToLower(part))
			if t != "" {
				all = append(all, t)
			}
		}
	}
	if len(all) == 0 {
		all = append(all, defaults...)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(all))
	for _, s := range all {
		if !contains(allSurfaces, s) {
			return nil, fmt.Errorf("unsupported surface: %s", s)
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}

func contains(rows []string, value string) bool {
	for _, row := range rows {
		if strings.TrimSpace(strings.ToLower(row)) == strings.TrimSpace(strings.ToLower(value)) {
			return true
		}
	}
	return false
}

func hasSurfacePrefix(rows []string, prefix string) bool {
	for _, s := range rows {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func hasAnyShellStage(atraktaHome string) bool {
	for _, s := range []string{"shell.on_cd", "shell.on_exec"} {
		ok, _ := exists(filepath.Join(atraktaHome, "hooks.d", s, "20-atrakta.sh"))
		if ok {
			return true
		}
	}
	return false
}

func rcHasSourceLine(rcPath, sourceLine string) bool {
	b, err := os.ReadFile(rcPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(b), sourceLine)
}

func ensureSourceLine(rcPath, scriptPath string) error {
	line := "[ -f \"" + scriptPath + "\" ] && . \"" + scriptPath + "\""
	b, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", rcPath, err)
	}
	text := string(b)
	if strings.Contains(text, line) {
		return nil
	}
	if !strings.HasSuffix(text, "\n") && text != "" {
		text += "\n"
	}
	text += line + "\n"
	if err := os.WriteFile(rcPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", rcPath, err)
	}
	return nil
}

func removeSourceLine(rcPath, scriptPath string) error {
	line := "[ -f \"" + scriptPath + "\" ] && . \"" + scriptPath + "\""
	b, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", rcPath, err)
	}
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			continue
		}
		out = append(out, l)
	}
	text := strings.TrimRight(strings.Join(out, "\n"), "\n")
	if text != "" {
		text += "\n"
	}
	if err := os.WriteFile(rcPath, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", rcPath, err)
	}
	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
