package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorCmd(t *testing.T) {
	home := t.TempDir()
	pdir := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(pdir, "skills"), 0o755)
	os.WriteFile(filepath.Join(pdir, "skills", "a.md"), []byte("x"), 0o600)
	os.WriteFile(filepath.Join(pdir, "config.yaml"), []byte("model:\n  provider: \"openrouter:a\"\nmcp:\n  enabled: [x]\n"), 0o644)
	os.MkdirAll(filepath.Join(pdir, "scripts"), 0o755)
	os.WriteFile(filepath.Join(pdir, "scripts", "pre-commit"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"doctor", "--profile", "personal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorCmdFailsWhenBroken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"doctor", "--profile", "nope"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected failure for a missing profile")
	}
}
