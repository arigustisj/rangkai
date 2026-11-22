package pack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirSkipsSymlink(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "src")
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o600)
	// Reproduces the real-world failure: a nested symlink pointing at a directory.
	real := filepath.Join(base, "real")
	os.MkdirAll(real, 0o755)
	os.Symlink(real, filepath.Join(src, "linkdir"))
	dst := filepath.Join(base, "dst")
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "a.txt")); err != nil {
		t.Fatal("a.txt missing")
	}
}
