package pack

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func tarFiles(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var out []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, h.Name)
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func newTestExporter(t *testing.T, home, out string, opts ExportOptions) *Exporter {
	t.Helper()
	man := NewManifest("personal", "hermes")
	san := NewSanitizer(DefaultSanitizeRules(), man)
	return NewExporterWithOptions("personal", "hermes", out, san, man, nil, opts)
}

func TestExportStructureOnlyOmitsPersonal(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(prof, "memories"), 0o755)
	os.WriteFile(filepath.Join(prof, "memories", "MEMORY.md"), []byte("# personal"), 0o600)
	os.WriteFile(filepath.Join(prof, "memories", "conventions.md"), []byte("# conv"), 0o600)
	os.MkdirAll(filepath.Join(home, "secrets-central"), 0o755)
	os.WriteFile(filepath.Join(home, "secrets-central", "a.env"), []byte("K=V"), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	e := newTestExporter(t, home, out, DefaultExportOptions())
	e.Home = home
	if err := e.Export(); err != nil {
		t.Fatal(err)
	}
	files := tarFiles(t, out)
	if contains(files, "memory/MEMORY.md") {
		t.Fatal("personal memory in structure pack")
	}
	if contains(files, "secrets/a.env") {
		t.Fatal("secrets in structure pack")
	}
	if !contains(files, "memory/conventions.md") {
		t.Fatal("methodology memory missing from structure")
	}
}

func TestExportWithMemory(t *testing.T) {
	home := t.TempDir()
	prof := filepath.Join(home, ".hermes", "profiles", "personal")
	os.MkdirAll(filepath.Join(prof, "memories"), 0o755)
	os.WriteFile(filepath.Join(prof, "memories", "MEMORY.md"), []byte("# personal"), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	opts := DefaultExportOptions()
	opts.Layers = append(opts.Layers, LayerMemory)
	e := newTestExporter(t, home, out, opts)
	e.Home = home
	if err := e.Export(); err != nil {
		t.Fatal(err)
	}
	if !contains(tarFiles(t, out), "memory/MEMORY.md") {
		t.Fatal("memory layer did not include MEMORY.md")
	}
}

func TestExportWithSecrets(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "secrets-central"), 0o755)
	os.WriteFile(filepath.Join(home, "secrets-central", "a.env"), []byte("K=V"), 0o600)
	out := filepath.Join(t.TempDir(), "p.tar.gz")
	opts := DefaultExportOptions()
	opts.Layers = append(opts.Layers, LayerSecrets)
	e := newTestExporter(t, home, out, opts)
	e.Home = home
	if err := e.Export(); err != nil {
		t.Fatal(err)
	}
	if !contains(tarFiles(t, out), "secrets/a.env") {
		t.Fatal("secrets layer did not include a.env")
	}
}
