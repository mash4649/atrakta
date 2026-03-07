package syncpolicy

import (
	"testing"

	"atrakta/internal/contract"
)

func TestProposeFromAGENTS(t *testing.T) {
	c := contract.Default("/tmp/repo")
	agents := "sync.prefer_interfaces: cursor,trae\n" +
		"sync.disable_interfaces: opencode\n"

	sp, proposed, err := ProposeFromAGENTS(c, agents)
	if err != nil {
		t.Fatalf("proposal failed: %v", err)
	}
	if !sp.Needed || !sp.RequiresApproval {
		t.Fatalf("expected needed proposal")
	}
	if proposed.Hints == nil {
		t.Fatalf("expected hints initialized")
	}
	if len(proposed.Hints.Prefer) != 2 {
		t.Fatalf("unexpected prefer: %#v", proposed.Hints.Prefer)
	}
	if len(proposed.Hints.DisableInterfaces) != 1 || proposed.Hints.DisableInterfaces[0] != "opencode" {
		t.Fatalf("unexpected disable: %#v", proposed.Hints.DisableInterfaces)
	}
}

func TestProposeLevelParser(t *testing.T) {
	if ParseLevel("2") != Level2 {
		t.Fatal("expected level2")
	}
	if ParseLevel("level1") != Level1 {
		t.Fatal("expected level1")
	}
	if ParseLevel("") != Level0 {
		t.Fatal("expected level0")
	}
}
