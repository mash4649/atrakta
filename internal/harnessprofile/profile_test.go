package harnessprofile

import (
	"testing"
)

func TestGenerateCurrentProfile(t *testing.T) {
	report, err := Generate(t.TempDir(), "current")
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}
	if report.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version=%q", report.SchemaVersion)
	}
	if report.ModelGeneration != "current" {
		t.Fatalf("model generation=%q", report.ModelGeneration)
	}
	if len(report.LoadBearingComponents) != 3 {
		t.Fatalf("load bearing components=%v", report.LoadBearingComponents)
	}
	if len(report.RetirableComponents) != 0 {
		t.Fatalf("retirable components=%v", report.RetirableComponents)
	}
	if len(report.Ablations) != 3 {
		t.Fatalf("ablation count=%d", len(report.Ablations))
	}
}

func TestGenerateNextGenerationProfileRetiresReset(t *testing.T) {
	report, err := Generate(t.TempDir(), "gpt-5.4")
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}
	if report.ModelGeneration != "gpt-5.4" {
		t.Fatalf("model generation=%q", report.ModelGeneration)
	}
	if len(report.LoadBearingComponents) != 2 {
		t.Fatalf("load bearing components=%v", report.LoadBearingComponents)
	}
	if len(report.RetirableComponents) != 1 || report.RetirableComponents[0] != "reset" {
		t.Fatalf("retirable components=%v", report.RetirableComponents)
	}
}
