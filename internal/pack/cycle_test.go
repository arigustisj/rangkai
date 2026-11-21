package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestFullCarryCycle exercises the whole portable flow end to end: gather and
// encrypt a personal environment, restore it into a fresh machine, verify memory
// and secrets survived, then eject the restored machine cleanly.
func TestFullCarryCycle(t *testing.T) {
	src := t.TempDir()
	prof := filepath.Join(src, ".hermes", "profiles", "personal", "memory")
	os.MkdirAll(prof, 0o755)
	os.WriteFile(filepath.Join(prof, "MEMORY.md"), []byte("carry-me"), 0o600)
	secrets := filepath.Join(src, "secrets-central")
	os.MkdirAll(secrets, 0o755)
	os.WriteFile(filepath.Join(secrets, "a.env"), []byte("k=v"), 0o600)

	var arc bytes.Buffer
	if err := GatherPersonalToEncrypted(PersonalSource{Profile: "personal", HermesHome: src, SecretsDir: secrets}, &arc, "pw"); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := RestorePersonal(bytes.NewReader(arc.Bytes()), "pw", dest, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, ".hermes", "profiles", "personal", "memory", "MEMORY.md"))
	if err != nil || string(got) != "carry-me" {
		t.Fatalf("memory not carried: %v %q", err, got)
	}
	if _, err := os.Stat(filepath.Join(dest, "secrets-central", "a.env")); err != nil {
		t.Fatal("secrets not carried")
	}

	if _, err := Eject("personal", dest, filepath.Join(dest, "secrets-central"), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, ".hermes", "profiles", "personal")); !os.IsNotExist(err) {
		t.Fatal("eject left profile")
	}
	if _, err := os.Stat(filepath.Join(dest, "secrets-central")); !os.IsNotExist(err) {
		t.Fatal("eject left secrets")
	}
}
