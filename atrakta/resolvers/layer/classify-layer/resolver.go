package classifylayer

import (
	"sort"
	"strings"

	"github.com/mash4649/atrakta/v0/resolvers/common"
)

// Item identifies a unit to classify into owner layers.
type Item struct {
	Kind     string `json:"kind"`
	SchemaID string `json:"schema_id"`
}

var layerByKind = map[string]string{
	"request":         "core",
	"decision":        "core",
	"result":          "core",
	"error":           "core",
	"capability":      "canonical",
	"policy":          "canonical",
	"task_state":      "canonical",
	"task-state":      "canonical",
	"audit_event":     "canonical",
	"audit-event":     "canonical",
	"runtime_profile": "extension",
	"runtime-profile": "extension",
	"repo_map":        "extension",
	"repo-map":        "extension",
	"skill":           "extension",
	"workflow":        "extension",
	"provenance":      "extension",
}

// ClassifyLayer resolves owner layer by kind and schema id hints.
func ClassifyLayer(item Item) common.ResolverOutput {
	layer := "unknown"
	kind := normalize(item.Kind)
	if known, ok := layerByKind[kind]; ok {
		layer = known
	} else {
		layer = classifyFromSchema(item.SchemaID)
	}

	evidence := []string{}
	if item.Kind != "" {
		evidence = append(evidence, "kind="+kind)
	}
	if item.SchemaID != "" {
		evidence = append(evidence, "schema_id="+strings.ToLower(item.SchemaID))
	}
	sort.Strings(evidence)

	switch layer {
	case "core", "canonical", "extension":
		return common.NewOutput(item, layer, "layer classified", evidence, "inspect")
	default:
		return common.NewOutput(item, layer, "layer cannot be classified", evidence, "deny")
	}
}

func classifyFromSchema(schemaID string) string {
	n := normalize(schemaID)
	switch {
	case strings.Contains(n, "/schemas/core/"):
		return "core"
	case strings.Contains(n, "/schemas/canonical/"):
		return "canonical"
	case strings.Contains(n, "/schemas/extension/"):
		return "extension"
	default:
		return "unknown"
	}
}

func normalize(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, " ", "_")
	return v
}
