package importing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atrakta/internal/events"
	"atrakta/internal/util"
)

func AnalyzeCapability(repoRoot, id string) (CapabilityEntry, error) {
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return CapabilityEntry{}, err
	}
	entry, idx := findEntry(&reg, id)
	if entry == nil {
		return CapabilityEntry{}, fmt.Errorf("capability not found: %s", id)
	}
	if entry.Denied {
		entry.AnalysisStatus = AnalysisAnalyzed
		entry.Analysis = &CapabilityAnalysis{Summary: "denied at import stage", Risk: "deny", Bounded: true}
		entry.UpdatedAt = util.NowUTC()
		reg.Entries[idx] = *entry
		if err := SaveRegistry(repoRoot, reg); err != nil {
			return CapabilityEntry{}, err
		}
		_, _ = events.Append(repoRoot, events.EventCapabilityAnalyzed, "analyzer", map[string]any{
			"capability_id":   entry.ID,
			"import_batch_id": entry.ImportBatchID,
			"risk":            "deny",
			"summary":         "denied at import stage",
		})
		return *entry, nil
	}
	abs := filepath.Join(entry.SourcePath, filepath.FromSlash(entry.Path))
	b, err := os.ReadFile(abs)
	if err != nil {
		return CapabilityEntry{}, fmt.Errorf("read capability source: %w", err)
	}
	text := strings.ToLower(util.NormalizeContentLF(string(b)))
	analysis := &CapabilityAnalysis{
		FilesystemAccess: hasAny(text, []string{"os.open", "os.readfile", "os.writefile", "readfile(", "writefile(", "filepath."}),
		NetworkAccess:    hasAny(text, []string{"http.", "https://", "fetch(", "requests.", "net/http", "curl "}),
		SecretsAccess:    hasAny(text, []string{"secret", "token", "password", "apikey", "os.getenv"}),
		Bounded:          !hasAny(text, []string{"while(true)", "for(;;)", "loop {", "select {}"}),
	}
	analysis.Risk = "low"
	if analysis.SecretsAccess || analysis.NetworkAccess {
		analysis.Risk = "high"
	} else if analysis.FilesystemAccess || !analysis.Bounded {
		analysis.Risk = "medium"
	}
	analysis.Summary = fmt.Sprintf("fs=%v net=%v secrets=%v bounded=%v", analysis.FilesystemAccess, analysis.NetworkAccess, analysis.SecretsAccess, analysis.Bounded)

	entry.AnalysisStatus = AnalysisAnalyzed
	entry.Analysis = analysis
	if analysis.Risk != "low" && strings.TrimSpace(entry.QuarantineReason) == "" {
		entry.QuarantineReason = "analysis_requires_review"
	}
	entry.UpdatedAt = util.NowUTC()
	reg.Entries[idx] = *entry

	if err := SaveRegistry(repoRoot, reg); err != nil {
		return CapabilityEntry{}, err
	}
	_, _ = events.Append(repoRoot, events.EventCapabilityAnalyzed, "analyzer", map[string]any{
		"capability_id":     entry.ID,
		"import_batch_id":   entry.ImportBatchID,
		"risk":              analysis.Risk,
		"filesystem_access": analysis.FilesystemAccess,
		"network_access":    analysis.NetworkAccess,
		"secrets_access":    analysis.SecretsAccess,
		"bounded":           analysis.Bounded,
		"analysis_summary":  analysis.Summary,
	})
	return *entry, nil
}

func AnalyzeImportBatch(repoRoot, batchID string) ([]CapabilityEntry, error) {
	reg, _, err := LoadRegistry(repoRoot)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, e := range reg.Entries {
		if e.ImportBatchID == batchID {
			ids = append(ids, e.ID)
		}
	}
	out := make([]CapabilityEntry, 0, len(ids))
	for _, id := range ids {
		e, err := AnalyzeCapability(repoRoot, id)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func hasAny(text string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(text, n) {
			return true
		}
	}
	return false
}
