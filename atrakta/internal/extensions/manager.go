package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	resolveextensionorder "github.com/mash4649/atrakta/v0/resolvers/extension/resolve-extension-order"
)

// ManifestItem describes one extension entry.
type ManifestItem struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Source  string `json:"source,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Manifest describes one manifest file.
type Manifest struct {
	Name  string         `json:"name"`
	Items []ManifestItem `json:"items"`
}

// ResolveResult contains loaded manifests and deterministic extension order.
type ResolveResult struct {
	ProjectRoot string                       `json:"project_root"`
	Manifests   []string                     `json:"manifests"`
	Items       []resolveextensionorder.Item `json:"items"`
	OrderedIDs  []string                     `json:"ordered_ids"`
}

// Resolve loads extension manifests and resolves deterministic order.
func Resolve(projectRoot string) (ResolveResult, error) {
	root := projectRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ResolveResult{}, err
		}
		root = wd
	}

	manifestDir := filepath.Join(root, "extensions", "manifests")
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ResolveResult{
				ProjectRoot: root,
				Manifests:   []string{},
				Items:       []resolveextensionorder.Item{},
				OrderedIDs:  []string{},
			}, nil
		}
		return ResolveResult{}, err
	}

	items := make([]resolveextensionorder.Item, 0, 16)
	manifests := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(manifestDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return ResolveResult{}, err
		}
		var m Manifest
		if err := json.Unmarshal(b, &m); err != nil {
			return ResolveResult{}, fmt.Errorf("parse extension manifest %s: %w", e.Name(), err)
		}
		manifests = append(manifests, filepath.ToSlash(filepath.Join("extensions/manifests", e.Name())))
		for _, item := range m.Items {
			if !item.Enabled {
				continue
			}
			items = append(items, resolveextensionorder.Item{
				ID:   item.ID,
				Kind: item.Kind,
			})
		}
	}

	sort.Strings(manifests)
	orderOut := resolveextensionorder.ResolveExtensionOrder(items)
	decision := orderOut.Decision.(resolveextensionorder.ExtensionDecision)
	orderedIDs := make([]string, 0, len(decision.Ordered))
	for _, item := range decision.Ordered {
		orderedIDs = append(orderedIDs, item.ID)
	}

	return ResolveResult{
		ProjectRoot: root,
		Manifests:   manifests,
		Items:       items,
		OrderedIDs:  orderedIDs,
	}, nil
}
