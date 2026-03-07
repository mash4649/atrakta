package contract

import "testing"

func TestValidateRejectsInvalidNewFields(t *testing.T) {
	c := Default(t.TempDir())
	c.Security.Profile = "danger"
	if err := Validate(c); err == nil {
		t.Fatalf("expected invalid security profile to fail")
	}

	c = Default(t.TempDir())
	c.Context.Resolution = "root_only"
	if err := Validate(c); err == nil {
		t.Fatalf("expected invalid context resolution to fail")
	}

	c = Default(t.TempDir())
	c.Routing.Default.Quality = "fast"
	if err := Validate(c); err == nil {
		t.Fatalf("expected invalid routing quality to fail")
	}

	c = Default(t.TempDir())
	c.EditSafety.Languages["go"] = "invalid"
	if err := Validate(c); err == nil {
		t.Fatalf("expected invalid edit_safety language policy to fail")
	}

	c = Default(t.TempDir())
	ro := false
	c.Context.ConventionsReadOnly = &ro
	if err := Validate(c); err == nil {
		t.Fatalf("expected conventions_read_only=false to fail")
	}
}

func TestCanonicalizeAppliesNewDefaults(t *testing.T) {
	c := Default(t.TempDir())
	c.Context = &Context{}
	c.Security = &Security{}
	c.Policies = &Policies{PromptMin: &PromptMinRef{Ref: "./.atrakta/policies/prompt-min.json"}}

	n := CanonicalizeBoundary(c)
	if n.Context == nil || n.Context.Resolution != "nearest_with_import" || n.Context.MaxImportDepth != 6 {
		t.Fatalf("unexpected context defaults: %#v", n.Context)
	}
	if n.Context.RepoMapTokens != 1200 || n.Context.RepoMapRefreshSec != 300 {
		t.Fatalf("unexpected repo map defaults: %#v", n.Context)
	}
	if len(n.Context.Conventions) == 0 {
		t.Fatalf("expected default conventions")
	}
	if n.Context.ConventionsReadOnly == nil || !*n.Context.ConventionsReadOnly {
		t.Fatalf("expected conventions_read_only=true default")
	}
	if ResolveSecurityProfile(n) != "workspace_write" {
		t.Fatalf("unexpected security default: %s", ResolveSecurityProfile(n))
	}
	if n.Policies == nil || n.Policies.PromptMin == nil || n.Policies.PromptMin.Ref != ".atrakta/policies/prompt-min.json" {
		t.Fatalf("unexpected policy ref normalization: %#v", n.Policies)
	}
	if n.EditSafety == nil || n.EditSafety.Languages["go"] != "ast" || n.EditSafety.Languages["json"] != "parse" {
		t.Fatalf("unexpected edit safety language defaults: %#v", n.EditSafety)
	}
}
