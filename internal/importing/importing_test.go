package importing

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"atrakta/internal/events"
)

func TestImportDeterminismRegression(t *testing.T) {
	repoA := buildFixtureImportSource(t, true)
	repoB := buildFixtureImportSource(t, false)

	lA, err := LoadRepository(repoA)
	if err != nil {
		t.Fatalf("load repo A failed: %v", err)
	}
	lB, err := LoadRepository(repoB)
	if err != nil {
		t.Fatalf("load repo B failed: %v", err)
	}
	if lA.ImportBatchID != lB.ImportBatchID {
		t.Fatalf("import batch id differs for equivalent inputs: %s != %s", lA.ImportBatchID, lB.ImportBatchID)
	}
	if len(lA.Files) != len(lB.Files) {
		t.Fatalf("file count differs: %d != %d", len(lA.Files), len(lB.Files))
	}
	for i := range lA.Files {
		if lA.Files[i].RelPath != lB.Files[i].RelPath {
			t.Fatalf("file order differs at %d: %s != %s", i, lA.Files[i].RelPath, lB.Files[i].RelPath)
		}
		if lA.Files[i].ContentHash != lB.Files[i].ContentHash {
			t.Fatalf("content hash differs for %s", lA.Files[i].RelPath)
		}
	}
}

func TestQuarantineFirstRegression(t *testing.T) {
	workspace := t.TempDir()
	loaded, err := LoadRepository(filepath.Join(repoRootFromCaller(t), "testdata", "import", "fixture_repo"))
	if err != nil {
		t.Fatalf("load repository failed: %v", err)
	}
	report, err := ImportRepository(workspace, loaded)
	if err != nil {
		t.Fatalf("import repository failed: %v", err)
	}
	if len(report.ImportedCapabilities) == 0 {
		t.Fatalf("expected imported capabilities")
	}
	reg, _, err := LoadRegistry(workspace)
	if err != nil {
		t.Fatalf("load registry failed: %v", err)
	}
	for _, e := range reg.Entries {
		if e.Executable {
			t.Fatalf("imported capability became executable: %s", e.ID)
		}
		if e.QuarantineReason == "" {
			t.Fatalf("quarantine reason missing: %s", e.ID)
		}
	}
	evs, err := events.ReadAll(workspace)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	if !hasEventType(evs, events.EventCapabilityQuarantined) {
		t.Fatalf("expected %s event", events.EventCapabilityQuarantined)
	}
}

func TestRecipeBoundaryRegression(t *testing.T) {
	workspace := t.TempDir()
	loaded, err := LoadRepository(filepath.Join(repoRootFromCaller(t), "testdata", "import", "fixture_repo"))
	if err != nil {
		t.Fatalf("load repository failed: %v", err)
	}
	if _, err := ImportRepository(workspace, loaded); err != nil {
		t.Fatalf("import repository failed: %v", err)
	}
	reg, _, err := LoadRegistry(workspace)
	if err != nil {
		t.Fatalf("load registry failed: %v", err)
	}
	skillID := ""
	for _, e := range reg.Entries {
		if e.Kind == KindSkill {
			skillID = e.ID
			if e.Executable {
				t.Fatalf("skill must not be executable before conversion")
			}
			break
		}
	}
	if skillID == "" {
		t.Fatalf("skill fixture not found")
	}
	entry, err := ConvertRecipeCandidate(workspace, skillID, ReviewApproved, "inputs are deterministic snapshot", "", []string{"safe.echo"})
	if err != nil {
		t.Fatalf("convert recipe candidate failed: %v", err)
	}
	if entry.Kind != KindRecipeCandidate {
		t.Fatalf("expected recipe_candidate kind")
	}
	if RecipeAllows(entry, "os.exec") {
		t.Fatalf("allowlist boundary broken: os.exec must remain denied")
	}
	evs, err := events.ReadAll(workspace)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	if !hasEventType(evs, events.EventRecipeConversionReviewed) {
		t.Fatalf("expected recipe conversion review event")
	}
	if !hasEventType(evs, events.EventRecipeCandidateCreated) {
		t.Fatalf("expected recipe candidate created event")
	}
}

func TestMemoryBoundaryRegression(t *testing.T) {
	workspace := t.TempDir()
	loaded, err := LoadRepository(filepath.Join(repoRootFromCaller(t), "testdata", "import", "fixture_repo"))
	if err != nil {
		t.Fatalf("load repository failed: %v", err)
	}
	if _, err := ImportRepository(workspace, loaded); err != nil {
		t.Fatalf("import repository failed: %v", err)
	}
	reg, _, err := LoadRegistry(workspace)
	if err != nil {
		t.Fatalf("load registry failed: %v", err)
	}
	memoryID := ""
	for _, e := range reg.Entries {
		if e.Kind == KindReferenceMemory {
			memoryID = e.ID
			if e.CurrentMemorySurface != string(KindReferenceMemory) {
				t.Fatalf("memory must remain in reference_memory after import")
			}
			break
		}
	}
	if memoryID == "" {
		t.Fatalf("memory fixture not found")
	}
	res, err := ReviewMemoryPromotion(workspace, memoryID, ReviewPending, "", true)
	if err != nil {
		t.Fatalf("memory review failed: %v", err)
	}
	if res.Promoted {
		t.Fatalf("memory promotion without review approval must fail")
	}
	res2, err := ReviewMemoryPromotion(workspace, memoryID, ReviewApproved, "operator-1", true)
	if err != nil {
		t.Fatalf("memory review approval failed: %v", err)
	}
	if !res2.Promoted {
		t.Fatalf("expected approved promotion")
	}
	evs, err := events.ReadAll(workspace)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	if !hasEventType(evs, events.EventMemoryPromotionReviewed) {
		t.Fatalf("expected memory promotion review event")
	}
}

