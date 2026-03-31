package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMachineContract(t *testing.T) {
	root := t.TempDir()
	contractPath := filepath.Join(root, ".atrakta")
	if err := os.MkdirAll(contractPath, 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	payload := map[string]any{
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
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractPath, "contract.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	out, err := LoadMachineContract(root)
	if err != nil {
		t.Fatalf("load machine contract: %v", err)
	}
	if out["v"] != float64(1) {
		t.Fatalf("version=%v", out["v"])
	}
}

func TestLoadMachineContractMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadMachineContract(root); err == nil {
		t.Fatal("expected missing contract error")
	}
}

func TestLoadMachineContractInvalid(t *testing.T) {
	root := t.TempDir()
	contractPath := filepath.Join(root, ".atrakta")
	if err := os.MkdirAll(contractPath, 0o755); err != nil {
		t.Fatalf("mkdir .atrakta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractPath, "contract.json"), []byte(`{"v":0}`), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
	if _, err := LoadMachineContract(root); err == nil {
		t.Fatal("expected validation error")
	}
}
