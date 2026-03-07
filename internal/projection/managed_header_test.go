package projection

import "testing"

func TestManagedHeaderForPathUsesCommentStyle(t *testing.T) {
	hGo := ManagedHeaderForPath("main.go", "tmpl", "fp")
	if len(hGo) == 0 || hGo[:2] != "//" {
		t.Fatalf("expected go header with // prefix, got %q", hGo)
	}
	hJSON := ManagedHeaderForPath("a.json", "tmpl", "fp")
	if hJSON != "" {
		t.Fatalf("expected no header for json, got %q", hJSON)
	}
}
