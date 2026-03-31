package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPluginSDKSchemaDefinesSafeTargets(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "schemas", "extension", "plugin-sdk.schema.json")

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	if got, _ := schema["title"].(string); got != "PluginSDKDefinition" {
		t.Fatalf("schema title=%q", got)
	}

	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		t.Fatal("schema properties missing")
	}

	schemaVersion, _ := props["schema_version"].(map[string]any)
	if schemaVersion == nil {
		t.Fatal("schema_version missing")
	}
	if got, _ := schemaVersion["const"].(string); got != "plugin-sdk.v1" {
		t.Fatalf("schema_version const=%q", got)
	}

	targets, _ := props["targets"].(map[string]any)
	if targets == nil {
		t.Fatal("targets missing")
	}
	items, _ := targets["items"].(map[string]any)
	if items == nil {
		t.Fatal("targets items missing")
	}
	enum, _ := items["enum"].([]any)
	if len(enum) != 3 {
		t.Fatalf("targets enum length=%d", len(enum))
	}

	allowed := map[string]struct{}{
		"adapter":    {},
		"projection": {},
		"operations": {},
	}
	for _, raw := range enum {
		v, _ := raw.(string)
		if _, ok := allowed[v]; !ok {
			t.Fatalf("unexpected target value: %q", v)
		}
		delete(allowed, v)
	}
	if len(allowed) != 0 {
		t.Fatalf("missing targets: %v", allowed)
	}
}
