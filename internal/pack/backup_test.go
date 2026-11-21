package pack

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatherPersonalIncludesMemoryAndSecrets(t *testing.T) {
	home := t.TempDir()
	profile := filepath.Join(home, ".hermes", "profiles", "personal")
	if err := os.MkdirAll(filepath.Join(profile, "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "memory", "MEMORY.md"), []byte("m"), 0o600); err != nil {
		t.Fatal(err)
	}
	secrets := filepath.Join(home, "secrets-central")
	if err := os.MkdirAll(secrets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secrets, "a.env"), []byte("k=v"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	m, err := GatherPersonal(PersonalSource{Profile: "personal", HermesHome: home, SecretsDir: secrets}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(m.Files, "\n")
	if !strings.Contains(joined, "memory/MEMORY.md") || !strings.Contains(joined, "a.env") {
		t.Fatalf("manifest missing paths: %v", m.Files)
	}
	if buf.Len() == 0 {
		t.Fatal("empty archive")
	}
}

func TestGatherPersonalMissingProfileFails(t *testing.T) {
	if _, err := GatherPersonal(PersonalSource{Profile: "nope", HermesHome: t.TempDir()}, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestGatherPersonalSkipsRuntimeJunk(t *testing.T) {
	home := t.TempDir()
	profile := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(profile, "cache"), 0o755)
	os.WriteFile(filepath.Join(profile, "cache", "big.bin"), []byte("junk"), 0o600)
	os.MkdirAll(filepath.Join(profile, "memory"), 0o755)
	os.WriteFile(filepath.Join(profile, "memory", "MEMORY.md"), []byte("m"), 0o600)
	var buf bytes.Buffer
	m, _ := GatherPersonal(PersonalSource{Profile: "personal", HermesHome: home}, &buf)
	for _, f := range m.Files {
		if strings.Contains(f, "cache/") {
			t.Fatalf("runtime junk included: %s", f)
		}
	}
}
