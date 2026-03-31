package projection

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAgentsMDTarget(t *testing.T) {
	root := t.TempDir()
	res, err := Render(root, "agents_md", "generic-cli")
	if err != nil {
		t.Fatalf("Render agents_md: %v", err)
	}
	if res.TargetID != "agents_md" {
		t.Fatalf("TargetID=%q", res.TargetID)
	}
	if res.TargetPath != filepath.ToSlash(filepath.Join(root, "AGENTS.md")) {
		t.Fatalf("TargetPath=%q", res.TargetPath)
	}
	if !strings.Contains(res.Rendered, "# Atrakta Projection") {
		t.Fatalf("rendered output missing title: %s", res.Rendered)
	}
}

func TestRenderIDERulesTarget(t *testing.T) {
	root := t.TempDir()
	res, err := Render(root, "ide_rules", "cursor")
	if err != nil {
		t.Fatalf("Render ide_rules: %v", err)
	}
	if res.TargetID != "ide_rules" {
		t.Fatalf("TargetID=%q", res.TargetID)
	}
	want := filepath.ToSlash(filepath.Join(root, ".cursor", "rules", "atrakta.md"))
	if res.TargetPath != want {
		t.Fatalf("TargetPath=%q want=%q", res.TargetPath, want)
	}
}

func TestRenderSkillBundleTarget(t *testing.T) {
	root := t.TempDir()
	res, err := Render(root, "skill_bundle", "claude-code")
	if err != nil {
		t.Fatalf("Render skill_bundle: %v", err)
	}
	if res.TargetID != "skill_bundle" {
		t.Fatalf("TargetID=%q", res.TargetID)
	}
	want := filepath.ToSlash(filepath.Join(root, "skills", "generated", "atrakta-skill-bundle.md"))
	if res.TargetPath != want {
		t.Fatalf("TargetPath=%q want=%q", res.TargetPath, want)
	}
}
