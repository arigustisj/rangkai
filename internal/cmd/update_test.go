package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCmd(t *testing.T) {
	home := t.TempDir()
	pdir := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(pdir, "skills"), 0o755)
	os.WriteFile(filepath.Join(pdir, "config.yaml"), []byte("# Profile: personal\n# Stack: go\n# Model: openrouter:a\nmodel:\n  provider: \"openrouter:a\"\n"), 0o644)
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"update", "--profile", "personal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(pdir, "install.sh")); err != nil {
		t.Fatal("install.sh not regenerated")
	}
	b, _ := os.ReadFile(filepath.Join(pdir, "config.yaml"))
	if !strings.Contains(string(b), `provider: "openrouter:a"`) {
		t.Fatalf("model lost: %s", b)
	}
}

func TestUpdateCmdMissingProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"update", "--profile", "nope"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for a missing profile")
	}
}
