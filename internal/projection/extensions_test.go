package projection

import (
	"strings"
	"testing"

	"atrakta/internal/contract"
)

func TestExtensionDesiredRendersContractEntriesAndHooks(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	on := true
	off := false
	c.Extensions.MCP = []contract.ExtensionEntry{{ID: "local-mcp"}}
	c.Extensions.Plugins = []contract.ExtensionEntry{{ID: "my-plugin", Enabled: &on}, {ID: "disabled-plugin", Enabled: &off}}
	c.Extensions.Skills = []contract.ExtensionEntry{{ID: "my-skill"}}
	c.Extensions.Workflows = []contract.ExtensionEntry{{ID: "my-workflow"}}
	if c.Extensions.Hooks == nil {
		c.Extensions.Hooks = &contract.HooksExtension{}
	}
	c.Extensions.Hooks.Shell = &contract.ShellHooks{OnCD: &on}
	c.Extensions.Hooks.IDE = &contract.IDEHooks{OnOpen: &on}

	model := BuildCanonicalModel(c, "sha256:contract", "# root\n")
	rows := ExtensionDesired(model)
	want := map[string]bool{
		".extensions/mcp/local-mcp.md":         false,
		".extensions/plugins/my-plugin.md":     false,
		".extensions/skills/my-skill.md":       false,
		".extensions/workflows/my-workflow.md": false,
		".extensions/hooks/shell.on_cd.md":     false,
		".extensions/hooks/ide.on_open.md":     false,
	}
	for _, r := range rows {
		if _, ok := want[r.Path]; ok {
			want[r.Path] = true
		}
		if r.Interface != "extensions" {
			t.Fatalf("unexpected interface for extension projection: %#v", r)
		}
		if r.Target != "" {
			t.Fatalf("extension projection must be copy mode (empty target): %#v", r)
		}
	}
	for path, seen := range want {
		if !seen {
			t.Fatalf("missing extension projection path: %s", path)
		}
	}
	for _, r := range rows {
		if strings.Contains(r.Path, "disabled-plugin") {
			t.Fatalf("disabled extension should not render: %s", r.Path)
		}
	}
}

func TestParseExtensionTemplateIDAndContent(t *testing.T) {
	tid := "extensions:plugin:my-plugin@1"
	kind, id, ok := ParseExtensionTemplateID(tid)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if kind != "plugin" || id != "my-plugin" {
		t.Fatalf("unexpected parse result: kind=%s id=%s", kind, id)
	}
	content, ok := SyntheticTemplateContent(tid)
	if !ok {
		t.Fatalf("expected synthetic extension content")
	}
	if !strings.Contains(content, "kind: plugin") || !strings.Contains(content, "id: my-plugin") {
		t.Fatalf("unexpected extension synthetic content: %s", content)
	}
}
