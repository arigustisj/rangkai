package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withFakeSSHHome(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	old := sshHomeDir
	sshHomeDir = func() string { return h }
	t.Cleanup(func() { sshHomeDir = old })
	return h
}

func fakeKeygen(t *testing.T) *int {
	t.Helper()
	n := 0
	old := runKeygen
	runKeygen = func(keyPath, comment string) error {
		n++
		if err := os.WriteFile(keyPath, []byte("PRIVATE"), 0o600); err != nil {
			return err
		}
		return os.WriteFile(keyPath+".pub", []byte("ssh-ed25519 AAAA "+comment), 0o600)
	}
	t.Cleanup(func() { runKeygen = old })
	return &n
}

func TestParseHubTarget(t *testing.T) {
	a, err := ParseHubTarget("user@host:/path", "rangkai-x")
	if err != nil {
		t.Fatal(err)
	}
	if a.User != "user" || a.Host != "host" || a.Path != "/path" || a.Alias != "rangkai-x" {
		t.Fatalf("%+v", a)
	}
	b, err := ParseHubTarget("host:/path", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.User != "" || b.Host != "host" {
		t.Fatalf("%+v", b)
	}
	if _, err := ParseHubTarget("nocolon", ""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseHubTarget("host:", ""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestEnsureSSHKeyCreatesOnce(t *testing.T) {
	withFakeSSHHome(t)
	n := fakeKeygen(t)
	k1, p1, created, err := EnsureSSHKey("personal")
	if err != nil {
		t.Fatal(err)
	}
	if !created || *n != 1 {
		t.Fatalf("created=%v calls=%d", created, *n)
	}
	k2, p2, created2, err := EnsureSSHKey("personal")
	if err != nil {
		t.Fatal(err)
	}
	if created2 || *n != 1 || k1 != k2 || p1 != p2 {
		t.Fatalf("reuse failed: created=%v calls=%d", created2, *n)
	}
	if strings.Contains(p1, "PRIVATE") {
		t.Fatal("public path is wrong")
	}
}

func TestEnsureSSHKeyFailsCleanly(t *testing.T) {
	withFakeSSHHome(t)
	old := runKeygen
	runKeygen = func(keyPath, comment string) error { return os.ErrNotExist }
	t.Cleanup(func() { runKeygen = old })
	if _, _, _, err := EnsureSSHKey("personal"); err == nil {
		t.Fatal("expected an error when keygen fails")
	}
}

func TestWriteHubAliasIdempotent(t *testing.T) {
	h := withFakeSSHHome(t)
	tgt := HubTarget{User: "u", Host: "h", Path: "/p", Alias: "rangkai-x"}
	if err := WriteHubAlias(tgt, "/k"); err != nil {
		t.Fatal(err)
	}
	if err := WriteHubAlias(tgt, "/k"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(h, ".ssh", "config"))
	if n := strings.Count(string(b), "Host rangkai-x"); n != 1 {
		t.Fatalf("stanza count=%d: %s", n, b)
	}
	if !strings.Contains(string(b), "IdentityFile /k") {
		t.Fatalf("missing key: %s", b)
	}
}

func TestBootstrapProfile(t *testing.T) {
	withFakeSSHHome(t)
	fakeKeygen(t)
	tgt, err := ParseHubTarget("user@host:/path", "")
	if err != nil {
		t.Fatal(err)
	}
	alias, pub, err := BootstrapProfile(tgt, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if alias != "rangkai-personal" {
		t.Fatalf("alias=%q", alias)
	}
	if pub != "ssh-ed25519 AAAA rangkai-personal" {
		t.Fatalf("pub=%q", pub)
	}
	if strings.Contains(pub, "PRIVATE") {
		t.Fatal("leaked the private key")
	}
}
