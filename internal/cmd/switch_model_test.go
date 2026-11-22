package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwitchModelCmd(t *testing.T) {
	home := t.TempDir()
	pdir := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(pdir, 0o755)
	os.WriteFile(filepath.Join(pdir, "config.yaml"), []byte("model:\n  provider: \"openrouter:a\"\n"), 0o644)
	t.Setenv("HOME", home)
	rootCmd.SetArgs([]string{"switch-model", "ollama:llama3", "--profile", "personal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(pdir, "config.yaml"))
	if !strings.Contains(string(b), `provider: "ollama:llama3"`) {
		t.Fatalf("not switched: %s", b)
	}
}

func TestSwitchModelCmdRejectsBadArg(t *testing.T) {
	rootCmd.SetArgs([]string{"switch-model", "nocolon"})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for missing colon")
	}
}
