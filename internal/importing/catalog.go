package importing

import (
	"sort"
)

type CatalogOptions struct {
	ReviewedOnly bool
	Limit        int
}

func BuildCatalog(repoRoot string, opt CatalogOptions) (Catalog, error) {
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return Catalog{}, err
	}
	items := []CatalogItem{}
	for _, e := range reg.Entries {
		if e.Denied {
			continue
		}
		if e.Kind != KindReferenceMemory && e.Kind != KindSkill && e.Kind != KindRecipeCandidate {
			continue
		}
		if opt.ReviewedOnly && e.ReviewStatus != ReviewApproved {
			continue
		}
		items = append(items, CatalogItem{
			CapabilityID: e.ID,
			Kind:         string(e.Kind),
			SourcePath:   e.Path,
			ReviewStatus: e.ReviewStatus,
			Attribution:  "registry:" + e.ID,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].SourcePath != items[j].SourcePath {
			return items[i].SourcePath < items[j].SourcePath
		}
		return items[i].CapabilityID < items[j].CapabilityID
	})
	if opt.Limit > 0 && len(items) > opt.Limit {
		items = items[:opt.Limit]
	}
	return Catalog{Items: items}, nil
}
