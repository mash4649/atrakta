package importing

type ImportPulse struct {
	ImportBatches        int `json:"import_batches"`
	QuarantinedCaps      int `json:"quarantined_capabilities"`
	PendingConversions   int `json:"pending_conversions"`
	PendingMemoryReviews int `json:"pending_memory_reviews"`
}

func BuildImportPulse(repoRoot string) (ImportPulse, error) {
	reports, err := ListImportReports(repoRoot)
	if err != nil {
		return ImportPulse{}, err
	}
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return ImportPulse{}, err
	}
	pulse := ImportPulse{ImportBatches: len(reports)}
	for _, e := range reg.Entries {
		if e.Denied || e.QuarantineReason != "" {
			pulse.QuarantinedCaps++
		}
		if e.Denied {
			continue
		}
		if e.Kind == KindSkill || e.Kind == KindRecipeCandidate {
			if e.ConversionStatus != ConversionReviewedApproved {
				pulse.PendingConversions++
			}
		}
		if e.CurrentMemorySurface == "" {
			e.CurrentMemorySurface = e.DefaultMemorySurface
		}
		if e.CurrentMemorySurface == string(KindReferenceMemory) {
			if e.ReviewStatus != ReviewApproved {
				pulse.PendingMemoryReviews++
			}
		}
	}
	return pulse, nil
}
