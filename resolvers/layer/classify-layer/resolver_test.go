package classifylayer

import "testing"

func TestClassifyLayerByKind(t *testing.T) {
	tests := []struct {
		name   string
		item   Item
		want   string
		next   string
		reason string
	}{
		{name: "core request", item: Item{Kind: "request"}, want: "core", next: "inspect", reason: "layer classified"},
		{name: "canonical policy", item: Item{Kind: "policy"}, want: "canonical", next: "inspect", reason: "layer classified"},
		{name: "extension workflow", item: Item{Kind: "workflow"}, want: "extension", next: "inspect", reason: "layer classified"},
		{name: "unknown item", item: Item{Kind: "mystery"}, want: "unknown", next: "deny", reason: "layer cannot be classified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLayer(tt.item)
			if got.Decision != tt.want {
				t.Fatalf("decision = %v, want %v", got.Decision, tt.want)
			}
			if got.NextAllowedAction != tt.next {
				t.Fatalf("next = %q, want %q", got.NextAllowedAction, tt.next)
			}
			if got.Reason != tt.reason {
				t.Fatalf("reason = %q, want %q", got.Reason, tt.reason)
			}
		})
	}
}

func TestClassifyLayerBySchemaID(t *testing.T) {
	item := Item{SchemaID: "atrakta/schemas/core/request.schema.json"}
	got := ClassifyLayer(item)
	if got.Decision != "core" {
		t.Fatalf("decision = %v, want core", got.Decision)
	}
}
