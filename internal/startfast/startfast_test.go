package startfast

import (
	"testing"
	"time"
)

func TestCheckHitAndStrictInterval(t *testing.T) {
	repo := t.TempDir()
	in := Input{
		ContractHash:   "c1",
		WorkspaceStamp: "w1",
		Interfaces:     "cursor",
		FeatureID:      "adhoc",
		ConfigKey:      "cfg1",
	}
	now := time.Now().UTC()
	if err := SaveSuccess(repo, in, "explicit", now); err != nil {
		t.Fatalf("save success failed: %v", err)
	}
	dec, err := Check(repo, in, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !dec.Hit {
		t.Fatalf("expected fast-path hit, got miss: %s", dec.Reason)
	}
	dec, err = Check(repo, in, now.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if dec.Hit || dec.Reason != "strict_interval_elapsed" {
		t.Fatalf("expected strict interval miss, got: %#v", dec)
	}
}

func TestCheckMissOnInputChange(t *testing.T) {
	repo := t.TempDir()
	base := Input{
		ContractHash:   "c1",
		WorkspaceStamp: "w1",
		Interfaces:     "cursor",
		FeatureID:      "adhoc",
		ConfigKey:      "cfg1",
	}
	if err := SaveSuccess(repo, base, "explicit", time.Now().UTC()); err != nil {
		t.Fatalf("save success failed: %v", err)
	}
	cases := []struct {
		name string
		in   Input
	}{
		{name: "contract", in: Input{ContractHash: "c2", WorkspaceStamp: "w1", Interfaces: "cursor", FeatureID: "adhoc", ConfigKey: "cfg1"}},
		{name: "workspace", in: Input{ContractHash: "c1", WorkspaceStamp: "w2", Interfaces: "cursor", FeatureID: "adhoc", ConfigKey: "cfg1"}},
		{name: "interfaces", in: Input{ContractHash: "c1", WorkspaceStamp: "w1", Interfaces: "trae", FeatureID: "adhoc", ConfigKey: "cfg1"}},
		{name: "feature", in: Input{ContractHash: "c1", WorkspaceStamp: "w1", Interfaces: "cursor", FeatureID: "feat", ConfigKey: "cfg1"}},
		{name: "config", in: Input{ContractHash: "c1", WorkspaceStamp: "w1", Interfaces: "cursor", FeatureID: "adhoc", ConfigKey: "cfg2"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dec, err := Check(repo, tc.in, time.Now().UTC())
			if err != nil {
				t.Fatalf("check failed: %v", err)
			}
			if dec.Hit {
				t.Fatalf("expected miss for %s", tc.name)
			}
		})
	}
}
