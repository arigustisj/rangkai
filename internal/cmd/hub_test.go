package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arigustisj/rangkai/internal/pack"
)

func TestPushPullCmdFakeTransport(t *testing.T) {
	fakeHub := t.TempDir()
	resolve := func(p string) string {
		if i := strings.Index(p, ":"); i >= 0 {
			return filepath.Join(fakeHub, p[i+1:])
		}
		return p
	}
	restore := pack.SetTransportForTest(
		func(args ...string) error {
			data, err := os.ReadFile(resolve(args[len(args)-2]))
			if err != nil {
				return err
			}
			return os.WriteFile(resolve(args[len(args)-1]), data, 0o600)
		},
		func(host, script string) error {
			if strings.HasPrefix(script, "mkdir -p ") {
				return os.MkdirAll(filepath.Join(fakeHub, strings.TrimPrefix(script, "mkdir -p ")), 0o755)
			}
			return nil
		},
	)
	t.Cleanup(restore)

	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal", "memory"), 0o755)
	os.WriteFile(filepath.Join(home, ".hermes", "profiles", "personal", "memory", "MEMORY.md"), []byte("cmd-carry"), 0o600)
	t.Setenv("HOME", home)
	t.Setenv("RANGKAI_PASSPHRASE", "pw")
	rootCmd.SetArgs([]string{"push", "--hub", "fakehost:rangkai-hub", "--profile", "personal", "--secrets", "", "--ssot", ""})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	t.Setenv("HOME", dest)
	rootCmd.SetArgs([]string{"pull", "--hub", "fakehost:rangkai-hub", "--profile", "personal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, ".hermes", "profiles", "personal", "memory", "MEMORY.md"))
	if err != nil || string(got) != "cmd-carry" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestPushRequiresHub(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal"), 0o755)
	t.Setenv("HOME", home)
	t.Setenv("RANGKAI_PASSPHRASE", "pw")
	rootCmd.SetArgs([]string{"push", "--profile", "personal", "--secrets", "", "--ssot", ""})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected hub required error")
	}
}
