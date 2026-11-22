package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackImportPreservesExecBit(t *testing.T) {
	home := t.TempDir()
	pdir := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(pdir, "scripts"), 0o755)
	os.WriteFile(filepath.Join(pdir, "scripts", "pre-commit"), []byte("#!/bin/sh\n"), 0o755)
	pack := filepath.Join(home, "p.tar.gz")
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"pack", "export", "--profile", "personal", "--output", pack})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetArgs([]string{"pack", "import", "--profile", "friend", "--source", pack})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(home, ".hermes", "profiles", "friend", "scripts", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&0o111 == 0 {
		t.Fatalf("exec bit lost: %v", fi.Mode())
	}
}
