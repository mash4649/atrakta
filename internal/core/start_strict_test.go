package core

import (
	"reflect"
	"testing"

	"atrakta/internal/doctor"
)

func TestStrictRuntimeBlockingReasons(t *testing.T) {
	parity := doctor.ParityReport{
		BlockingIssues: []doctor.ParityFinding{
			{Code: "managed_block_corruption", Message: "managed projection artifact failed revalidation"},
		},
		Warnings: []doctor.ParityFinding{
			{Code: "extension_projection_drift", Message: "extension manifest file is missing"},
			{Code: "noninteractive_mismatch", Message: "stdin mismatch"},
		},
	}
	integration := doctor.IntegrationReport{
		BlockingIssues: []doctor.IntegrationFinding{
			{Code: "append_failure", Message: "managed append block markers are unbalanced"},
		},
		Warnings: []doctor.IntegrationFinding{
			{Code: "include_missing", Message: "include mode append file is missing"},
			{Code: "unsupported_extension_projection", Message: "extension is enabled but no native projection is available yet"},
		},
	}

	got := strictRuntimeBlockingReasons(parity, nil, integration, nil)
	want := []string{
		"integration.append_failure: managed append block markers are unbalanced",
		"integration.include_missing: include mode append file is missing",
		"parity.extension_projection_drift: extension manifest file is missing",
		"parity.managed_block_corruption: managed projection artifact failed revalidation",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected strict blocking reasons:\n got=%v\nwant=%v", got, want)
	}
}

func TestStrictRuntimeBlockingReasonsDeduplicatesAndSorts(t *testing.T) {
	parity := doctor.ParityReport{
		BlockingIssues: []doctor.ParityFinding{
			{Code: "manifest_missing", Message: "projection manifest is missing"},
			{Code: "manifest_missing", Message: "projection manifest is missing"},
		},
	}
	integration := doctor.IntegrationReport{
		Warnings: []doctor.IntegrationFinding{
			{Code: "include_missing", Message: "include mode append file is missing"},
			{Code: "include_missing", Message: "include mode append file is missing"},
		},
	}
	got := strictRuntimeBlockingReasons(parity, nil, integration, nil)
	want := []string{
		"integration.include_missing: include mode append file is missing",
		"parity.manifest_missing: projection manifest is missing",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected dedup result: got=%v want=%v", got, want)
	}
}
