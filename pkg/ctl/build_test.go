package ctl

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestFindModuleRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "strategies", "demo")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	goMod := []byte("module example.com/strategy\n\ngo 1.25\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("write go.mod failed: %v", err)
	}

	got, err := findModuleRoot(nested)
	if err != nil {
		t.Fatalf("findModuleRoot returned error: %v", err)
	}
	if got != root {
		t.Fatalf("findModuleRoot = %s, want %s", got, root)
	}
}

func TestResolveModuleRootIgnoresSourceModuleRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "strategies", "demo")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	goMod := []byte("module example.com/strategy\n\ngo 1.25\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("write go.mod failed: %v", err)
	}

	b := NewBuilder(filepath.Join(nested, "strategy.go"), "strategy.so")
	b.SetIgnoreSourceModuleRoot(true)
	got, explicit, err := b.resolveModuleRoot(nested)
	if err != nil {
		t.Fatalf("resolveModuleRoot returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("resolveModuleRoot = %s, want empty", got)
	}
	if explicit {
		t.Fatal("resolveModuleRoot explicit = true, want false")
	}
}

func TestFixGoModRewritesRelativeReplace(t *testing.T) {
	moduleRoot := t.TempDir()
	tempDir := t.TempDir()
	privateDir := filepath.Join(moduleRoot, "private")
	if err := os.MkdirAll(privateDir, 0755); err != nil {
		t.Fatalf("mkdir private failed: %v", err)
	}
	rootGoMod := []byte("module example.com/root\n\ngo 1.25\n")
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), rootGoMod, 0644); err != nil {
		t.Fatalf("write module root go.mod failed: %v", err)
	}
	goMod := []byte("module example.com/strategy\n\ngo 1.25\n\nrequire example.com/private v0.0.0\n\nreplace example.com/private => ./private\n")
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), goMod, 0644); err != nil {
		t.Fatalf("write go.mod failed: %v", err)
	}

	b := NewBuilder(filepath.Join(moduleRoot, "strategy.go"), filepath.Join(tempDir, "strategy.so"))
	b.SetModuleRoot(moduleRoot)

	changed, err := b.fixGoMod(tempDir)
	if err != nil {
		t.Fatalf("fixGoMod returned error: %v", err)
	}
	if !changed {
		t.Fatal("fixGoMod did not report changes")
	}

	buf, err := os.ReadFile(filepath.Join(tempDir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod failed: %v", err)
	}
	mf, err := modfile.Parse("go.mod", buf, nil)
	if err != nil {
		t.Fatalf("parse go.mod failed: %v", err)
	}
	if len(mf.Replace) != 1 {
		t.Fatalf("replace count = %d, want 1", len(mf.Replace))
	}
	if mf.Replace[0].New.Path != privateDir {
		t.Fatalf("replace path = %s, want %s", mf.Replace[0].New.Path, privateDir)
	}
}
