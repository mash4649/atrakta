package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Install(selfExe string) error {
	h, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(h, ".atrakta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	scriptPath := filepath.Join(dir, "hook.sh")
	script := buildHookScript(selfExe)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write hook script: %w", err)
	}
	for _, rc := range []string{filepath.Join(h, ".zshrc"), filepath.Join(h, ".bashrc")} {
		if err := ensureSourceLine(rc, scriptPath); err != nil {
			return err
		}
	}
	fmt.Printf("hooks installed via %s\n", scriptPath)
	return nil
}

func Uninstall() error {
	h, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(h, ".atrakta", "hook.sh")
	for _, rc := range []string{filepath.Join(h, ".zshrc"), filepath.Join(h, ".bashrc")} {
		if err := removeSourceLine(rc, scriptPath); err != nil {
			return err
		}
	}
	_ = os.Remove(scriptPath)
	fmt.Println("hooks uninstalled")
	return nil
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
		"  _atrakta_run_stage pre_start || return\n" +
		"  ATRAKTA_TRIGGER_SOURCE=hook ATRAKTA_NONINTERACTIVE=1 " + qExe + " start </dev/null >/dev/null 2>&1\n" +
		"  code=\"$?\"\n" +
		"  if [ \"$code\" -eq 0 ]; then\n" +
		"    _atrakta_run_stage post_start || true\n" +
		"  else\n" +
		"    _atrakta_run_stage on_error || true\n" +
		"  fi\n" +
		"  return 0\n" +
		"}\n" +
		"case \"$SHELL\" in\n" +
		"  *zsh) autoload -Uz add-zsh-hook 2>/dev/null || true; add-zsh-hook chpwd _atrakta_chpwd_hook 2>/dev/null || true ;;\n" +
		"  *bash) PROMPT_COMMAND=\"_atrakta_chpwd_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}\" ;;\n" +
		"esac\n"
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
