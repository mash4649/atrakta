package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPServerSchemaDefinesSafeCapabilities(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "schemas", "extension", "mcp-server.schema.json")

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatalf("parse schema: %v", err)
	}

	if got, _ := schema["title"].(string); got != "MCPServerDefinition" {
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
	if got, _ := schemaVersion["const"].(string); got != "mcp-server.v1" {
		t.Fatalf("schema_version const=%q", got)
	}

	transport, _ := props["transport"].(map[string]any)
	if transport == nil {
		t.Fatal("transport missing")
	}
	transportProps, _ := transport["properties"].(map[string]any)
	if transportProps == nil {
		t.Fatal("transport properties missing")
	}
	typeField, _ := transportProps["type"].(map[string]any)
	if typeField == nil {
		t.Fatal("transport.type missing")
	}
	enum, _ := typeField["enum"].([]any)
	if len(enum) != 2 {
		t.Fatalf("transport.type enum length=%d", len(enum))
	}
}
