package maa

import (
	"path/filepath"
	"testing"
)

func TestNewAtWithResourceUsesSharedResourceRoot(t *testing.T) {
	root := t.TempDir()
	core := NewAtWithResource(
		filepath.Join(root, "core"),
		filepath.Join(root, "base"),
	)
	want := filepath.Join(root, "base", "resource")
	if got := core.ResourceDir(); got != want {
		t.Fatalf("ResourceDir = %q, want %q", got, want)
	}
	if core.Dir() != filepath.Join(root, "core") {
		t.Fatalf("Dir = %q", core.Dir())
	}
}

func TestNewAtUsesParentResourceRoot(t *testing.T) {
	root := t.TempDir()
	core := NewAt(filepath.Join(root, "core"))
	want := filepath.Join(root, "resource")
	if got := core.ResourceDir(); got != want {
		t.Fatalf("ResourceDir = %q, want %q", got, want)
	}
}
