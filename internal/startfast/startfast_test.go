package startfast

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeKeyStable(t *testing.T) {
	root := t.TempDir()
	writeFixtureLayout(t, root)

	a, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute key A: %v", err)
	}
	b, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute key B: %v", err)
	}
	if a.Key != b.Key {
		t.Fatalf("key mismatch: %q != %q", a.Key, b.Key)
	}
}

func TestComputeKeyChangesWhenAssetsChange(t *testing.T) {
	root := t.TempDir()
	writeFixtureLayout(t, root)

	before, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute key before: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	after, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute key after: %v", err)
	}
	if before.Key == after.Key {
		t.Fatal("expected key change after workspace asset change")
	}
}

func TestComputeKeyChangesWhenApplyRequestedChanges(t *testing.T) {
	root := t.TempDir()
	writeFixtureLayout(t, root)

	withoutApply, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute key without apply: %v", err)
	}
	withApply, err := ComputeKey(root, "generic-cli", true)
	if err != nil {
		t.Fatalf("compute key with apply: %v", err)
	}
	if withoutApply.Key == withApply.Key {
		t.Fatal("expected key change when apply_requested changes")
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeFixtureLayout(t, root)

	key, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute key: %v", err)
	}
	in := Snapshot{
		Key:                 key.Key,
		ContractHash:        key.ContractHash,
		CanonicalPolicyHash: key.CanonicalPolicyHash,
		WorkspaceStamp:      key.WorkspaceStamp,
		InterfaceID:         key.InterfaceID,
		ApplyRequested:      false,
	}
	if err := SaveSnapshot(root, in); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	out, err := LoadSnapshot(root)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if !IsMatch(out, key) {
		t.Fatalf("snapshot should match key")
	}
	if out.InterfaceID != "generic-cli" {
		t.Fatalf("interface id=%q", out.InterfaceID)
	}
}

func TestSnapshotLegacyCompatibility(t *testing.T) {
	root := t.TempDir()
	writeFixtureLayout(t, root)

	current, err := ComputeKey(root, "generic-cli", false)
	if err != nil {
		t.Fatalf("compute current key: %v", err)
	}
	legacy := Snapshot{
		Key:                 legacyKey(current.ContractHash, current.InterfaceID, current.CanonicalPolicyHash, current.WorkspaceStamp),
		ContractHash:        current.ContractHash,
		CanonicalPolicyHash: current.CanonicalPolicyHash,
		WorkspaceStamp:      current.WorkspaceStamp,
		InterfaceID:         current.InterfaceID,
		ApplyRequested:      false,
	}
	if !IsMatch(legacy, current) {
		t.Fatal("legacy snapshot should match current key")
	}
}

func writeFixtureLayout(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".atrakta", "canonical", "policies", "registry"), 0o755); err != nil {
		t.Fatalf("mkdir canonical path: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".atrakta"), 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".atrakta", "canonical", "policies", "registry", "index.json"), []byte("{\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatalf("write policy index: %v", err)
	}
	contract := map[string]any{
		"v": 1,
		"interfaces": map[string]any{
			"supported": []string{"generic-cli"},
			"fallback":  "generic-cli",
		},
		"boundary": map[string]any{
			"managed_root": ".atrakta/",
		},
		"tools": map[string]any{
			"allow": []string{"create", "edit", "run"},
		},
		"security": map[string]any{
			"destructive":      "deny",
			"external_send":    "deny",
			"approval":         "explicit",
			"permission_model": "proposal_only",
		},
		"routing": map[string]any{
			"default": map[string]any{"worker": "general"},
		},
	}
	raw, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".atrakta", "contract.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
}
