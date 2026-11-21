package pack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEjectRemovesProfileNotHome(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(prof, 0o755)
	os.WriteFile(filepath.Join(prof, "x"), []byte("1"), 0o600)
	rep, err := Eject("personal", home, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, e := os.Stat(prof); !os.IsNotExist(e) {
		t.Fatal("profile not removed")
	}
	if _, e := os.Stat(home); e != nil {
		t.Fatal("HOME removed")
	}
	if len(rep.Residue) == 0 {
		t.Fatal("expected residue report")
	}
}

func TestEjectRefusesEmptyProfile(t *testing.T) {
	if _, err := Eject("", t.TempDir(), "", false); err == nil {
		t.Fatal("expected refusal")
	}
}

func TestEjectRefusesTraversalProfile(t *testing.T) {
	if _, err := Eject("../../etc", t.TempDir(), "", false); err == nil {
		t.Fatal("expected refusal for traversal profile")
	}
}

func TestEjectDryRunKeepsFiles(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(prof, 0o755)
	if _, err := Eject("personal", home, "", true); err != nil {
		t.Fatal(err)
	}
	if _, e := os.Stat(prof); e != nil {
		t.Fatal("dry-run deleted")
	}
}

func TestEjectRemovesSecrets(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(prof, 0o755)
	secrets := filepath.Join(home, "secrets-central")
	os.MkdirAll(secrets, 0o755)
	os.WriteFile(filepath.Join(secrets, "a.env"), []byte("k=v"), 0o600)
	if _, err := Eject("personal", home, secrets, false); err != nil {
		t.Fatal(err)
	}
	if _, e := os.Stat(secrets); !os.IsNotExist(e) {
		t.Fatal("secrets not removed")
	}
}