func TestKernelInvariantRegression(t *testing.T) {
	d1 := CompareBeforePromote(10, 11)
	d2 := CompareBeforePromote(10, 11)
	if d1 != d2 {
		t.Fatalf("same state must produce same decision: %#v != %#v", d1, d2)
	}
	tie := CompareBeforePromote(10, 10)
	if tie.Promote {
		t.Fatalf("tie must not promote")
	}
	if tie.Reason != "tie_no_promote" {
		t.Fatalf("unexpected tie reason: %s", tie.Reason)
	}
}

func TestAnalyzeOnlyHookKeepsManualGates(t *testing.T) {
	workspace := t.TempDir()
	loaded, err := LoadRepository(filepath.Join(repoRootFromCaller(t), "testdata", "import", "fixture_repo"))
	if err != nil {
		t.Fatalf("load repository failed: %v", err)
	}
	rep, err := ImportRepository(workspace, loaded)
	if err != nil {
		t.Fatalf("import repository failed: %v", err)
	}
	if _, err := AnalyzeImportBatch(workspace, rep.ImportBatchID); err != nil {
		t.Fatalf("analyze import batch failed: %v", err)
	}
	reg, _, err := LoadRegistry(workspace)
	if err != nil {
		t.Fatalf("load registry failed: %v", err)
	}
	for _, e := range reg.Entries {
		if e.Executable {
			t.Fatalf("analyze-only hook must not elevate trust/executable state: %s", e.ID)
		}
	}
	evs, err := events.ReadAll(workspace)
	if err != nil {
		t.Fatalf("read events failed: %v", err)
	}
	if hasEventType(evs, events.EventRecipeConversionReviewed) {
		t.Fatalf("analyze-only hook must not auto-convert recipes")
	}
	if hasEventType(evs, events.EventCapabilityPromoted) {
		t.Fatalf("analyze-only hook must not auto-promote capabilities")
	}
}

func TestCatalogReviewedOnlyOptIn(t *testing.T) {
	workspace := t.TempDir()
	loaded, err := LoadRepository(filepath.Join(repoRootFromCaller(t), "testdata", "import", "fixture_repo"))
	if err != nil {
		t.Fatalf("load repository failed: %v", err)
	}
	if _, err := ImportRepository(workspace, loaded); err != nil {
		t.Fatalf("import repository failed: %v", err)
	}
	reg, _, err := LoadRegistry(workspace)
	if err != nil {
		t.Fatalf("load registry failed: %v", err)
	}
	var skillID string
	for _, e := range reg.Entries {
		if e.Kind == KindSkill {
			skillID = e.ID
			break
		}
	}
	if skillID == "" {
		t.Fatalf("skill fixture not found")
	}
	if _, err := ConvertRecipeCandidate(workspace, skillID, ReviewApproved, "deterministic", "", []string{"safe.echo"}); err != nil {
		t.Fatalf("convert recipe candidate failed: %v", err)
	}
	all, err := BuildCatalog(workspace, CatalogOptions{ReviewedOnly: false})
	if err != nil {
		t.Fatalf("build catalog failed: %v", err)
	}
	reviewed, err := BuildCatalog(workspace, CatalogOptions{ReviewedOnly: true})
	if err != nil {
		t.Fatalf("build reviewed-only catalog failed: %v", err)
	}
	if len(reviewed.Items) == 0 {
		t.Fatalf("expected reviewed-only catalog entries")
	}
	if len(reviewed.Items) >= len(all.Items) {
		t.Fatalf("reviewed-only opt-in should narrow input set")
	}
}

func TestCatalogDeterministicForSameState(t *testing.T) {
	workspace := t.TempDir()
	loaded, err := LoadRepository(filepath.Join(repoRootFromCaller(t), "testdata", "import", "fixture_repo"))
	if err != nil {
		t.Fatalf("load repository failed: %v", err)
	}
	if _, err := ImportRepository(workspace, loaded); err != nil {
		t.Fatalf("import repository failed: %v", err)
	}
	c1, err := BuildCatalog(workspace, CatalogOptions{ReviewedOnly: false})
	if err != nil {
		t.Fatalf("build first catalog failed: %v", err)
	}
	c2, err := BuildCatalog(workspace, CatalogOptions{ReviewedOnly: false})
	if err != nil {
		t.Fatalf("build second catalog failed: %v", err)
	}
	if len(c1.Items) != len(c2.Items) {
		t.Fatalf("catalog size changed across same state")
	}
	for i := range c1.Items {
		if c1.Items[i] != c2.Items[i] {
			t.Fatalf("catalog order/content changed at %d: %#v != %#v", i, c1.Items[i], c2.Items[i])
		}
	}
}

func hasEventType(list []events.Event, typ string) bool {
	for _, e := range list {
		if t, _ := e.Raw["type"].(string); t == typ {
			return true
		}
	}
	return false
}

func buildFixtureImportSource(t *testing.T, forward bool) string {
	t.Helper()
	repo := t.TempDir()
	files := []struct {
		Rel string
		Txt string
	}{
		{"skills/alpha.skill.md", "# alpha\n"},
		{"memory/ref_memory.md", "reference\n"},
		{"api/openapi.yaml", "openapi: 3.0.0\n"},
	}
	if !forward {
		rev := make([]struct {
			Rel string
			Txt string
		}, len(files))
		copy(rev, files)
		sort.Slice(rev, func(i, j int) bool { return rev[i].Rel > rev[j].Rel })
		files = rev
	}
	for _, f := range files {
		p := filepath.Join(repo, filepath.FromSlash(f.Rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(p, []byte(f.Txt), 0o644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}
	return repo
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
