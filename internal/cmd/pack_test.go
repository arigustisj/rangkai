package cmd

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func packTarFiles(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var out []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, h.Name)
	}
	return out
}

func packHasFile(t *testing.T, path, want string) bool {
	t.Helper()
	for _, f := range packTarFiles(t, path) {
		if f == want {
			return true
		}
	}
	return false
}

func TestPackExportDefaultOmitsMemory(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal", "memories"), 0o755)
	os.WriteFile(filepath.Join(home, ".hermes", "profiles", "personal", "memories", "MEMORY.md"), []byte("x"), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"pack", "export", "--profile", "personal", "--agent", "hermes", "--output", out})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if packHasFile(t, out, "memory/MEMORY.md") {
		t.Fatal("default export leaked personal memory")
	}
}

func TestPackExportWithMemory(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal", "memories"), 0o755)
	os.WriteFile(filepath.Join(home, ".hermes", "profiles", "personal", "memories", "MEMORY.md"), []byte("x"), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"pack", "export", "--profile", "personal", "--agent", "hermes", "--output", out, "--with-memory"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !packHasFile(t, out, "memory/MEMORY.md") {
		t.Fatal("--with-memory did not include memory")
	}
}
