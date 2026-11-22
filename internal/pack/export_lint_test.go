package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStructureExportFailsOnLeak(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(prof, "skills", "demo"), 0o755)
	os.WriteFile(filepath.Join(prof, "skills", "demo", "SKILL.md"), []byte("token ghp_"+strings.Repeat("a", 36)), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	e := newTestExporter(t, home, out, DefaultExportOptions())
	e.Home = home
	if err := e.Export(); err == nil {
		t.Fatal("expected leak to block structure export")
	}
}

func TestCleanStructureExportPasses(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(prof, "memories"), 0o755)
	os.WriteFile(filepath.Join(prof, "memories", "conventions.md"), []byte("# conv"), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	e := newTestExporter(t, home, out, DefaultExportOptions())
	e.Home = home
	if err := e.Export(); err != nil {
		t.Fatalf("clean export should pass: %v", err)
	}
}
