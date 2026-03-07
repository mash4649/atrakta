package wrapper

import (
	"path/filepath"
	"testing"
)

func TestPathPosition(t *testing.T) {
	bin := filepath.Clean("/tmp/a/.local/bin")
	contains, prefers := pathPosition(bin, bin+string(filepath.ListSeparator)+"/usr/bin")
	if !contains || !prefers {
		t.Fatalf("expected preferred bin path")
	}
	contains, prefers = pathPosition(bin, "/usr/bin"+string(filepath.ListSeparator)+bin)
	if !contains || prefers {
		t.Fatalf("expected contained but not preferred")
	}
	contains, prefers = pathPosition(bin, "/usr/bin")
	if contains || prefers {
		t.Fatalf("expected missing bin path")
	}
}
