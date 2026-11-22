package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// resetBootstrapFlags clears cobra's persisted flag values between tests.
func resetBootstrapFlags(t *testing.T) {
	t.Helper()
	oldHub, oldProfile := bootstrapHub, bootstrapProfile
	bootstrapHub, bootstrapProfile = "", "personal"
	t.Cleanup(func() { bootstrapHub, bootstrapProfile = oldHub, oldProfile })
}

func TestBootstrapCmdNoPull(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}
	t.Setenv("HOME", t.TempDir())
	resetBootstrapFlags(t)
	rootCmd.SetArgs([]string{"bootstrap", "--hub", "user@example.com:/hub", "--profile", "personal", "--no-pull"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(filepath.Join(home, ".ssh", "rangkai_personal")); err != nil {
		t.Fatal("ssh key not created")
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh", "config")); err != nil {
		t.Fatal("ssh config not written")
	}
}

func TestBootstrapCmdRequiresHub(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetBootstrapFlags(t)
	rootCmd.SetArgs([]string{"bootstrap"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected an error without --hub")
	}
}
