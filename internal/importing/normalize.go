package importing

import (
	"fmt"
	"path/filepath"
	"strings"

	"atrakta/internal/events"
	"atrakta/internal/util"
)

func ImportRepository(repoRoot string, loaded LoadResult) (ImportReport, error) {
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return ImportReport{}, err
	}
	entries := make([]CapabilityEntry, 0, len(loaded.Files))
	importedIDs := []string{}
	deniedIDs := []string{}
	now := util.NowUTC()
	evs := make([]events.AppendInput, 0, len(loaded.Files)*3)

	for _, f := range loaded.Files {
		entry := CapabilityEntry{
			ID:               capabilityID(loaded.ImportBatchID, f.RelPath),
			Kind:             classifyCapability(f.RelPath),
			Path:             f.RelPath,
			SourceType:       loaded.SourceType,
			SourcePath:       loaded.SourcePath,
			ImportBatchID:    loaded.ImportBatchID,
			AnalysisStatus:   AnalysisPending,
			QuarantineReason: "quarantine_first_default",
			ConversionStatus: ConversionNone,
			ReviewStatus:     ReviewPending,
			Executable:       false,
			ContentHash:      f.ContentHash,
			Provenance: CapabilityProvenance{
				RootPath:    loaded.SourcePath,
				SourcePath:  f.RelPath,
				ContentHash: f.ContentHash,
				ImportedAt:  now,
			},
			UpdatedAt: now,
		}

		if isDenyTarget(f, entry.Kind) {
			entry.Denied = true
			entry.DenyReason = denyReason(f, entry.Kind)
			entry.QuarantineReason = entry.DenyReason
			entry.Kind = KindUnsupported
			deniedIDs = append(deniedIDs, entry.ID)
		} else {
			importedIDs = append(importedIDs, entry.ID)
		}

		if isMemoryLike(entry.Kind, entry.Path) {
			entry.DefaultMemorySurface = string(KindReferenceMemory)
			entry.CurrentMemorySurface = string(KindReferenceMemory)
			evs = append(evs, events.AppendInput{
				Type:  events.EventMemorySurfaceAssigned,
				Actor: "importer",
				Payload: map[string]any{
					"capability_id":   entry.ID,
					"memory_surface":  entry.CurrentMemorySurface,
					"import_batch_id": entry.ImportBatchID,
				},
			})
		}

		entries = append(entries, entry)
		evs = append(evs, events.AppendInput{
			Type:  events.EventCapabilityImported,
			Actor: "importer",
			Payload: map[string]any{
				"capability_id":   entry.ID,
				"kind":            entry.Kind,
				"path":            entry.Path,
				"source_type":     entry.SourceType,
				"source_path":     entry.SourcePath,
				"import_batch_id": entry.ImportBatchID,
				"content_hash":    entry.ContentHash,
				"denied":          entry.Denied,
			},
		})
		evs = append(evs, events.AppendInput{
			Type:  events.EventCapabilityQuarantined,
			Actor: "importer",
			Payload: map[string]any{
				"capability_id":     entry.ID,
				"kind":              entry.Kind,
				"path":              entry.Path,
				"import_batch_id":   entry.ImportBatchID,
				"quarantine_reason": entry.QuarantineReason,
			},
		})
	}

	upsertEntries(&reg, entries)
	if err := SaveRegistry(repoRoot, reg); err != nil {
		return ImportReport{}, err
	}
	rep := ImportReport{
		V:                    1,
		ImportBatchID:        loaded.ImportBatchID,
		SourceType:           loaded.SourceType,
		SourcePath:           loaded.SourcePath,
		ImportedAt:           now,
		ImportedCapabilities: importedIDs,
		DeniedCapabilities:   deniedIDs,
		QuarantinedCount:     len(entries),
		PendingConversions:   countPendingConversions(entries),
		PendingMemoryReviews: countPendingMemoryReviews(entries),
	}
	if err := SaveImportReport(repoRoot, rep); err != nil {
		return ImportReport{}, err
	}
	if _, err := events.AppendBatch(repoRoot, evs); err != nil {
		return ImportReport{}, err
	}
	return rep, nil
}

func capabilityID(batchID, relPath string) string {
	seed := batchID + "|" + util.NormalizeRelPath(relPath)
	h := util.SHA256Hex([]byte(seed))
	if len(h) > 16 {
		h = h[:16]
	}
	return "cap_" + h
}

func classifyCapability(relPath string) CapabilityKind {
	l := strings.ToLower(util.NormalizeRelPath(relPath))
	base := strings.ToLower(filepath.Base(l))
	switch {
	case strings.Contains(l, "/skills/") || strings.HasPrefix(base, "skill") || strings.HasSuffix(base, ".skill.md"):
		return KindSkill
	case strings.Contains(l, "/recipes/") || strings.Contains(base, "recipe"):
		return KindRecipeCandidate
	case isMemoryLike(KindUnsupported, l):
		return KindReferenceMemory
	case strings.Contains(l, "gateway"):
		return KindGateway
	case strings.Contains(l, "openapi") || strings.Contains(l, "swagger") || strings.Contains(l, "/api/"):
		return KindAPI
	default:
		return KindUnsupported
	}
}

func isMemoryLike(kind CapabilityKind, relPath string) bool {
	if kind == KindReferenceMemory {
		return true
	}
	l := strings.ToLower(relPath)
	return strings.Contains(l, "/memory/") || strings.Contains(filepath.Base(l), "memory")
}

func isDenyTarget(f LoadedFile, kind CapabilityKind) bool {
	if f.SecretLike || f.Binary || f.Executable {
		return true
	}
	return kind == KindUnsupported
}

func denyReason(f LoadedFile, kind CapabilityKind) string {
	switch {
	case f.SecretLike:
		return "deny_secret_like_file"
	case f.Binary:
		return "deny_binary_blob"
	case f.Executable:
		return "deny_unsupported_executable_blob"
	case kind == KindUnsupported:
		return "deny_unsupported_kind"
	default:
		return "deny_unknown"
	}
}

func countPendingConversions(entries []CapabilityEntry) int {
	n := 0
	for _, e := range entries {
		if e.Denied {
			continue
		}
		if e.Kind == KindSkill || e.Kind == KindRecipeCandidate {
			n++
		}
	}
	return n
}

func countPendingMemoryReviews(entries []CapabilityEntry) int {
	n := 0
	for _, e := range entries {
		if e.Denied {
			continue
		}
		if e.CurrentMemorySurface == string(KindReferenceMemory) {
			n++
		}
	}
	return n
}

func latestImportReport(repoRoot string) (ImportReport, error) {
	reports, err := ListImportReports(repoRoot)
	if err != nil {
		return ImportReport{}, err
	}
	if len(reports) == 0 {
		return ImportReport{}, fmt.Errorf("no import reports found")
	}
	return reports[0], nil
}
