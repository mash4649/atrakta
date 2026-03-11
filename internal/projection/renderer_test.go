package projection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"atrakta/internal/contract"
	"atrakta/internal/registry"
	"atrakta/internal/util"
)

func TestBuildCanonicalModelNormalizesAndCarriesParityExtensions(t *testing.T) {
	c := contract.Default(t.TempDir())
	c.Parity = nil
	c.Extensions = nil

	m := BuildCanonicalModel(c, "sha256:contract", "line1\r\nline2\r\n")
	if m.Contract.Parity == nil {
		t.Fatalf("expected parity defaults in canonical model")
	}
	if m.Contract.Extensions == nil {
		t.Fatalf("expected extensions defaults in canonical model")
	}
	if got := m.SourceText["AGENTS.md"]; got != "line1\nline2\n" {
		t.Fatalf("expected normalized AGENTS source, got %q", got)
	}
	wantHash := util.SHA256Tagged([]byte("line1\nline2\n"))
	if got := m.SourceHash["AGENTS.md"]; got != wantHash {
		t.Fatalf("unexpected AGENTS hash: got=%s want=%s", got, wantHash)
	}
}

func TestRenderTargetsDeterministicForSameInput(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".atrakta"), 0o755); err != nil {
		t.Fatal(err)
	}
	c := contract.Default(repo)
	c.Projections.OptionalTemplates = map[string][]string{
		"cursor": {"contract-json"},
	}
	cb, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(filepath.Join(repo, ".atrakta", "contract.json"), cb, 0o644); err != nil {
		t.Fatal(err)
	}
	hash := contract.ContractHash(cb)
	reg := registry.Default()

	a, err := RequiredForTargets(repo, c, reg, []string{"windsurf", "cursor", "cursor"}, hash, "# root\n")
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}
	b, err := RequiredForTargets(repo, c, reg, []string{"cursor", "windsurf"}, hash, "# root\n")
	if err != nil {
		t.Fatalf("second render failed: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("expected deterministic projections\nA=%#v\nB=%#v", a, b)
	}
	for i := 1; i < len(a); i++ {
		if a[i-1].Path > a[i].Path {
			t.Fatalf("projections not sorted by path: %s > %s", a[i-1].Path, a[i].Path)
		}
	}
	ha, err := StableRenderHash(a)
	if err != nil {
		t.Fatalf("hash A failed: %v", err)
	}
	hb, err := StableRenderHash(b)
	if err != nil {
		t.Fatalf("hash B failed: %v", err)
	}
	if ha != hb {
		t.Fatalf("expected stable render hash for equal projections: %s != %s", ha, hb)
	}
}

func TestStableRenderHashIgnoresInputOrder(t *testing.T) {
	rowsA := []Desired{
		{Interface: "cursor", TemplateID: "cursor:agents-md@1", Path: ".cursor/AGENTS.md", Source: "AGENTS.md", Target: "AGENTS.md", Fingerprint: "sha256:a"},
		{Interface: "claude_code", TemplateID: "claude_code:agents-md@1", Path: "CLAUDE.md", Source: "AGENTS.md", Target: "AGENTS.md", Fingerprint: "sha256:b"},
	}
	rowsB := []Desired{rowsA[1], rowsA[0]}

	ha, err := StableRenderHash(rowsA)
	if err != nil {
		t.Fatalf("hash A failed: %v", err)
	}
	hb, err := StableRenderHash(rowsB)
	if err != nil {
		t.Fatalf("hash B failed: %v", err)
	}
	if ha != hb {
		t.Fatalf("expected order-independent hash: %s != %s", ha, hb)
	}
}

func TestClaudeRendererProducesNativeTargets(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	model := BuildCanonicalModel(c, "sha256:contract", "# root\n")
	reg := registry.Default()
	rows, err := DefaultEngine().RenderTargets(repo, model, reg, []string{"claude_code"})
	if err != nil {
		t.Fatalf("render claude targets failed: %v", err)
	}
	want := map[string]bool{
		"CLAUDE.md":                 false,
		".claude/settings.json":     false,
		".claude/mcp.json":          false,
		".claude/agents/atrakta.md": false,
	}
	for _, r := range rows {
		if _, ok := want[r.Path]; ok {
			want[r.Path] = true
		}
	}
	for p, ok := range want {
		if !ok {
			t.Fatalf("missing claude projection path: %s (rows=%#v)", p, rows)
		}
	}
}

func TestCodexRendererProducesNativeTargets(t *testing.T) {
	repo := t.TempDir()
	c := contract.Default(repo)
	model := BuildCanonicalModel(c, "sha256:contract", "# root\n")
	reg := registry.Default()
	rows, err := DefaultEngine().RenderTargets(repo, model, reg, []string{"codex_cli"})
	if err != nil {
		t.Fatalf("render codex targets failed: %v", err)
	}
	want := map[string]bool{
		"AGENTS.md":          false,
		".codex/config.toml": false,
	}
	for _, r := range rows {
		if _, ok := want[r.Path]; ok {
			want[r.Path] = true
		}
	}
	for p, ok := range want {
		if !ok {
			t.Fatalf("missing codex projection path: %s (rows=%#v)", p, rows)
		}
	}
}
