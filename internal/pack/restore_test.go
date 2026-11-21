package pack

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupRestoreRoundTrip(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal", "memory")
	os.MkdirAll(prof, 0o755)
	os.WriteFile(filepath.Join(prof, "MEMORY.md"), []byte("hello"), 0o600)
	var arc bytes.Buffer
	if err := GatherPersonalToEncrypted(PersonalSource{Profile: "personal", HermesHome: home}, &arc, "pw"); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := RestorePersonal(&arc, "pw", dest, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, ".hermes", "profiles", "personal", "memory", "MEMORY.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestRestoreWrongPassphraseNoTouch(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal"), 0o755)
	var arc bytes.Buffer
	_ = GatherPersonalToEncrypted(PersonalSource{Profile: "personal", HermesHome: home}, &arc, "right")
	dest := t.TempDir()
	if err := RestorePersonal(&arc, "wrong", dest, false); err == nil {
		t.Fatal("expected error")
	}
	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Fatal("dest touched on failure")
	}
}

func TestRestoreCollisionNeedsForce(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes", "profiles", "personal", "memory"), 0o755)
	os.WriteFile(filepath.Join(home, ".hermes", "profiles", "personal", "memory", "M.md"), []byte("a"), 0o600)
	var arc bytes.Buffer
	_ = GatherPersonalToEncrypted(PersonalSource{Profile: "personal", HermesHome: home}, &arc, "pw")
	dest := t.TempDir()
	os.MkdirAll(filepath.Join(dest, ".hermes", "profiles", "personal"), 0o755)
	arcBytes := arc.Bytes()
	if err := RestorePersonal(bytes.NewReader(arcBytes), "pw", dest, false); err == nil {
		t.Fatal("expected collision error")
	}
	if err := RestorePersonal(bytes.NewReader(arcBytes), "pw", dest, true); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsTraversal(t *testing.T) {
	var tarbuf bytes.Buffer
	gz := gzip.NewWriter(&tarbuf)
	tw := tar.NewWriter(gz)
	m := Manifest{SourceProfile: "personal", Files: []string{}}
	md, _ := json.Marshal(m)
	tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0o644, Size: int64(len(md))})
	tw.Write(md)
	evil := "profile/../../../../../../../../evil"
	tw.WriteHeader(&tar.Header{Name: evil, Mode: 0o600, Size: 3})
	tw.Write([]byte("bad"))
	tw.Close()
	gz.Close()
	var arc bytes.Buffer
	if err := EncryptStream(&arc, &tarbuf, "pw"); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := RestorePersonal(&arc, "pw", dest, false); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
