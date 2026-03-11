package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atrakta/internal/contract"
)

func TestRunIntegrationDetectsOverwriteRisk(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, repo, "AGENTS.md", "# Root\n")
	mustWriteFile(t, repo, "CLAUDE.md", "# user-managed\n")
	c := contract.Default(repo)
	c.Interfaces.CoreSet = []string{"claude_code"}
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}

	rep, err := RunIntegration(repo)
	if err != nil {
		t.Fatalf("run integration failed: %v", err)
	}
	if rep.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s", rep.Outcome)
	}
	if !hasIntegrationFinding(rep.BlockingIssues, "overwrite_risk") {
		t.Fatalf("expected overwrite_risk, got %#v", rep.BlockingIssues)
	}
}

func TestRunIntegrationDetectsIncludeMissing(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, repo, "AGENTS.md", "# Root\n")
	c := contract.Default(repo)
	c.Extensions.Agents.Mode = "include"
	c.Extensions.Agents.AppendFile = ".atrakta/AGENTS.append.md"
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}

	rep, err := RunIntegration(repo)
	if err != nil {
		t.Fatalf("run integration failed: %v", err)
	}
	if !hasIntegrationFinding(rep.Warnings, "include_missing") {
		t.Fatalf("expected include_missing warning, got %#v", rep.Warnings)
	}
}

func TestRunIntegrationDetectsUnsupportedExtensionProjection(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, repo, "AGENTS.md", "# Root\n")
	c := contract.Default(repo)
	c.Extensions.Plugins = []contract.ExtensionEntry{{ID: "demo-plugin"}}
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}

	rep, err := RunIntegration(repo)
	if err != nil {
		t.Fatalf("run integration failed: %v", err)
	}
	if rep.Outcome != "WARN" {
		t.Fatalf("expected WARN, got %s", rep.Outcome)
	}
	if !hasIntegrationFinding(rep.Warnings, "unsupported_extension_projection") {
		t.Fatalf("expected unsupported_extension_projection warning, got %#v", rep.Warnings)
	}
}

func TestRunIntegrationDetectsAppendFailure(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, repo, "AGENTS.md", "# Root\n\n<!-- ATRAKTA_MANAGED:START -->\n")
	c := contract.Default(repo)
	c.Extensions.Agents.Mode = "append"
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}

	rep, err := RunIntegration(repo)
	if err != nil {
		t.Fatalf("run integration failed: %v", err)
	}
	if rep.Outcome != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s", rep.Outcome)
	}
	if !hasIntegrationFinding(rep.BlockingIssues, "append_failure") {
		t.Fatalf("expected append_failure, got %#v", rep.BlockingIssues)
	}
}

func TestIntegrationJSONIncludesFindings(t *testing.T) {
	repo := t.TempDir()
	mustWriteFile(t, repo, "AGENTS.md", "# Root\n")
	c := contract.Default(repo)
	c.Extensions.Plugins = []contract.ExtensionEntry{{ID: "demo-plugin"}}
	if _, err := contract.Save(repo, c); err != nil {
		t.Fatalf("save contract failed: %v", err)
	}
	rep, err := RunIntegration(repo)
	if err != nil {
		t.Fatalf("run integration failed: %v", err)
	}
	js := rep.JSON()
	if !strings.Contains(js, "unsupported_extension_projection") {
		t.Fatalf("json output missing finding: %s", js)
	}
}

func hasIntegrationFinding(list []IntegrationFinding, code string) bool {
	for _, f := range list {
		if f.Code == code {
			return true
		}
	}
	return false
}

func mustWriteFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
