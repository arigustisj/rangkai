package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintFindsLeaks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("K=1"), 0o600)
	os.WriteFile(filepath.Join(dir, "token.txt"), []byte("token ghp_"+strings.Repeat("a", 36)), 0o600)
	f, err := LintPath(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f) < 2 {
		t.Fatalf("expected >=2 findings, got %v", f)
	}
}

func TestLintCleanPlaceholdersPass(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("host: 192.0.2.10\nsite: ssh.example.com\n"), 0o600)
	f, err := LintPath(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("placeholders flagged: %v", f)
	}
}

func TestLintPackMissing(t *testing.T) {
	if _, err := LintPack(filepath.Join(t.TempDir(), "nope.tar.gz"), nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestLintIgnoresPlaceholderExamples(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "doc.md"), []byte(`Authorization: "Bearer sk-`+strings.Repeat("x", 24)+`"`), 0o600)
	f, err := LintPath(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f) != 0 {
		t.Fatalf("placeholder flagged: %v", f)
	}
}

func TestLintPackFindsLeak(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, ".env"), []byte("K=1"), 0o600)
	packPath := filepath.Join(t.TempDir(), "p.tar.gz")
	e := &Exporter{OutputPath: packPath}
	if err := e.createTarGz(src); err != nil {
		t.Fatal(err)
	}
	f, err := LintPack(packPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f) == 0 {
		t.Fatal("expected leak in pack")
	}
}

func TestLintExtraInfraFlags(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("server 203.0.113.9 internal\n"), 0o600)
	// Without the extra pattern, TEST-NET must not be flagged.
	if f, _ := LintPath(dir, nil); len(f) != 0 {
		t.Fatalf("TEST-NET flagged without pattern: %v", f)
	}
	f, err := LintPath(dir, []string{`203\.0\.113\.`})
	if err != nil {
		t.Fatal(err)
	}
	if len(f) == 0 {
		t.Fatal("extra infra pattern did not flag")
	}
}
