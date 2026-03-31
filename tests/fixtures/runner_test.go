package fixtures_test

import (
	"testing"

	"github.com/mash4649/atrakta/v0/internal/fixtures"
)

func TestRunAllFixtures(t *testing.T) {
	report, err := fixtures.RunAll("../../fixtures")
	if err != nil {
		t.Fatalf("run fixtures error: %v", err)
	}
	if report.Failed != 0 {
		t.Fatalf("fixture failures: %d", report.Failed)
	}
	if report.Passed == 0 {
		t.Fatalf("expected passed fixtures")
	}
}
