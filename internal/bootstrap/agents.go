package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
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

func EnsureRootAGENTS(repoRoot string) (content string, created bool, err error) {
	p := filepath.Join(repoRoot, "AGENTS.md")
	b, err := os.ReadFile(p)
	if err == nil {
		return string(b), false, nil
	}
	if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("read AGENTS.md: %w", err)
	}
	if err := os.WriteFile(p, []byte(defaultAGENTS), 0o644); err != nil {
		return "", false, fmt.Errorf("write AGENTS.md: %w", err)
	}
	return defaultAGENTS, true, nil
}
