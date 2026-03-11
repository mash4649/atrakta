package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultAGENTS = "" +
	"# Atrakta\n\n" +
	"This document defines human-readable principles.\n" +
	"Machine-executable contract is located at `.atrakta/contract.json`.\n\n" +
	"## Principles\n" +
	"- Detect -> Plan -> Apply discipline\n" +
	"- Managed-only destructive mutation\n" +
	"- Never prune under ambiguity\n" +
	"- Append-only event observability\n" +
	"- Deterministic replay and explicit approval\n"

const (
	managedBlockStart = "<!-- ATRAKTA_MANAGED:START -->"
	managedBlockEnd   = "<!-- ATRAKTA_MANAGED:END -->"
	defaultAppendFile = ".atrakta/AGENTS.append.md"
)

func managedBlockContent() string {
	return "" +
		"<!-- ATRAKTA_MANAGED:START -->\n" +
		"## Atrakta Managed\n" +
		"- Canonical contract: `.atrakta/contract.json`\n" +
		"- Projection status: `atrakta projection status --json`\n" +
		"- Parity doctor: `atrakta doctor --parity`\n" +
		"<!-- ATRAKTA_MANAGED:END -->\n"
}

func includeAppendContent() string {
	return "" +
		"# Atrakta Managed Appendix\n\n" +
		"This file is managed by Atrakta and intended for AGENTS include mode.\n\n" +
		"- Canonical contract: `.atrakta/contract.json`\n" +
		"- Projection status: `atrakta projection status --json`\n" +
		"- Parity doctor: `atrakta doctor --parity`\n"
}

func EnsureRootAGENTS(repoRoot string) (content string, created bool, err error) {
	return EnsureRootAGENTSWithMode(repoRoot, "append", defaultAppendFile)
}

func EnsureRootAGENTSWithMode(repoRoot, mode, appendFile string) (content string, created bool, err error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "append"
	}
	if appendFile == "" {
		appendFile = defaultAppendFile
	}
	p := filepath.Join(repoRoot, "AGENTS.md")
	b, err := os.ReadFile(p)
	if err == nil {
		text := string(b)
		switch mode {
		case "include":
			if err := ensureAppendFile(repoRoot, appendFile); err != nil {
				return "", false, err
			}
			return text, false, nil
		case "append":
			next, changed := ensureManagedBlock(text)
			if changed {
				if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
					return "", false, fmt.Errorf("write AGENTS.md: %w", err)
				}
			}
			return next, false, nil
		case "generate":
			return text, false, nil
		default:
			return "", false, fmt.Errorf("unsupported agents mode %q", mode)
		}
	}
	if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("read AGENTS.md: %w", err)
	}
	if err := os.WriteFile(p, []byte(defaultAGENTS), 0o644); err != nil {
		return "", false, fmt.Errorf("write AGENTS.md: %w", err)
	}
	if mode == "include" {
		if err := ensureAppendFile(repoRoot, appendFile); err != nil {
			return "", true, err
		}
	}
	return defaultAGENTS, true, nil
}

func ensureManagedBlock(content string) (string, bool) {
	block := managedBlockContent()
	if strings.Count(content, managedBlockStart) == 1 &&
		strings.Count(content, managedBlockEnd) == 1 &&
		strings.Contains(content, block) {
		return content, false
	}
	start := strings.Index(content, managedBlockStart)
	end := strings.Index(content, managedBlockEnd)
	if start >= 0 && end > start {
		end = end + len(managedBlockEnd)
		next := content[:start] + block + content[end:]
		return next, next != content
	}
	if start >= 0 || end >= 0 {
		next := strings.TrimRight(content, "\n") + "\n\n" + block
		return next, next != content
	}
	next := strings.TrimRight(content, "\n") + "\n\n" + block
	return next, next != content
}

func ensureAppendFile(repoRoot, appendFile string) error {
	rel := strings.TrimSpace(appendFile)
	if rel == "" {
		rel = defaultAppendFile
	}
	p := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("mkdir append file dir: %w", err)
	}
	content := includeAppendContent()
	if b, err := os.ReadFile(p); err == nil {
		if string(b) == content {
			return nil
		}
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write append file: %w", err)
	}
	return nil
}
