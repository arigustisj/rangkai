package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupProducesEncryptedArchive(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal", "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".hermes", "profiles", "personal", "memory", "MEMORY.md"), []byte("m"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(home, "b.tar.gz.age")
	t.Setenv("HOME", home)
	t.Setenv("RANGKAI_PASSPHRASE", "test-pass-123")
	rootCmd.SetArgs([]string{"backup", "--profile", "personal", "--output", out, "--ssot", "", "--secrets", ""})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("age-encryption.org/")) {
		t.Fatal("not age ciphertext")
	}
}

func TestBackupRequiresPassphrase(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal"), 0o755)
	t.Setenv("HOME", home)
	t.Setenv("RANGKAI_PASSPHRASE", "")
	rootCmd.SetArgs([]string{"backup", "--profile", "personal", "--output", filepath.Join(home, "x.age"), "--ssot", "", "--secrets", ""})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected passphrase error")
	}
}
